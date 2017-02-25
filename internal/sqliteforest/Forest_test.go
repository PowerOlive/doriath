package sqliteforest

import "testing"

func TestCreateForest(t *testing.T) {
	_, err := OpenForest("/scratch/test-forest.db")
	if err != nil {
		t.Error(err)
	}
}
