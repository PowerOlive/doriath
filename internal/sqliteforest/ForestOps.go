package sqliteforest

import (
	"bytes"
	"time"

	"github.com/rensa-labs/doriath/internal/libkataware"
)

// Record is a struct representing a node in the forest.
type Record struct {
	Key       string
	Value     []byte
	LeftHash  []byte
	RightHash []byte
}

// Hash computes the "standard" hash of a record as specified in the docs.
func (rec Record) Hash() []byte {
	buf := new(bytes.Buffer)
	buf.Write([]byte(rec.Key))
	buf.Write(libkataware.DoubleSHA256(rec.Value))
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

// StageDiff atomically stages a key and value into the staging area.
func (fst *Forest) StageDiff(key string, value []byte) (err error) {
	tx, err := fst.sdb.Begin()
	if err != nil {
		return
	}
	defer tx.Rollback()
	_, err = tx.Exec("INSERT INTO uncommitted VALUES ($1, $2)", key, value)
	if err != nil {
		return
	}
	return tx.Commit()
}

// Commit commits everything staged into a new tree, and puts the root of that tree in the tree root list, all as one atomic transaction. It does not touch the blockchain.
func (fst *Forest) Commit() (err error) {
	tx, err := fst.sdb.Begin()
	if err != nil {
		return
	}
	defer tx.Rollback()
	// Gather everything into a dict
	toinsert := make(map[string][]byte)
	iter, err := tx.Query("SELECT * FROM uncommitted")
	if err != nil {
		return
	}
	for iter.Next() {
		var key string
		var val []byte
		err = iter.Scan(&key, &val)
		if err != nil {
			return
		}
		toinsert[key] = val
	}
	// Use the standard dict insertion
	nroot, err := allocDict(tx, toinsert)
	if err != nil {
		return
	}
	// Write the root to treeroots
	_, err = tx.Exec("INSERT INTO treeroots VALUES ((SELECT COUNT(*) FROM treeroots), $2, $3)",
		time.Now().Unix(), nroot.loc)
	if err != nil {
		return
	}
	// Clear the staging area
	_, err = tx.Exec("DELETE FROM uncommitted")
	if err != nil {
		return
	}
	return tx.Commit()
}
