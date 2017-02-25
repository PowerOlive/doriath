package libkataware

import (
	"crypto/sha256"
	"encoding/binary"
	"io"
)

// SwapBytes swaps bytes.
func SwapBytes(b []byte) []byte {
	// TODO in-place
	var toret []byte
	for _, v := range b {
		toret = append([]byte{v}, toret...)
	}
	return toret
}

// DoubleSHA256 computes sha256(sha256(b)).
func DoubleSHA256(b []byte) []byte {
	fst := sha256.Sum256(b)
	snd := sha256.Sum256(fst[:])
	return snd[:]
}

// ReadVarint reads a Bitcoin variable-length integer.
func ReadVarint(r io.Reader) (res uint64, err error) {
	var discr uint8
	err = binary.Read(r, binary.LittleEndian, &discr)
	if err != nil {
		return
	}
	switch discr {
	case 0xFF:
		// 64-bit
		err = binary.Read(r, binary.LittleEndian, &res)
		return
	case 0xFE:
		// 32-bit
		var r32 uint32
		err = binary.Read(r, binary.LittleEndian, &r32)
		res = uint64(r32)
		return
	case 0xFD:
		// 16-bit
		var r16 uint16
		err = binary.Read(r, binary.LittleEndian, &r16)
		res = uint64(r16)
		return
	default:
		// 8-bit
		res = uint64(discr)
		return
	}
}

// WriteVarint writes a 64-bit value as a Bitcoin variable-length integer.
func WriteVarint(w io.Writer, val uint64) error {
	if val < 0xFD {
		_, err := w.Write([]byte{byte(val)})
		return err
	} else if val <= 0xFFFF {
		w.Write([]byte{0xFD})
		return binary.Write(w, binary.LittleEndian, uint16(val))
	} else if val <= 0xFFFFFFFF {
		w.Write([]byte{0xFE})
		return binary.Write(w, binary.LittleEndian, uint32(val))
	}
	w.Write([]byte{0xFF})
	return binary.Write(w, binary.LittleEndian, val)
}
