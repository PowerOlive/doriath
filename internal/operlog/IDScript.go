package operlog

// IDScript is an identity script, which is simply represented in binary form.
type IDScript []byte

// AssembleID takes in an identity script represented in assembly, and returns a binary ID script.
func AssembleID(asm string) (IDScript, error) {
	// use strings.Split and loop through the array of tokens, converting to binary on the fly
	panic("TODO")
}

// Verify runs the script with the given array of signatures as input. Null error means signature check is good. It must not panic even if the given array is too short, the script is garbage, etc, but should return appropriate errors.
func (ids IDScript) Verify(sigs []byte) error {
	// loop through the IDScript and interpret it
	panic("TODO")
}
