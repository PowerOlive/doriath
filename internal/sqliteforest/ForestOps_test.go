package sqliteforest

import (
	"bytes"
	"fmt"
	"math/rand"
	"testing"

	"github.com/rensa-labs/doriath/internal/libkataware"
	"github.com/rensa-labs/doriath/operlog"
)

func demoOp(data string) (val operlog.Operation) {
	val.Data = data
	return
}

func TestWeirdProofs(t *testing.T) {
	frst, err := OpenForest("file::memory:?cache=shared")
	if err != nil {
		t.Error(err)
		return
	}
	for i := 0; i < 10; i++ {
		for j := 0; j < 10; j++ {
			key := fmt.Sprintf("key%v,%v", i, j)
			value := demoOp(fmt.Sprintf("val%v,%v", i, j))
			err = frst.Stage(key, value)
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
			valop := demoOp(fmt.Sprintf("val%v,%v", i, j))
			value := libkataware.DoubleSHA256(valop.ToBytes())
			var proof Proof
			proof, _, err = frst.FindProof(roots[i], key)
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

func TestMultipleOps(t *testing.T) {
	frst, err := OpenForest("file::memory:?cache=shared")
	defer frst.Close()
	if err != nil {
		panic(err)
	}
	operations := make([]operlog.Operation, 10)
	for i := 0; i < 10; i++ {
		operations[i] = demoOp(fmt.Sprintf("Data number %v", i))
		frst.Stage("FOOBAR", operations[i])
	}
	res, err := frst.SearchStaging("FOOBAR")
	if err != nil {
		panic(err)
	}
	for i, v := range res {
		if bytes.Compare(v.ToBytes(), operations[i].ToBytes()) != 0 {
			t.FailNow()
		}
	}
	frst.Commit()
	tr, _ := frst.TreeRoots()
	theroot := tr[len(tr)-1]
	_, value, _ := frst.FindProof(theroot, "FOOBAR")
	for i, v := range value {
		if bytes.Compare(v.ToBytes(), operations[i].ToBytes()) != 0 {
			t.FailNow()
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
	frst.sdb.Exec("PRAGMA synchronous=NORMAL")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key%v", i)
		value := demoOp("helloworld")
		err := frst.Stage(key, value)
		if err != nil {
			b.Error(err)
			return
		}
	}
	b.StopTimer()
	frst.Commit()
}
