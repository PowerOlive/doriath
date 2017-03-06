package sqliteforest

import (
	"database/sql"
	"fmt"
	"io"
	"log"

	"github.com/mattn/go-sqlite3"
)

// Forest represents a SQLite-backed diff forest.
type Forest struct {
	sdb *sql.DB
}

// OpenForest opens or creates a forest with the given filename.
func OpenForest(fname string) (forest *Forest, err error) {
	lol, err := sql.Open("sqlite3_with_fk", fname)
	if err != nil {
		return
	}
	forest = &Forest{
		sdb: lol,
	}
	tx, err := forest.sdb.Begin()
	if err != nil {
		return
	}
	defer tx.Commit()
	tx.Exec(`CREATE TABLE IF NOT EXISTS treenodes (
                hash      BLOB PRIMARY KEY,
                key       TEXT NOT NULL,
                value     BLOB NOT NULL,
                lefthash  BLOB REFERENCES treenodes(hash),
                righthash BLOB REFERENCES treenodes(hash))`)
	tx.Exec(`CREATE TABLE IF NOT EXISTS treeroots (
                serial INTEGER PRIMARY KEY,
                ctime  INTEGER NOT NULL,
                rhash  BLOB REFERENCES treenodes(hash))`)
	tx.Exec(`CREATE TABLE IF NOT EXISTS uncommitted (
                key   TEXT PRIMARY KEY,
                value NOT NULL)`)
	return
}

// Close releases all non-garbage-collectible resources that the forest holds.
func (fst *Forest) Close() {
	fst.sdb.Close()
}

// DumpDOT dumps a GraphViz .dot for debugging.
func (fst *Forest) DumpDOT(out io.Writer) {
	fmt.Fprintf(out, "digraph G {\nrankdir=\"TB\"\n")

	defer fmt.Fprintf(out, "}\n")
	dump, err := fst.sdb.Query("SELECT hash,key,lefthash,righthash FROM treenodes")
	if err != nil {
		log.Println(err.Error())
		return
	}
	var count int
	for dump.Next() {
		var item Record
		var hash []byte
		dump.Scan(&hash, &item.Key, &item.LeftHash, &item.RightHash)
		fmt.Fprintf(out, "\"%x\" [label=\"%v\"]\n", hash[:8], item.Key)
		if item.LeftHash != nil {
			fmt.Fprintf(out, "\"%x\" -> \"%x\"\n", hash[:8], item.LeftHash[:8])
		}
		if item.RightHash != nil {
			fmt.Fprintf(out, "\"%x\" -> \"%x\"\n", hash[:8], item.RightHash[:8])
		}
		count++
	}
}

// SQLite3 with foreign keys
func init() {
	sql.Register("sqlite3_with_fk",
		&sqlite3.SQLiteDriver{
			ConnectHook: func(conn *sqlite3.SQLiteConn) error {
				_, err := conn.Exec("PRAGMA foreign_keys = ON", nil)
				return err
			},
		})
}
