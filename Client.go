package doriath

import (
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
const BlockHeaderFileName = "block_headers"

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
	// store txChain and headers
	err := clnt.downloadBlockchainHeaders()
	if err != nil {
		return err
	}

	err = clnt.downloadTxChain()
	if err != nil {
		return err
	}

	headers, _ := clnt.getBlockchainHeaders()
	if !checkBlockchainHeaders(headers) {
		return ErrInvBlockchainHeaders
	}

	txChain, _ := clnt.getTxChain()
	if !checkTxChain(txChain, headers) {
		return ErrInvTxChain
	}

	return nil
}

// GetOpLog returns an OperLog containing all operations regarding
// the provided name and the number of confirmed operations in
// the returning OperLog object.
func (clnt *Client) GetOpLog(name string) (operlog.OperLog, int, error) {
	resp, err := getHttpClient().Get(clnt.NaURL.String() + "/oplogs/" + name + ".json")
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	opEntries := make([]opEntry, 0)
	err = json.Unmarshal(body, &opEntries)
	if err != nil {
		return nil, 0, err
	}

	lenOpEntries := len(opEntries)
	lastOpEntry := opEntries[lenOpEntries-1]
	// TODO find a null proof indicating unstaged operations
	for i := 0; i < len(lastOpEntry.Proof); i++ {
		if lastOpEntry.Proof[i] == nil {
			lenOpEntries--
			break
		}
	}

	txChain, err := clnt.getTxChain()
	if err != nil {
		return nil, 0, err
	}

	lenTxChain := len(txChain)
	if txChain[lenTxChain-1].BlockIdx < 0 {
		lenTxChain--
	}

	log.Printf("%d op_entries\n", lenOpEntries)
	log.Printf("%d tx_chain\n", lenTxChain)
	// check whether the cached data is in sync
	if lenTxChain < lenOpEntries {
		return nil, 0, ErrOutOfSync
	}

	toret := operlog.OperLog{}
	cnt := 0
	for i := 0; i < lenOpEntries; i++ {
		oe := opEntries[i]

		var valHash []byte
		valHash = nil
		proof := sqliteforest.Proof{}
		for j := 0; j < len(oe.Proof); j++ {
			anode := sqliteforest.AbbrNode{}
			anode.FromBytes(oe.Proof[j])
			proof = append(proof, anode)

			if anode.Key == name {
				valHash = anode.VHash
			}
		}

		btx := libkataware.Transaction{}
		btx.FromBytes(txChain[0].RawTx)
		rootHash := btx.Outputs[1].Script[3:23]

		if !proof.Check(rootHash, name, valHash) {
			return nil, 0, ErrInvOpEntries
		}

		cnt++
		op := operlog.Operation{}
		op.FromBytes(oe.RawOps)
		toret = append(toret, op)
	}

	for i := lenOpEntries; i < len(opEntries); i++ {
		oe := opEntries[i]
		op := operlog.Operation{}
		op.FromBytes(oe.RawOps)
		toret = append(toret, op)
	}

	return toret, cnt, nil
}

func checkTxChain(txChain []tx, headers []libkataware.Header) bool {
	// TODO GenTx?
	// forward checking ==>
	for i := 0; i < len(txChain); i++ {
		txi := txChain[i]
		btx := libkataware.Transaction{}
		btx.FromBytes(txi.RawTx)
		// check staged chains only
		if txi.BlockIdx >= 0 && !headers[txi.BlockIdx].CheckMerkle(txi.Merkle, txi.PosInBlk, btx) {
			return false
		}
	}
	return true
}

func checkBlockchainHeaders(headers []libkataware.Header) bool {
	// TODO GenTx?
	// backward checking <==
	for i := len(headers) - 1; i > 0; i-- {
		if subtle.ConstantTimeCompare(headers[i].HashPrevBlock, libkataware.DoubleSHA256(headers[i-1].Serialize())) == 0 {
			return false
		}
	}
	return true
}

func getHttpClient() *http.Client {
	tr := &http.Transport{
		DisableKeepAlives: false,
	}
	client := &http.Client{Transport: tr}
	return client
}

func (clnt *Client) downloadBlockchainHeaders() error {
	resp, err := getHttpClient().Get(clnt.NaURL.String() + "/blockchain_headers")
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
	resp, err := getHttpClient().Get(clnt.NaURL.String() + "/txchain.json")
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
