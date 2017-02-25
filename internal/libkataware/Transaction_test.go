package libkataware

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"testing"
	"time"
)

func randTx() Transaction {
	numIn := rand.Int() % 10
	numOut := rand.Int() % 10
	var toret Transaction
	toret.Version = 0x01
	toret.Inputs = make([]TxInput, numIn)
	toret.Outputs = make([]TxOutput, numOut)
	for i := range toret.Inputs {
		toret.Inputs[i].PrevHash = make([]byte, 32)
		rand.Read(toret.Inputs[i].PrevHash)
		toret.Inputs[i].PrevIdx = rand.Int() % 10
		toret.Inputs[i].Script = make([]byte, 32)
		rand.Read(toret.Inputs[i].Script)
	}
	for i := range toret.Outputs {
		toret.Outputs[i].Value = uint64(rand.Int() % 100000)
		toret.Outputs[i].Script = make([]byte, 32)
		rand.Read(toret.Outputs[i].Script)
	}
	return toret
}

func TestTxRandPack(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	tx := randTx()
	txJSON, _ := json.MarshalIndent(&tx, "", "    ")
	buf := new(bytes.Buffer)
	tx.Pack(buf)
	var ntx Transaction
	ntx.Unpack(buf)
	ntxJSON, _ := json.MarshalIndent(&ntx, "", "    ")
	if string(txJSON) != string(ntxJSON) {
		log.Println(string(txJSON))
		log.Println(string(ntxJSON))
		t.Error("tx not what it originally was")
	}
}

func TestRealTx(t *testing.T) {
	rawtx, _ := base64.StdEncoding.DecodeString(
		"AQAAAAEBgg4haRMad5ds8gTOKGheSabSJ4hhwztiQbo64+CknwIAAACLSDBFAiEAmKKFFCD" +
			"k2rplb9ectgy1Zb1yGLaxF/2ppRL/vxf48XgCIAXGHzH+884/kG62cuBbZfUGBFplqAQxte" +
			"ryjgmZJmmTAUEE8PhvpXxCTesWDQ/HaT8T/OXtZULClIPFGVPk+ofr8kdIftebHdzz3maxg" +
			"iF/yvP87z/LRHN+uTsfy4kn6+zqJv////8CgFzXBQAAAAAZdqkUKdajVArPoKlQvvK/3HXN" +
			"UcJDkP2IrICEHgAAAAAAGXapFBe1A4pBP1xe4ojKpkz6s1oMAZFOiKwAAAAA")
	buf := bytes.NewBuffer(rawtx)
	var tx Transaction
	tx.Unpack(buf)
	if fmt.Sprintf("%x", tx.Hash256()) !=
		"daf0e3b16dc84af1804bd72c9e0466ac8a41bcd6fcffda042e0edf031d99f6b6" {
		t.Error("mismatched hash")
	}
}
