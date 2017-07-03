// +build gofuzz

package operlog

import (
	"bytes"
	"encoding/json"
	"log"
)

// Fuzz function for use with go-fuzz
func Fuzz(data []byte) int {
	var op Operation
	err := op.FromBytes(data)
	if err != nil {
		return 0
	}
	jsn, _ := json.MarshalIndent(op, "", "    ")
	out := op.ToBytes()
	if bytes.Compare(out, data) != 0 {
		log.Print(string(jsn))
		log.Printf("%x", out)
		log.Printf("%x", data)
		panic("does not work!")
	}
	return 1
}
