package sqliteforest

import (
	"fmt"
	"math/rand"
	"os"
	"testing"
)

func TestWeirdTree(t *testing.T) {
	frst, err := OpenForest("file::memory:?cache=shared")
	if err != nil {
		t.Error(err)
		return
	}
	for i := 0; i < 2; i++ {
		for j := 0; j < rand.Int()%20+2; j++ {
			key := fmt.Sprintf("tfo%v,%v", i, j)
			value := make([]byte, 32)
			err := frst.StageDiff(key, value)
			if err != nil {
				t.Error(err)
				return
			}
		}
		frst.Commit()
	}
	frst.DumpDOT(os.Stdout)
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
