package doriath

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/rensa-labs/doriath/internal/libkataware"
	"github.com/rensa-labs/doriath/internal/sqliteforest"
	"github.com/rensa-labs/doriath/operlog"
)

// Client represents a Bitforest client.
type Client struct {
	GenTx    []byte
	NaURL    *url.URL
	CacheDir string
}

type tx struct {
	RawTx    []byte
	BlockIdx int
	PosInBlk int
	Merkle   [][]byte
}

type opEntry struct {
	RawOps []byte
	Proof  [][]byte
}

// TxChainFileName is a name of a cache file storing TxChain.
const TxChainFileName = "txchain.json"

// BlockHeaderFileName is a name of a cache file storing blockchain headers.
const BlockHeaderFileName = "blockchain_headers"

// ErrOutOfSync is an error object used when the cache is out of sync.
var ErrOutOfSync = errors.New("cache out of sync")

// ErrInvTxChain is an error object used when the provided TxChain is invalid.
var ErrInvTxChain = errors.New("invalid TxChain")

// ErrInvBlockchainHeaders is an error object used when the provided blockchain headers are invalid.
var ErrInvBlockchainHeaders = errors.New("invalid blockchain headers")

// ErrInvOpEntries is an error object for invalid operation entries.
var ErrInvOpEntries = errors.New("invalid operation entries")

// Sync downloads transaction chains and blockchain headers.
func (clnt *Client) Sync() error {
	os.MkdirAll(clnt.CacheDir, 0777)
	log.Println("sync starting")
	// store txChain and headers
	err := clnt.downloadBlockchainHeaders()
	if err != nil {
		return err
	}
	log.Println("downloaded blockchain headers")
	err = clnt.downloadTxChain()
	if err != nil {
		return err
	}
	log.Println("downloaded txchain")

	if !clnt.checkBlockchainHeaders() {
		return ErrInvBlockchainHeaders
	}
	log.Println("checked blockchain headers")

	txChain, _ := clnt.getTxChain()
	if !clnt.checkTxChain(txChain) {
		return ErrInvTxChain
	}
	log.Println("checked txchain")

	return nil
}

// GetOpLog returns an OperLog containing all operations regarding
// the provided name and the number of confirmed operations in
// the returning OperLog object.
func (clnt *Client) GetOpLog(name string) (operlog.OperLog, int, error) {
	fd, err := os.Open(clnt.CacheDir + BlockHeaderFileName)
	if err != nil {
		panic(err)
	}
	defer fd.Close()
	resp, err := clnt.getHTTPClient().Get(clnt.NaURL.String() + "/oplogs/" + name + ".json")
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	var opEntries []opEntry
	err = decoder.Decode(&opEntries)
	if err != nil {
		return nil, 0, err
	}

	txChain, err := clnt.getTxChain()
	if err != nil {
		return nil, 0, err
	}

	// check whether the cached data is in sync
	if len(txChain) < len(opEntries) {
		log.Println("txchain less than opentries", len(txChain), len(opEntries))
		return nil, 0, ErrOutOfSync
	}

	// first pass: ensure it goes confirmed->unconfirmed->(staging) using a simple SM
	cflag := 'c'
	ccount := 0
	for i, ope := range opEntries {
		txce := txChain[i]
		if ope.Proof == nil {
			if cflag == 'c' || cflag == 'u' {
				cflag = 's'
				continue
			} else {
				log.Println("state machine fail on null proof")
				return nil, 0, ErrInvOpEntries
			}
		}
		if txce.BlockIdx < 0 {
			if cflag == 'c' || cflag == 'u' {
				cflag = 'u'
				continue
			} else {
				log.Println("state machine fail on lack of confirm")
				return nil, 0, ErrInvOpEntries
			}
		}
		if cflag == 'c' {
			cflag = 'c'
			ccount++
			continue
		} else {
			log.Println("state machine fail on steady")
			return nil, 0, ErrInvOpEntries
		}
	}

	var toret operlog.OperLog

	// second pass: check all proofs
	for i, ope := range opEntries {
		txce := txChain[i]
		var tx libkataware.Transaction
		tx.FromBytes(txce.RawTx)
		if txce.BlockIdx >= 0 {
			var prf sqliteforest.Proof
			roothash := tx.Outputs[1].Script[3:23]
			for _, b := range ope.Proof {
				var lol sqliteforest.AbbrNode
				lol.FromBytes(b)
				prf = append(prf, lol)
			}
			valHash := libkataware.DoubleSHA256(ope.RawOps)
			if ope.RawOps == nil {
				valHash = nil
			}
			if ope.Proof != nil && !prf.Check(roothash, name, valHash) {
				log.Println("proof check failed")
				return nil, 0, ErrInvOpEntries
			}
			buf := bytes.NewReader(ope.RawOps)
			for buf.Len() != 0 {
				var op operlog.Operation
				err := op.Unpack(buf)
				if err != nil {
					log.Println("unpack failed")
					return nil, 0, ErrInvOpEntries
				}
				toret = append(toret, op)
			}
		}
	}

	if !toret.IsValid() {
		return nil, 0, ErrInvOpEntries
	}

	return toret, ccount, nil
}

