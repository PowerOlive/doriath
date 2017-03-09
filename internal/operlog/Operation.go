package operlog

// Operation represents a single operation.
type Operation struct {
	NextID     IDScript
	Data       []byte
	Signatures [][]byte
}

// ToBytes serializes an operation to a byte array.
func (op *Operation) ToBytes() []byte {
	// Hint: make a bytes.Buffer and write to it using binary.Write
	panic("TODO")
}

// FromBytes fills an operation struct by parsing a byte array.
func (op *Operation) FromBytes() error {
	// Note that op is passed by reference.
	// Hint: make a bytes.Buffer and read from it using binary.Read
	panic("TODO")
}
