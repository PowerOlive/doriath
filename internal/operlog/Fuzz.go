// +build gofuzz

package operlog

// Fuzz function for use with go-fuzz
func Fuzz(data []byte) int {
	/*var op Operation
	err := op.FromBytes(data)
	if err != nil {
		log.Println(err.Error())
		return 0
	}
	jsn, _ := json.MarshalIndent(op, "", "    ")
	log.Print(string(jsn))
	out := op.ToBytes()
	if bytes.Compare(out, data) != 0 {
		log.Printf("%x", out)
		log.Printf("%x", data)
		panic("does not work!")
	}
	return 1*/
	// fuzzing the IDScript assembler
	ids, err := AssembleID(string(data))
	if err != nil {
		//panic(err.Error())
		return 0
	}
	err = ids.Verify(nil, make([][]byte, 100))
	if err != nil && err != ErrNoQuorum {
		return 0
	}
	return 1
}
