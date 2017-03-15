// +build gofuzz

package libkataware

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// Fuzz function for use with go-fuzz
func Fuzz(data []byte) int {
	// interpret as block and start from there
	var blk Block
	err := blk.Deserialize(data)
	if err != nil {
		return 0
	}
	// we want "interesting" blocks
	if len(blk.Bdy) > 5 {
		return 0
	}
	srlz := blk.Serialize()
	if bytes.Compare(srlz, data) != 0 {
		fmt.Println("Oh no, the fuzzer gonna crash!")
		jsn, _ := json.MarshalIndent(&blk, "", "    ")
		fmt.Println(string(jsn))
		fmt.Println(hex.EncodeToString(data))
		fmt.Println(hex.EncodeToString(srlz))
		panic("Wrong!!!!")
	}
	for _, v := range blk.Bdy {
		blk.GenMerkle(v.Hash256())
	}
	return 1
}
