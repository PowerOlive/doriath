package electrumclient

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/rensa-labs/curve"
	"github.com/rensa-labs/doriath/internal/libkataware"
)

// ElectrumClient represents a BitcoinClient using Electrum.
type ElectrumClient struct {
	servHost string
	hdrCache map[int][]byte
	hcLock   sync.Mutex

	httpClient *http.Client
}

// NewElectrumClient creates an Electrum client that connects to the given server.
func NewElectrumClient(servHost string) *ElectrumClient {
	return &ElectrumClient{
		servHost: servHost,
		hdrCache: make(map[int][]byte),
		httpClient: &http.Client{
			Transport: &http.Transport{
				DisableKeepAlives: true,
			},
		},
	}
}

func (ec *ElectrumClient) callMethod(mname string,
	params ...interface{}) (map[string]json.RawMessage, error) {
	if params == nil {
		params = make([]interface{}, 0)
	}
	parms, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	tosend := fmt.Sprintf(`{"jsonrpc": "2.0", "id":%v, "method": "%v", "params": %v }`,
		rand.Int()%10000, mname, string(parms))
	conn, err := net.DialTimeout("tcp", ec.servHost, time.Second*10)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	fmt.Fprintln(conn, tosend)
	rdr := bufio.NewReader(conn)
	respstr, err := rdr.ReadString('\n')
	if err != nil {
		return nil, err
	}
	var jsresp map[string]json.RawMessage
	err = json.NewDecoder(bytes.NewReader([]byte(respstr))).Decode(&jsresp)
	if err != nil {
		return nil, err
	}
	return jsresp, nil
}

// GetBlockCount obtains the total number of blocks in the canonical chain.
func (ec *ElectrumClient) GetBlockCount() (cnt int, err error) {
	jsr, err := ec.callMethod("blockchain.numblocks.subscribe")
	if err != nil {
		return
	}
	err = json.Unmarshal(jsr["result"], &cnt)
	return
}

// GetHeader obtains the block header given the index.
func (ec *ElectrumClient) GetHeader(idx int) (blk []byte, err error) {
	// read ahead
	chunkIdx := idx / 2016
	chunkOffset := (idx % 2016) * 80
	ec.hcLock.Lock()
	chunk := ec.hdrCache[chunkIdx]
	ec.hcLock.Unlock()
	if chunk == nil {
		var jsr map[string]json.RawMessage
		jsr, err = ec.callMethod("blockchain.block.get_chunk", chunkIdx)
		log.Println("returned")
		if err != nil {
			return
		}
		var hexec string
		err = json.Unmarshal(jsr["result"], &hexec)
		if err != nil {
			return
		}
		chunk, err = hex.DecodeString(hexec)
		if err != nil {
			return
		}
		ec.hcLock.Lock()
		ec.hdrCache[chunkIdx] = chunk
		ec.hcLock.Unlock()
	}
	if len(chunk) <= chunkOffset {
		err = errors.New("too new")
		return
	}
	blk = chunk[chunkOffset:][:80]
	return
}

// LocateTx locates in which block a transaction exists.
// TODO don't use blockchain.info
func (ec *ElectrumClient) LocateTx(txhsh []byte) (idx int, err error) {
	hresp, err := ec.httpClient.Get(fmt.Sprintf("https://blockchain.info/rawtx/%x",
		libkataware.SwapBytes(txhsh)))
	if err != nil {
		return
	}
	defer hresp.Body.Close()
	resp := make(map[string]json.RawMessage)
	err = json.NewDecoder(hresp.Body).Decode(&resp)
	if err != nil {
		return
	}
	err = json.Unmarshal(resp["block_height"], &idx)
	return
}

// GetMerkle returns merkle.
func (ec *ElectrumClient) GetMerkle(txhsh []byte, blkidx int) (mk [][]byte, ps int, err error) {
	jsr, err := ec.callMethod("blockchain.transaction.get_merkle",
		hex.EncodeToString(libkataware.SwapBytes(txhsh)), blkidx)
	if err != nil {
		return
	}
	var resdict map[string]json.RawMessage
	var mkhexes []string
	err = json.Unmarshal(jsr["result"], &resdict)
	if err != nil {
		return
	}
	err = json.Unmarshal(resdict["merkle"], &mkhexes)
	if err != nil {
		return
	}
	for _, v := range mkhexes {
		var mkb []byte
		mkb, err = hex.DecodeString(v)
		if err != nil {
			return
		}
		mk = append(mk, mkb)
	}
	err = json.Unmarshal(resdict["pos"], &ps)
	return
}

// EstimateFee estimates fees in satoshis per byte.
func (ec *ElectrumClient) EstimateFee(within int) uint64 {
	const satoshisPerBtc = 100000000
	jsr, err := ec.callMethod("blockchain.estimatefee", within)
	if err != nil {
		return 100
	}
	btcpkb := float64(0.0)
	err = json.Unmarshal(jsr["result"], &btcpkb)
	if err != nil {
		log.Println(err, jsr)
		return 100
	}
	toret := uint64(btcpkb*satoshisPerBtc) / 1000
	if toret > 200 {
		toret = 200
	}
	if toret < 10 {
		toret = 10
	}
	return toret
}

// SignTx signs a transaction, given a private key in WIF format.
func (ec *ElectrumClient) SignTx(tx []byte, skWIF string) (signed []byte, err error) {
	var secKey curve.PrivateKey
	err = secKey.FromWIF(skWIF)
	if err != nil {
		return
	}
	var parsedTx libkataware.Transaction
	err = parsedTx.FromBytes(tx)
	if err != nil {
		return
	}
	for i := 0; i < len(parsedTx.Inputs); i++ {
		// create fake TX
		fakeTx := parsedTx
		fakeTx.Inputs = make([]libkataware.TxInput, len(parsedTx.Inputs))
		copy(fakeTx.Inputs, parsedTx.Inputs)
		for j := 0; j < len(parsedTx.Inputs); j++ {
			if j != i {
				fakeTx.Inputs[j].Script = nil
			}
		}
		// Step 1: append hash code type
		toSign := append(fakeTx.ToBytes(), []byte{1, 0, 0, 0}...)
		// Step 2: double SHA256
		toSignHash := libkataware.DoubleSHA256(toSign)
		// Step 3: create signature
		signat, e := secKey.Sign(toSignHash)
		if e != nil {
			err = e
			return
		}
		// Step 4: create scriptSig
		der := append(signat.Serialize(), 0x01)
		scriptSig := append([]byte{byte(len(der))}, der...)
		pubkBytes := secKey.PubKey().ToBytes()
		scriptSig = append(scriptSig,
			append([]byte{byte(len(pubkBytes))}, pubkBytes...)...)
		// Step 5: replace temporary scriptSig
		parsedTx.Inputs[i].Script = scriptSig
	}
	// Step 6: reserialize
	signed = parsedTx.ToBytes()
	return
}

// BroadcastTx broadcasts a transaction to the blockchain.
func (ec *ElectrumClient) BroadcastTx(tx []byte) (err error) {
	jsr, err := ec.callMethod("blockchain.transaction.broadcast", hex.EncodeToString(tx))
	if err != nil {
		return err
	}
	var resp string
	err = json.Unmarshal(jsr["result"], &resp)
	if err != nil {
		return err
	}
	_, e := hex.DecodeString(resp)
	if e != nil {
		err = errors.New(resp)
	}
	return
}
