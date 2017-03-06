package sqliteforest

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
)

func CreateTempDbFile(t *testing.T, dbFileName string) string {
	f, err := ioutil.TempFile(".", dbFileName)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	return f.Name()
}

func RemoveTempDbFile(t *testing.T, dbFileName string) {
	os.Remove(dbFileName)
}

func CreateTestDict(size int) map[string][]byte {
	tmp := make(map[string][]byte)
	for i := 0; i < size; i++ {
		tmp[fmt.Sprintf("key%v", i)] = []byte(fmt.Sprintf("value%v", i))
	}
	return tmp
}

func TestCreateForest(t *testing.T) {
	dbFileName := CreateTempDbFile(t, "test-forest.db")
	frst, err := OpenForest(dbFileName)
	if err != nil {
		t.Error(err)
		return
	}
	frst.Close()
	RemoveTempDbFile(t, dbFileName)
}

func TestNonExistingDbDir(t *testing.T) {
	frst, err := OpenForest("/does/not/exist/invalid.db")
	if err == nil {
		// error happens because the *directory* doesn't exist
		t.Error("we expected an error")
		return
	}
	frst.DumpDOT(ioutil.Discard) // should do nothing
}

func TestFindProofFailOnMissingDb(t *testing.T) {
	dbFileName := CreateTempDbFile(t, "test-forest.db")
	fst, err := OpenForest(dbFileName)
	if err != nil {
		t.Error(err)
		return
	}

	tx, err := fst.sdb.Begin()
	if err != nil {
		t.Error(err)
		return
	}

	testDict := CreateTestDict(10)

	root, err := allocDict(tx, testDict)
	if err != nil {
		t.Error(err)
		return
	}
	tx.Commit()

	// close and delete the temp db file now
	fst.sdb.Close()
	RemoveTempDbFile(t, dbFileName)

	// calling FindProof on fst will fail
	_, err = fst.FindProof(root.loc, "key1")
	if err == nil {
		t.Error("we expected an error because fst is closed")
	}
}
