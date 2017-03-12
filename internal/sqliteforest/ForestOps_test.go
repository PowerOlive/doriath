package sqliteforest

import (
	"fmt"
	"math/rand"
	"testing"
)

func TestWeirdProofs(t *testing.T) {
	frst, err := OpenForest("file::memory:?cache=shared")
	if err != nil {
		t.Error(err)
		return
	}
	for i := 0; i < 10; i++ {
		for j := 0; j < 10; j++ {
			key := fmt.Sprintf("key%v,%v", i, j)
			value := []byte(fmt.Sprintf("val%v,%v", i, j))
			err = frst.StageDiff(key, value)
			if err != nil {
				t.Error(err)
				return
			}
		}
		frst.Commit()
	}
	roots, err := frst.TreeRoots()
	if err != nil {
		t.Error(err)
		return
	}
	for i := 0; i < 10; i++ {
		for j := 0; j < 10; j++ {
			key := fmt.Sprintf("key%v,%v", i, j)
			value := []byte(fmt.Sprintf("val%v,%v", i, j))
			var proof Proof
			proof, err = frst.FindProof(roots[i], key)
			if err != nil {
				t.Error(err)
				return
			}
			// vanilla proof should work
			if !proof.Check(roots[i], key, value) {
				t.Error("failed proof")
				return
			}
			// wrong proof type should fail
			if proof.Check(roots[i], key, nil) {
				t.Error("should have failed nonexistence proof!")
				return
			}
			// corupted proof should fail
			lol := make([]byte, 32*3+10)
			rand.Read(lol)
			var badnode AbbrNode
			badnode.FromBytes(lol)
			proof[rand.Int()%len(proof)] = badnode
			if proof.Check(roots[i], key, value) {
				t.Error("should have failed corrupted proof!")
				return
			}
		}
	}
}

func BenchmarkStaging(b *testing.B) {
	frst, err := OpenForest("/var/tmp/benchStaging.db")
	if err != nil {
		b.Error(err)
		return
	}
	frst.sdb.Exec("PRAGMA journal_mode=WAL")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key%v", i)
		value := make([]byte, 32)
		err := frst.StageDiff(key, value)
		if err != nil {
			b.Error(err)
			return
		}
	}
	b.StopTimer()
	frst.Commit()
}
