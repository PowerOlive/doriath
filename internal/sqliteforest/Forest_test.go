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
	defer os.Remove(dbFileName)
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
	}
	frst.sdb.Close()
	RemoveTempDbFile(t, dbFileName)
}

func TestNonExistingDb(t *testing.T) {
	frst, err := OpenForest("/does/not/exist/invalid.db")
	if err != nil {
		// expecting an error; simply ignore
	}
	frst.DumpDOT(ioutil.Discard) // should do nothing
}

func TestFindProofFailOnMissingDb(t *testing.T) {
	dbFileName := CreateTempDbFile(t, "test-forest.db")
	fst, err := OpenForest(dbFileName)
	if err != nil {
		t.Error(err)
	}

	tx, err := fst.BeginTx()
	if err != nil {
		t.Error(err)
	}

	testDict := CreateTestDict(10)

	root, err := allocDict(tx, testDict)
	if err != nil {
		t.Error(err)
	}
	tx.Commit()

	// close and delete the temp db file now
	fst.sdb.Close()
	RemoveTempDbFile(t, dbFileName)

	// calling FindProof on fst will raise an exception
	_, err = fst.FindProof(root.loc, "key1")
	if err != nil {
		// expecting this; ignore
	}
}