func (clnt *Client) idx2hdr(fd *os.File, idx int) libkataware.Header {
	fd.Seek(int64(idx*libkataware.HeaderLen), 0)
	buf := make([]byte, 80)
	_, e := io.ReadFull(fd, buf)
	if e != nil {
		panic(e)
	}
	var toret libkataware.Header
	toret.Deserialize(buf)
	return toret
}

func (clnt *Client) checkTxChain(txChain []tx) bool {
	fd, err := os.Open(clnt.CacheDir + BlockHeaderFileName)
	if err != nil {
		panic(err)
	}
	defer fd.Close()
	// TODO GenTx?
	// forward checking ==>
	for i := 0; i < len(txChain); i++ {
		txi := txChain[i]
		btx := libkataware.Transaction{}
		btx.FromBytes(txi.RawTx)
		// check that the block spends the previous one
		if i > 0 {
			btxlast := libkataware.Transaction{}
			btxlast.FromBytes(txChain[i-1].RawTx)
			if len(btx.Inputs) == 0 ||
				bytes.Compare(btx.Inputs[0].PrevHash, btxlast.Hash256()) != 0 {
				return false
			}
		}
		// check blockchain inclusion for confirmed chains only
		if txi.BlockIdx >= 0 {
			hdx := clnt.idx2hdr(fd, i)
			if !hdx.CheckMerkle(txi.Merkle, txi.BlockIdx, btx) {
				return false
			}
		}
	}
	return true
}

func (clnt *Client) checkBlockchainHeaders() bool {
	fd, err := os.Open(clnt.CacheDir + BlockHeaderFileName)
	if err != nil {
		panic(err)
	}
	defer fd.Close()

	stat, err := fd.Stat()
	if err != nil {
		panic(err)
	}
	// backward checking <==
	for i := 1; i < int(stat.Size()/80); i++ {
		hi := clnt.idx2hdr(fd, i)
		hlast := clnt.idx2hdr(fd, i-1)
		if subtle.ConstantTimeCompare(hi.HashPrevBlock,
			libkataware.DoubleSHA256(hlast.Serialize())) == 0 {
			return false
		}
	}
	return true
}

func (clnt *Client) getHTTPClient() *http.Client {
	tr := &http.Transport{
		DisableKeepAlives: false,
	}
	client := &http.Client{Transport: tr}
	return client
}

func (clnt *Client) downloadBlockchainHeaders() error {
	resp, err := clnt.getHTTPClient().Get(clnt.NaURL.String() + "/blockchain_headers")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// remove?
	f, err := os.Create(clnt.CacheDir + "/" + BlockHeaderFileName)
	if err != nil {
		return nil
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return err
	}
	f.Sync()

	return nil
}

func (clnt *Client) downloadTxChain() error {
	resp, err := clnt.getHTTPClient().Get(clnt.NaURL.String() + "/txchain.json")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// remove beforehand?
	f, err := os.Create(clnt.CacheDir + "/" + TxChainFileName)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return err
	}
	f.Sync()

	return nil
}

func (clnt *Client) getBlockchainHeaders() ([]libkataware.Header, error) {
	data, err := ioutil.ReadFile(clnt.CacheDir + "/" + BlockHeaderFileName)
	if err != nil {
		return nil, err
	}

	headers := make([]libkataware.Header, 0)

	headerLen := libkataware.HeaderLen
	numHeader := len(data) / headerLen

	var hdr libkataware.Header
	for i := 0; i < numHeader; i++ {
		hdr = libkataware.Header{}
		hdr.Deserialize(data[i*headerLen : (i+1)*headerLen])
		headers = append(headers, hdr)
	}
	return headers, nil
}

func (clnt *Client) getTxChain() ([]tx, error) {
	data, err := ioutil.ReadFile(clnt.CacheDir + "/" + TxChainFileName)
	if err != nil {
		return nil, err
	}

	txChain := make([]tx, 0)
	err = json.Unmarshal(data, &txChain)
	if err != nil {
		return nil, err
	}

	return txChain, nil
}
