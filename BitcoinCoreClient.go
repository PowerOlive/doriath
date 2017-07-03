package doriath

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/rensa-labs/doriath/internal/libkataware"
)

// BitcoinCoreClient is an implementation of BitcoinClient that wraps around Bitcoin Core.
type BitcoinCoreClient struct {
	rpcAddr string
	rpcUser string
	rpcPwd  string

	hclient *http.Client
}

func (bcc *BitcoinCoreClient) callMethod(mname string,
	params ...interface{}) (map[string]json.RawMessage, error) {
	parms, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	tosend := fmt.Sprintf(`{"jsonrpc": "1.0", "id":%v, "method": "%v", "params": %v }`,
		rand.Int(), mname, string(parms))
	req, err := http.NewRequest("POST", bcc.rpcAddr, bytes.NewReader([]byte(tosend)))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(bcc.rpcUser, bcc.rpcPwd)
	resp, err := bcc.hclient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("not OK %v", resp.StatusCode)
	}
	var jsresp map[string]json.RawMessage
	err = json.NewDecoder(resp.Body).Decode(&jsresp)
	if err != nil {
		return nil, err
	}
	return jsresp, nil
}

// GetBlockCount obtains the total number of blocks in the canonical blockchain.
func (bcc *BitcoinCoreClient) GetBlockCount() (bcount int, err error) {
	jsresp, err := bcc.callMethod("getinfo")
	if err != nil {
		return
	}
	err = json.Unmarshal(jsresp["result"], &jsresp)
	if err != nil {
		return
	}
	err = json.Unmarshal(jsresp["blocks"], &bcount)
	return
}

// GetBlockHash takes a number representing an index in the canonical blockchain, and returns a 32-byte block hash in standard order.
func (bcc *BitcoinCoreClient) GetBlockHash(idx int) (hsh []byte, err error) {
	jsresp, err := bcc.callMethod("getblockhash", idx)
	if err != nil {
		return
	}
	var hexstring string
	err = json.Unmarshal(jsresp["result"], &hexstring)
	if err != nil {
		return
	}
	hsh, err = hex.DecodeString(hexstring)
	if err != nil {
		return
	}
	hsh = libkataware.SwapBytes(hsh)
	return
}

// GetBlockIdx takes in a 32-byte block hash in standard order, and returns the index of the corresponding block in the canonical blockchain.
func (bcc *BitcoinCoreClient) GetBlockIdx(hsh []byte) (idx int, err error) {
	jsresp, err := bcc.callMethod("getblock", hex.EncodeToString(libkataware.SwapBytes(hsh)), true)
	if err != nil {
		return
	}
	err = json.Unmarshal(jsresp["result"], &jsresp)
	if err != nil {
		return
	}
	err = json.Unmarshal(jsresp["height"], &idx)
	return
}

// GetBlock takes in a 32-byte block hash in standard order, and returns the entire block as a byte array.
func (bcc *BitcoinCoreClient) GetBlock(idx int) (blk []byte, err error) {
	hsh, err := bcc.GetBlockHash(idx)
	if err != nil {
		return
	}
	jsresp, err := bcc.callMethod("getblock", hex.EncodeToString(libkataware.SwapBytes(hsh)), false)
	if err != nil {
		return
	}
	var hexstr string
	err = json.Unmarshal(jsresp["result"], &hexstr)
	if err != nil {
		return
	}
	blk, err = hex.DecodeString(hexstr)
	return
}

// GetHeader takes in a block index and returns the 80-byte block header.
func (bcc *BitcoinCoreClient) GetHeader(idx int) (hdr []byte, err error) {
	hsh, err := bcc.GetBlockHash(idx)
	if err != nil {
		return
	}
	jsresp, err := bcc.callMethod("getblockheader", hex.EncodeToString(libkataware.SwapBytes(hsh)), false)
	if err != nil {
		return
	}
	var hexstr string
	err = json.Unmarshal(jsresp["result"], &hexstr)
	if err != nil {
		return
	}
	hdr, err = hex.DecodeString(hexstr)
	return
}

// LocateTx returns the hash of the block containing the given transaction hash.
func (bcc *BitcoinCoreClient) LocateTx(txhsh []byte) (idx int, err error) {
	jsresp, err := bcc.callMethod("getrawtransaction",
		hex.EncodeToString(libkataware.SwapBytes(txhsh)), true)
	if err != nil {
		return
	}
	err = json.Unmarshal(jsresp["result"], &jsresp)
	if err != nil {
		return
	}
	var hexstr string
	err = json.Unmarshal(jsresp["blockhash"], &hexstr)
	if err != nil {
		return
	}
	hdhsh, err := hex.DecodeString(hexstr)
	if err != nil {
		return
	}
	hdhsh = libkataware.SwapBytes(hdhsh)
	idx, err = bcc.GetBlockIdx(hdhsh)
	return
}

// SignTx signs a transaction, returning the signed version.
func (bcc *BitcoinCoreClient) SignTx(tx []byte, skWIF string) (stx []byte, err error) {
	jsresp, err := bcc.callMethod("signrawtransaction",
		hex.EncodeToString(tx), nil, []string{skWIF})
	if err != nil {
		return
	}
	err = json.Unmarshal(jsresp["result"], &jsresp)
	if err != nil {
		return
	}
	var hexstr string
	err = json.Unmarshal(jsresp["hex"], &hexstr)
	if err != nil {
		return
	}
	stx, err = hex.DecodeString(hexstr)
	return
}

// BroadcastTx broadcasts a transaction.
func (bcc *BitcoinCoreClient) BroadcastTx(tx []byte) (err error) {
	_, err = bcc.callMethod("sendrawtransaction", hex.EncodeToString(tx))
	return
}

// NewBitcoinCoreClient creates a new BitcoinCoreClient.
func NewBitcoinCoreClient(addr string, user string, pwd string) *BitcoinCoreClient {
	return &BitcoinCoreClient{
		rpcAddr: fmt.Sprintf("http://%v/", addr),
		rpcUser: user,
		rpcPwd:  pwd,
		hclient: &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:        50,
				MaxIdleConnsPerHost: 32,
				IdleConnTimeout:     time.Second * 120,
			},
			Timeout: time.Second * 200,
		},
	}
}
