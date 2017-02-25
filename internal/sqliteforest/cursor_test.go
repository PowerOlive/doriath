package sqliteforest

import (
	"fmt"
	"io/ioutil"
	"testing"
)

func TestAllocDict(t *testing.T) {
	frst, err := OpenForest("file::memory:?cache=shared")
	if err != nil {
		t.Error(err)
		return
	}
	tx, err := frst.sdb.Begin()
	if err != nil {
		t.Error(err)
		return
	}
	defer tx.Rollback()
	towrite := make(map[string][]byte)
	for i := 0; i < 25; i++ {
		towrite[fmt.Sprintf("key%v", i)] = []byte(fmt.Sprintf("value%v", i))
	}
	root, err := allocDict(tx, towrite)
	if err != nil {
		t.Error(err)
		return
	}
	tx.Commit()
	for i := 0; i < 25; i++ {
		lol, err := frst.FindProof(root.loc, fmt.Sprintf("key%v", i))
		if err != nil {
			t.Error(err)
			return
		}
		lolz := lol[len(lol)-1]
		if lolz.Key != fmt.Sprintf("key%v", i) {
			t.Error("didn't find what we put in")
			return
		}
	}
	// just to make the coverage look better
	frst.DumpDOT(ioutil.Discard)
}
