package main

import (
	"fmt"
	"os"

	"github.com/rensa-labs/doriath/internal/sqliteforest"
)

func main() {
	frst, err := sqliteforest.OpenForest("file::memory:?cache=shared")
	if err != nil {
		panic(err.Error())
	}
	for i := 1; i < 10; i++ {
		for j := 0; j < i; j++ {
			key := fmt.Sprintf("%v", j)
			value := make([]byte, 32)
			err := frst.StageDiff(key, value)
			if err != nil {
				panic(err.Error())
			}
		}
		frst.Commit()
	}
	frst.DumpDOT(os.Stdout)
}
