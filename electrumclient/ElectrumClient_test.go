package electrumclient

import (
	"encoding/hex"
	"fmt"
	"log"
	"testing"

	"github.com/rensa-labs/doriath/internal/libkataware"
)

func TestBasic(t *testing.T) {
	ec := NewElectrumClient("electrum.coinucopia.io:50001")
	cnt, err := ec.GetBlockCount()
	if err != nil {
		panic(err)
	}
	log.Println("Block count:", cnt)
	return
}

func TestLocate(t *testing.T) {
	ec := NewElectrumClient("electrum.coinucopia.io:50001")
	bts, _ := hex.DecodeString("7f4c1620eb31d3b09aacdfa606cd43ca51f1d04cedb19f65b9206bfefe71ffb5")
	blkid, err := ec.LocateTx(libkataware.SwapBytes(bts))
	if err != nil {
		panic(err)
	}
	log.Println("located at", blkid)
}

func TestMerkle(t *testing.T) {
	ec := NewElectrumClient("electrum.coinucopia.io:50001")
	bts, _ := hex.DecodeString("7f4c1620eb31d3b09aacdfa606cd43ca51f1d04cedb19f65b9206bfefe71ffb5")
	blkid, err := ec.LocateTx(libkataware.SwapBytes(bts))
	if err != nil {
		panic(err)
	}
	mk, _, err := ec.GetMerkle(libkataware.SwapBytes(bts), blkid)
	if err != nil {
		panic(err)
	}
	for _, v := range mk {
		log.Printf("%x\n", v)
	}
}

func TestEstimateFee(t *testing.T) {
	ec := NewElectrumClient("electrum.coinucopia.io:50001")
	log.Println("recommended fee", ec.EstimateFee(25))
}

func TestSignTx(t *testing.T) {
	ec := NewElectrumClient("electrum.coinucopia.io:50001")
	prevTxBts, _ := hex.DecodeString("0100000001ab6030f026696111d8a802e19f8e255b64e252e693996ee7678bd3e362e76050000000006a4730440220768e885ce1eef6b39fcbc2cfe763bedc5d4d46d43320a2e962b8d841779f3bf302202ed8a564400e6cf3e26cf83a50f9424727cdf454aea7a7949521aa72aedd266e012102b6a7bc5b8e36fa39283105bea7318efa54eafc4716d696c98b221fd9fb2657b0ffffffff023c8c0000000000001976a914a841a327af514a03b961b94cdcaf0f5d1242557a88ac50c30000000000001976a914294a46fc929ee04f0b077c954af3aad0154ede5288ac00000000")
	var ptx libkataware.Transaction
	ptx.FromBytes(prevTxBts)
	feeAmt := uint64(ec.EstimateFee(25) * 200)
	// TX just eats up money
	tx := libkataware.Transaction{
		Version: 1,
		Inputs: []libkataware.TxInput{
			libkataware.TxInput{
				PrevHash: ptx.Hash256(),
				PrevIdx:  1,
				Script:   ptx.Outputs[1].Script, // will be overwritten
				Seqno:    0xffffffff,
			},
		},
		Outputs: []libkataware.TxOutput{
			libkataware.TxOutput{
				Value:  ptx.Outputs[1].Value - feeAmt,
				Script: ptx.Outputs[1].Script,
			},
		},
	}
	// sign TX
	signed, err := ec.SignTx(tx.ToBytes(), "Kyr7e45WdaeJhJEpBhCadcb4CgJDNkkFhjVy6kkTKXiGstN9YDAs")
	if err != nil {
		panic(err)
	}
	fmt.Printf("%x\n", signed)
}
