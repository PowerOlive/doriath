package electrumclient

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"time"
)

// ElectrumClient represents a BitcoinClient using Electrum.
type ElectrumClient struct {
	Host string
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
	log.Println(tosend)
	conn, err := net.DialTimeout("tcp", ec.Host, time.Second*10)
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

// GetBlock obtains the block given the index.
func (ec *ElectrumClient) GetBlock(idx int) (blk []byte, err error) {
	jsr, err := ec.callMethod("blockchain.block.get_chunk", idx)
	if err != nil {
		return
	}
	var hexec string
	err = json.Unmarshal(jsr["result"], &hexec)
	if err != nil {
		return
	}
	blk, err = hex.DecodeString(hexec)
	return
}
