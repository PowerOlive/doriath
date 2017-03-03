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

// TestFaultyCursor creates a faulty cursor which contains an SQL tx which has already been committed. The purpose of this test is to show cursor error handlings are working.
func TestFaultyCursor(t *testing.T) {
	dbFileName := CreateTempDbFile(t, "test-forest.db")
	fst, err := OpenForest(dbFileName)
	if err != nil {
		t.Error(err)
	}
	defer fst.sdb.Close()
	defer RemoveTempDbFile(t, dbFileName)

	tx, err := fst.BeginTx()
	if err != nil {
		t.Error(err)
	}

	r := Record{
		Key: "key", Value: []byte("value")}

	cr, err := allocCursor(tx, r)
	if err != nil {
		t.Error(err)
	}

	// finish the transaction on purpose
	tx.Commit()
	// From this point, the cursor becomes 'faulty'

	// this should fail as tx has already been committed
	_, err = cr.getLeft()
	if err != nil {
		// expected
	}

	// this should fail as tx has already been committed
	_, err = cr.getRight()
	if err != nil {
		// expected
	}

	_, err = allocCursor(tx, r)
	if err != nil {
		// expected
	}

	_, err = searchTree(cr, "key")
	if err != nil {
		// expected
	}
}
