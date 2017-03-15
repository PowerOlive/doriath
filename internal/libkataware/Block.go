package libkataware

import (
	"bytes"
	"errors"
)

// Block represents a full block in the Bitcoin blockchain.
type Block struct {
	Hdr Header
	Bdy []Transaction
}

// Serialize serializes a block.
func (blk *Block) Serialize() []byte {
	buf := new(bytes.Buffer)
	buf.Write(blk.Hdr.Serialize())
	WriteVarint(buf, uint64(len(blk.Bdy)))
	for _, v := range blk.Bdy {
		v.Pack(buf)
	}
	return buf.Bytes()
}

// Deserialize deserializes a block.
func (blk *Block) Deserialize(bts []byte) error {
	if len(bts) < 80 {
		return errors.New("too short to possibly contain a block")
	}
	blk.Hdr.Deserialize(bts[:80])
	buf := bytes.NewBuffer(bts[80:])
	txcount, err := ReadVarint(buf)
	if err != nil {
		return err
	}
	blk.Bdy = nil
	for i := uint64(0); i < txcount; i++ {
		var lol Transaction
		err := lol.Unpack(buf)
		if err != nil {
			return err
		}
		blk.Bdy = append(blk.Bdy, lol)
	}
	if buf.Len() != 0 {
		return errors.New("garbage after block")
	}
	return nil
}

// GenMerkle generates a merkle tree branch and position (null if not possible)
func (blk *Block) GenMerkle(txHash []byte) ([][]byte, int) {
	// find tx idx first
	txIdx := -1
	for idx, tx := range blk.Bdy {
		if bytes.Compare(tx.Hash256(), txHash) == 0 {
			txIdx = idx
			break
		}
	}
	if txIdx == -1 {
		return nil, -1
	}
	// TODO understand this code, this is a stupid translation of blockchain_processor.py
	merkle := make([][]byte, len(blk.Bdy))
	for i, tx := range blk.Bdy {
		merkle[i] = tx.Hash256()
	}
	var s [][]byte
	for len(merkle) != 1 {
		if len(merkle)%2 == 1 {
			merkle = append(merkle, merkle[len(merkle)-1])
		}
		var n [][]byte
		for len(merkle) != 0 {
			newHash := DoubleSHA256(append(merkle[0], merkle[1]...))
			if bytes.Compare(merkle[0], txHash) == 0 {
				s = append(s, merkle[1])
				txHash = newHash
			} else if bytes.Compare(merkle[1], txHash) == 0 {
				s = append(s, merkle[0])
				txHash = newHash
			}
			n = append(n, newHash)
			merkle = merkle[2:]
		}
		merkle = n
	}
	return s, txIdx
}
