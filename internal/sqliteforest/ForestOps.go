package sqliteforest

import (
	"bytes"
	"crypto/subtle"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rensa-labs/doriath/internal/libkataware"
)

// fullNode is a struct representing a node in the forest.
type fullNode struct {
	Key       string
	Value     []byte
	LeftHash  []byte
	RightHash []byte
}

// Hash computes the "standard" hash of a record as specified in the docs.
func (rec fullNode) Hash() []byte {
	if rec.LeftHash == nil {
		rec.LeftHash = make([]byte, 32)
	}
	if rec.RightHash == nil {
		rec.RightHash = make([]byte, 32)
	}
	buf := new(bytes.Buffer)
	buf.Write([]byte(rec.Key))
	buf.Write(libkataware.DoubleSHA256(rec.Value))
	buf.Write(rec.LeftHash)
	buf.Write(rec.RightHash)
	return libkataware.DoubleSHA256(buf.Bytes())
}

// Proof is a proof of inclusion or exclusion, represented as an array of abbreviated nodes.
type Proof []AbbrNode

// Check checks the proof for correctness
func (pr Proof) Check(rootHash []byte, key string, value []byte) bool {
	// General idea of the algorithm:
	// Check that there are no duplicate keys in the proof.
	// For each pair of adjacent nodes x and y in the hash,
	//   figure out which child of x is y (pretend we are doing an interactive search)
	//   make sure y hashes to that child
	//   make sure y's key is greater/lesser than x
	// Check that no nodes before the last have our key.
	// If we want a nonexistence proof, check that the last node in the proof has no children and does not have K
	// If we want an existence proof, check that the last node has the K/V

	comp := subtle.ConstantTimeCompare
	H := libkataware.DoubleSHA256
	if len(pr) < 1 {
		return false // WTF
	}
	if len(pr) == 1 {
		return comp(H(pr[0].ToBytes()), rootHash) == 1 &&
			pr[0].Key == key &&
			comp(pr[0].VHash, H(value)) == 1
	}
	// general case
	seenKeys := make(map[string]bool)
	for _, node := range pr {
		if seenKeys[node.Key] {
			return false
		}
		seenKeys[node.Key] = true
	}
	for i := 0; i < len(pr)-1; i++ {
		x := pr[i]
		y := pr[i+1]
		// check y is the correct child of x
		switch strings.Compare(key, x.Key) {
		case -1:
			if comp(x.LHash, H(y.ToBytes())) != 1 ||
				strings.Compare(y.Key, x.Key) != -1 {
				return false
			}
		case 1:
			if comp(x.RHash, H(y.ToBytes())) != 1 ||
				strings.Compare(y.Key, x.Key) != 1 {
				return false
			}
		case 0:
			// key should not appear here!
			return false
		}
	}
	// check last node
	last := pr[len(pr)-1]
	if value == nil {
		return last.Key != key &&
			comp(last.LHash, make([]byte, 32)) == 1 &&
			comp(last.LHash, make([]byte, 32)) == 1
	}
	return last.Key == key && comp(last.VHash, H(value)) == 1
}

// AbbrNode is a struct representing an abbreviated node used in a proof.
type AbbrNode struct {
	Key   string
	VHash []byte
	LHash []byte
	RHash []byte
}

// ToBytes serializes an abbreviated node.
func (an *AbbrNode) ToBytes() []byte {
	buf := new(bytes.Buffer)
	buf.Write([]byte(an.Key))
	buf.Write(an.VHash)
	buf.Write(an.LHash)
	buf.Write(an.RHash)
	return buf.Bytes()
}

// FromBytes deserializes an abbreviated node.
func (an *AbbrNode) FromBytes(b []byte) error {
	if len(b) < 32*3 {
		return errors.New("malformed AbbrNode bytes")
	}
	cb := make([]byte, len(b))
	copy(cb, b) // avoid aliasing shenanigans
	an.Key = string(cb[:len(b)-32*3])
	cb = cb[len(b)-32*3:]
	an.VHash = cb[0:32]
	an.LHash = cb[32:64]
	an.RHash = cb[64:96]
	return nil
}

// String gets a string from an abbrnode
func (an AbbrNode) String() string {
	return fmt.Sprintf("{K=%v V=%x L=%x R=%x}",
		an.Key, an.VHash[:10], an.LHash[:10], an.RHash[:10])
}

// FindProof returns a proof of (non)existence given a tree-root hash and a key.
func (fst *Forest) FindProof(trhash []byte, key string) (proof Proof, err error) {
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
		var rec fullNode
		rec, err = v.getRecord()
		if err != nil {
			return
		}
		if rec.LeftHash == nil {
			rec.LeftHash = make([]byte, 32)
		}
		if rec.RightHash == nil {
			rec.RightHash = make([]byte, 32)
		}
		proof = append(proof, AbbrNode{
			Key:   rec.Key,
			VHash: libkataware.DoubleSHA256(rec.Value),
			LHash: rec.LeftHash,
			RHash: rec.RightHash,
		})
	}
	return
}

// TreeRoots returns an array of all the tree root hashes in the forest, in chronological order.
func (fst *Forest) TreeRoots() (roots [][]byte, err error) {
	tx, err := fst.sdb.Begin()
	if err != nil {
		return
	}
	defer tx.Commit()
	rows, err := tx.Query("SELECT rhash FROM treeroots")
	if err != nil {
		return
	}
	for rows.Next() {
		var nv []byte
		err = rows.Scan(&nv)
		if err != nil {
			return
		}
		roots = append(roots, nv)
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
