package operlog

// OperLog represents an entire operation log.
type OperLog []Operation

// IsValid checks whether the signature chain on the operation log is valid.
func (ol OperLog) IsValid() bool {
	// TODO non-recursive
	if len(ol) < 2 {
		return true // trivial case
	}
	frst, secd := ol[0], ol[1]
	return frst.NextID.Verify(secd.SignedPart(), secd.Signatures) == nil && ol[1:].IsValid()
}

// LastData returns the data field of the last operation; i.e. the current data binding.
func (ol OperLog) LastData() []byte {
	return ol[len(ol)-1].Data
}
