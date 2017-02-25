package libkataware

import (
	"crypto/subtle"
	"encoding/binary"
	"time"
)

// HeaderLen is the length of a serialized Header.
const HeaderLen = 80

// Header represents a Bitcoin block header.
type Header struct {
	Version        uint32
	HashPrevBlock  []byte
	HashMerkleRoot []byte
	Time           time.Time
	Bits           uint32
	Nonce          uint32
}

// Serialize serializes the header into wire format.
func (hdr *Header) Serialize() []byte {
	tgt := make([]byte, HeaderLen)
	binary.LittleEndian.PutUint32(tgt[0:4], hdr.Version)
	copy(tgt[4:36], hdr.HashPrevBlock)
	copy(tgt[36:68], hdr.HashMerkleRoot)
	binary.LittleEndian.PutUint32(tgt[68:72], uint32(hdr.Time.Unix()))
	binary.LittleEndian.PutUint32(tgt[72:76], hdr.Bits)
	binary.LittleEndian.PutUint32(tgt[76:80], hdr.Nonce)
	return tgt
}

// Deserialize populates the fields of the header from wire format.
func (hdr *Header) Deserialize(ob []byte) {
	b := make([]byte, len(ob))
	copy(b, ob)
	if len(b) != HeaderLen {
		panic("attempted to deserialize from header of wrong length")
	}
	hdr.Version = binary.LittleEndian.Uint32(b[0:4])
	//swapBytes(b[4:36])
	hdr.HashPrevBlock = b[4:36]
	//(b[36:68])
	hdr.HashMerkleRoot = b[36:68]
	hdr.Time = time.Unix(int64(binary.LittleEndian.Uint32(b[68:72])), 0)
	hdr.Bits = binary.LittleEndian.Uint32(b[72:76])
	hdr.Nonce = binary.LittleEndian.Uint32(b[76:80])
}

// CheckExists checks that a certain transaction exists in the block identified by the header, given a Merkle tree branch.
func (hdr *Header) CheckExists(merkle [][]byte, pos int, tx Transaction) bool {
	h := tx.Hash256()
	for i, elem := range merkle {
		if (uint(pos)>>uint(i))&1 != 0 {
			h = DoubleSHA256(append(elem, h...))
		} else {
			h = DoubleSHA256(append(h, elem...))
		}
	}
	if subtle.ConstantTimeCompare(h, hdr.HashMerkleRoot) == 1 {
		return true
	}
	return false
}
