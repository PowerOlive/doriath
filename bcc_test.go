// +build debug

package doriath

import (
	"fmt"
	"testing"

	"github.com/rensa-labs/doriath/internal/libkataware"
)

func TestBCC(t *testing.T) {
	bcc := NewBitcoinCoreClient("localhost:8332", "user", "pwd")
	bcount, err := bcc.GetBlockCount()
	if err != nil {
		panic(err)
	}
	fmt.Printf("GetBlockCount() = %v\n", bcount)
	bhash, err := bcc.GetBlockHash(bcount)
	if err != nil {
		panic(err)
	}
	fmt.Printf("GetBlockHash(%v) = %x\n", bcount, bhash[:10])
	bcount2, err := bcc.GetBlockIdx(bhash)
	if err != nil {
		panic(err)
	}
	fmt.Printf("GetBlockIdx(%x) = %v\n", bhash[:10], bcount2)
	blk, err := bcc.GetBlock(bhash)
	if err != nil {
		panic(err)
	}
	var lkblk libkataware.Block
	lkblk.Deserialize(blk)
	fmt.Printf("GetBlock(%x) = (%v with %v transactions)\n", bhash[:10],
		lkblk.Hdr.Time, len(lkblk.Bdy))
	blkhdr, err := bcc.GetHeader(bhash)
	if err != nil {
		panic(err)
	}
	var lkblkhdr libkataware.Header
	lkblkhdr.Deserialize(blkhdr)
	fmt.Printf("GetBlockHeader(%x) = (%v with %x)\n", bhash[:10],
		lkblkhdr.Time, libkataware.DoubleSHA256(lkblkhdr.Serialize())[:10])
	for _, tx := range lkblk.Bdy {
		bhsh, err := bcc.LocateTx(tx.Hash256())
		if err != nil {
			panic(err)
		}
		fmt.Printf("LocateTx(%x) = %x\n", tx.Hash256()[:10], bhsh[:10])
	}
}
