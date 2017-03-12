package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"log"
	"os"
	"strconv"

	"github.com/rensa-labs/doriath/internal/libkataware"
)

func pretty(v interface{}) string {
	lol, err := json.MarshalIndent(v, "", "    ")
	if err != nil {
		panic(err.Error())
	}
	return string(lol)
}

func main() {
	toRdm, err := hex.DecodeString(os.Args[1])
	if err != nil {
		log.Fatalln(err)
	}
	var origTx libkataware.Transaction
	origTx.Unpack(bytes.NewReader(toRdm))
	rdIdx, err := strconv.Atoi(os.Args[2])
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("wanna redeem %v,%v\n", pretty(origTx), rdIdx)
	toSign := libkataware.Transaction{
		Version: 1,
		Inputs: []libkataware.TxInput{
			libkataware.TxInput{
				PrevHash: origTx.Hash256(),
				PrevIdx:  rdIdx,
				Script:   origTx.Outputs[rdIdx].Script,
				Seqno:    0xffffffff,
			},
		},
		Outputs: []libkataware.TxOutput{
			libkataware.TxOutput{
				Value:  490000,
				Script: origTx.Outputs[rdIdx].Script,
			},
		},
	}
	buf := new(bytes.Buffer)
	toSign.Pack(buf)
	log.Printf("toSign is %x\n", buf.Bytes())
	log.Println(pretty(toSign))
}
