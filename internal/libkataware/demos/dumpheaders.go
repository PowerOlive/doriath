package main

import (
	"crypto/subtle"
	"fmt"
	"io"
	"os"

	"github.com/rensa-labs/doriath/internal/libkataware"
)

func main() {
	hresp, err := os.Open(os.Args[1])
	if err != nil {
		panic(err.Error())
	}
	defer hresp.Close()
	buf := make([]byte, libkataware.HeaderLen)
	pre := make([]byte, libkataware.HeaderLen)
	for i := 0; ; i++ {
		var hdr libkataware.Header
		_, err := io.ReadFull(hresp, buf)
		if err != nil {
			if err == io.EOF {
				return
			}
			panic(err.Error())
		}
		hdr.Deserialize(buf)
		if i > 0 {
			if subtle.ConstantTimeCompare(hdr.HashPrevBlock, libkataware.DoubleSHA256(pre)) == 1 {
				fmt.Printf("Verified: %x (%v, %v, %v)\n", hdr.HashPrevBlock,
					hdr.Version, hdr.Bits, hdr.Time)
			} else {
				panic("bad hash")
			}
		}
		/*fmt.Printf("Version: %v\n", hdr.Version)
		fmt.Printf("HashPrevBlock: %x\n", hdr.HashPrevBlock)
		fmt.Printf("HashPrevBlock (real): %x\n", libkataware.DoubleSHA256(pre))
		fmt.Printf("HashMerkleRoot: %x\n", hdr.HashMerkleRoot)
		fmt.Printf("Time: %v\n", hdr.Time)
		fmt.Printf("Bits: %v\n", hdr.Bits)
		fmt.Printf("Nonce: %v\n\n", hdr.Nonce)*/
		pre = hdr.Serialize()
	}
}
