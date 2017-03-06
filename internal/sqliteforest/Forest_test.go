package sqliteforest

import "testing"

func TestCreateForest(t *testing.T) {
	_, err := OpenForest("file::memory:?cache=shared")
	if err != nil {
		t.Error(err)
	}
}
