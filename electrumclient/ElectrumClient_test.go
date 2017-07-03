package electrumclient

import (
	"log"
	"testing"
)

func TestBasic(t *testing.T) {
	ec := &ElectrumClient{
		Host: "electrum.no-ip.org:50001",
	}
	cnt, err := ec.GetBlockCount()
	if err != nil {
		panic(err)
	}
	log.Println("Block count:", cnt)
	blk, err := ec.GetBlock(cnt - 1)
	if err != nil {
		panic(err)
	}
	log.Println("Actual block:", blk)
}
