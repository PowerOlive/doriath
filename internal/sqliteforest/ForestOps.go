package sqliteforest

import (
	"bytes"

	"github.com/rensa-labs/doriath/internal/libkataware"
)

// Record is a struct representing a node in the forest.
type Record struct {
	Key       string
	Value     []byte
	LeftHash  []byte
	RightHash []byte
}

// Hash computes the double-SHA256 hash of a record.
func (rec Record) Hash() []byte {
	buf := new(bytes.Buffer)
	buf.Write([]byte(rec.Key))
	buf.Write(rec.Value)
	buf.Write(rec.LeftHash)
	buf.Write(rec.RightHash)
	return libkataware.DoubleSHA256(buf.Bytes())
}

// FindProof returns a proof of (non)existence given a tree-root hash and a key. The proof is structured as a list of records starting from the tree root.
func (fst *Forest) FindProof(trhash []byte, key string) (proof []Record, err error) {
	tx, err := fst.sdb.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Commit()
	rcurs := cursor{tx, trhash}
	ptrs, err := searchTree(rcurs, key)
	if err != nil {
		return nil, err
	}
	for _, v := range ptrs {
		var rec Record
		rec, err = v.getRecord()
		if err != nil {
			return
		}
		proof = append(proof, rec)
	}
	return
}
