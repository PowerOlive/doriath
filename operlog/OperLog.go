package operlog

// OperLog represents an entire operation log.
type OperLog []Operation

// IsValid checks whether the signature chain on the operation log is valid.
func (ol OperLog) IsValid() bool {
	return ol.isSigValid() && ol.isNonceValid()
}

func (ol OperLog) isSigValid() bool {
	// TODO non-recursive
	if len(ol) < 2 {
		return true // trivial case
	}
	frst, secd := ol[0], ol[1]
	return frst.NextID.Verify(secd.SignedPart(), secd.Signatures) == nil && ol[1:].isSigValid()
}

func (ol OperLog) isNonceValid() bool {
	seen := make(map[string]bool)
	for _, v := range ol {
		if seen[string(v.Nonce)] {
			return false
		}
		seen[string(v.Nonce)] = true
	}
	return true
}

// LastData returns the data field of the last operation; i.e. the current data binding.
func (ol OperLog) LastData() string {
	return ol[len(ol)-1].Data
}
