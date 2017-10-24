package indextree

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"log"

	// sqlite3
	_ "github.com/mattn/go-sqlite3"

	"github.com/rensa-labs/doriath/electrumclient"
	"github.com/rensa-labs/doriath/internal/libkataware"
	"github.com/rensa-labs/doriath/operlog"
)

// ErrNoFunds is the error returned when there just isn't enough money.
var ErrNoFunds = errors.New("no more bitcoins left to fund the index tree")

// IndexTree is an index tree
type IndexTree struct {
	db      *sql.DB
	privKey string
	eclient *electrumclient.ElectrumClient
	bcast   bool
}

// NewIndexTree opens an index tree from a file name.
func NewIndexTree(fname string, privKey string,
	elecClient *electrumclient.ElectrumClient, bcast bool) (*IndexTree, error) {
	db, err := sql.Open("sqlite3", fmt.Sprintf("%v?_foreign_keys=1", fname))
	if err != nil {
		return nil, err
	}
	db.Exec("PRAGMA journal_mode=delete")
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS indexTree (
        txIndex INTEGER PRIMARY KEY,
        rawTx BLOB NOT NULL,
        leftChild INTEGER,
        rightChild INTEGER)`)
	if err != nil {
		panic(err)
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS txConfCache (
		txHash BLOB PRIMARY KEY,
		block INTEGER NOT NULL,
		merkle BLOB NOT NULL,
        posn INTEGER NOT NULL)`)
	if err != nil {
		panic(err)
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS funding (
		rawTx BLOB NOT NULL, outIdx INTEGER NOT NULL)`)
	if err != nil {
		panic(err)
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS addendum (
		parentIdx INTEGER NOT NULL REFERENCES indexTree(txIndex),
		rawTx BLOB NOT NULL)`)
	if err != nil {
		panic(err)
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS hashblobs (
		hash BLOB PRIMARY KEY, data BLOB NOT NULL)`)
	if err != nil {
		panic(err)
	}
	return &IndexTree{
		db:      db,
		privKey: privKey,
		eclient: elecClient,
		bcast:   bcast,
	}, nil
}

const mBTC = 100000

// Initialize initializes the tree if it isn't already initialized. It returns the root tree.
func (it *IndexTree) Initialize() (err error) {
	tx, err := it.db.Begin()
	if err != nil {
		return
	}
	defer tx.Rollback()
	// skip everything if we already have an index tree
	var nodecount int
	err = tx.QueryRow("SELECT COUNT(txIndex) FROM indexTree").Scan(&nodecount)
	if err != nil || nodecount > 0 {
		return
	}
	funds, totalMoney, err := it.getFunds(tx, mBTC)
	if err != nil {
		return
	}
	fee := int(it.eclient.EstimateFee(25) * 300)
	totalMoney -= fee
	noobuf := new(bytes.Buffer)
	fmt.Fprint(noobuf, "BFT")
	rootData := make([]byte, 64)
	rootData[0] = 127
	noobuf.Write(rootData)
	rootTx := it.treeNode(funds, totalMoney, noobuf.Bytes())
	_, err = tx.Exec("INSERT INTO indexTree (txIndex, rawTx) VALUES ($1, $2)", 0, rootTx.ToBytes())
	if err != nil {
		return
	}
	if it.bcast {
		err = it.eclient.BroadcastTx(rootTx.ToBytes())
		if err != nil {
			return
		}
	}
	tx.Commit()
	return
}

// Insert inserts a key-value pair into the index tree.
func (it *IndexTree) Insert(key string, data operlog.Operation) (err error) {
	index32 := sha256.Sum256([]byte(key))
	index := index32[:]
	if len(index) != 32 {
		panic("index not 32 bytes long")
	}
	tx, err := it.db.Begin()
	if err != nil {
		return
	}
	defer tx.Rollback()
	// insert the blob into the blob storage
	ophash := sha256.Sum256(data.ToBytes())
	_, err = tx.Exec("INSERT INTO hashblobs VALUES ($1, $2)", ophash[:], data.ToBytes())
	if err != nil {
		return
	}
	// find path of node numbers
	path, err := it.findPath(tx, index)
	if err != nil {
		return
	}
	parsedTx, _, _, err := it.getNode(tx, path[len(path)-1])
	if err != nil {
		return
	}
	tipIndex, _ := unpackNodeInfo(parsedTx.Outputs[3].Script[1:])
	// grow the tree
	var toSpend libkataware.TxInput
	var totalMoney uint64
	comparison := bytes.Compare(index, tipIndex)
	switch comparison {
	case -1:
		toSpend = libkataware.TxInput{
			PrevHash: parsedTx.Hash256(),
			PrevIdx:  0,
			Script:   parsedTx.Outputs[0].Script,
			Seqno:    0xffffffff,
		}
		totalMoney = parsedTx.Outputs[0].Value
	case 1:
		toSpend = libkataware.TxInput{
			PrevHash: parsedTx.Hash256(),
			PrevIdx:  1,
			Script:   parsedTx.Outputs[1].Script,
			Seqno:    0xffffffff,
		}
		totalMoney = parsedTx.Outputs[1].Value
	default:
		existingLog, e := it.getLog(tx, path[len(path)-1])
		if e != nil {
			err = e
			return
		}
		if len(existingLog) > 0 {
			logTip := existingLog[len(existingLog)-1]
			toSpend = libkataware.TxInput{
				PrevHash: logTip.Hash256(),
				PrevIdx:  0,
				Script:   logTip.Outputs[0].Script,
				Seqno:    0xffffffff,
			}
			totalMoney = logTip.Outputs[0].Value
		} else {
			toSpend = libkataware.TxInput{
				PrevHash: parsedTx.Hash256(),
				PrevIdx:  2,
				Script:   parsedTx.Outputs[2].Script,
				Seqno:    0xffffffff,
			}
			totalMoney = parsedTx.Outputs[2].Value
		}
	}
	// add the new transaction
	funds := []libkataware.TxInput{toSpend}
	if totalMoney < mBTC/10 {
		// get another mBTC if we don't have enough
		newfunds, newMoney, e := it.getFunds(tx, mBTC)
		if e != nil {
			err = e
			return
		}
		totalMoney += uint64(newMoney)
		funds = append(funds, newfunds...)
	}
	// create the new transaction
	feeFactor := it.eclient.EstimateFee(25)
	if feeFactor > 100 {
		feeFactor = 100
	} else if feeFactor < 10 {
		feeFactor = 10
	}
	fee := feeFactor * 300
	totalMoney -= uint64(fee)
	// if this is an append transaction just do it now
	if comparison == 0 {
		newTx := it.logNode(funds, int(totalMoney), ophash[:])
		_, err = tx.Exec("INSERT INTO addendum VALUES ($1, $2)",
			path[len(path)-1], newTx.ToBytes())
		if err != nil {
			return
		}
		// look up and make sure nothing's funny
		_, _, err = it.lookupWithTx(tx, index)
		if err != nil {
			return
		}
		if it.bcast {
			err = it.eclient.BroadcastTx(newTx.ToBytes())
			if err != nil {
				return
			}
		}
		return tx.Commit()
	}
	newTx := it.treeNode(funds, int(totalMoney), packNodeInfo(index, ophash[:]))
	// insert the transaction into DB and link it to its parent
	_, err = tx.Exec("INSERT INTO indexTree (rawTx) VALUES ($1)", newTx.ToBytes())
	if err != nil {
		return
	}
	var newID int
	err = tx.QueryRow("SELECT last_insert_rowid()").Scan(&newID)
	if err != nil {
		return
	}
	switch comparison {
	case -1:
		_, err = tx.Exec("UPDATE indexTree SET leftChild = $1 WHERE txIndex = $2",
			newID, path[len(path)-1])
	case 1:
		_, err = tx.Exec("UPDATE indexTree SET rightChild = $1 WHERE txIndex = $2",
			newID, path[len(path)-1])
	}
	if err != nil {
		return
	}
	// broadcast and commit
	if it.bcast {
		err = it.eclient.BroadcastTx(newTx.ToBytes())
		if err != nil {
			return
		}
	}
	return tx.Commit()
}

// ConfirmedTx represents a transaction that's confirmed at a certain block height.
type ConfirmedTx struct {
	Transaction []byte
	BlockNum    int
	Merkle      [][]byte
	Index       int
}

// Lookup looks up a name, returning a list of ops, and confirmed transactions.
func (it *IndexTree) Lookup(key string) (ops operlog.OperLog, txes []ConfirmedTx, err error) {
	tx, err := it.db.Begin()
	if err != nil {
		return
	}
	defer tx.Rollback()
	index32 := sha256.Sum256([]byte(key))
	index := index32[:]
	bops, path, err := it.lookupWithTx(tx, index)
	if err != nil {
		return
	}
	var parseds []libkataware.Transaction
	for _, v := range path {
		var parsedTx libkataware.Transaction
		err = parsedTx.FromBytes(v)
		if err != nil {
			panic(err)
		}
		confTx := ConfirmedTx{
			Transaction: parsedTx.ToBytes(),
		}
		txes = append(txes, confTx)
		parseds = append(parseds, parsedTx)
	}
	for i := range path {
		blkIdx, merkle, posn, e := it.getConfirm(tx, parseds[i])
		if e != nil {
			break
		}
		txes[i].BlockNum, txes[i].Merkle, txes[i].Index = blkIdx, merkle, posn
	}
	for _, v := range bops {
		log.Printf("v = %x", v)
		var op operlog.Operation
		err = op.FromBytes(v)
		if err != nil {
			panic(err)
		}
		ops = append(ops, op)
	}
	return
}

// lookupWithTx looks up while holding a transaction.
func (it *IndexTree) lookupWithTx(tx *sql.Tx, index []byte) (ops [][]byte, path [][]byte, err error) {
	treePath, err := it.findPath(tx, index)
	if err != nil {
		return
	}
	var parsedPath []libkataware.Transaction
	for _, txi := range treePath {
		nd, _, _, e := it.getNode(tx, txi)
		if e != nil {
			err = e
			return
		}
		parsedPath = append(parsedPath, nd)
	}
	// now that we have path, decide whether to go further based on the tip
	treetip := parsedPath[len(parsedPath)-1]
	tipIdx, tipOpHash := unpackNodeInfo(treetip.Outputs[3].Script[1:])
	if bytes.Compare(index, tipIdx) != 0 {
		// bail now, this is a (weak) proof of nonexistence
		for _, v := range parsedPath {
			path = append(path, v.ToBytes())
			_, hsh := unpackNodeInfo(v.Outputs[3].Script[1:])
			var blob []byte
			err = tx.QueryRow("SELECT data FROM hashblobs WHERE hash=$1", hsh).Scan(&blob)
			if err != nil {
				return
			}
		}
		return
	}
	// we do go further; start by appending the tip to the ops, then get the log
	var firstOp []byte
	err = tx.QueryRow("SELECT data FROM hashblobs WHERE hash=$1", tipOpHash).Scan(&firstOp)
	if err != nil {
		return
	}
	ops = append(ops, firstOp)
	rem, err := it.getLog(tx, treePath[len(treePath)-1])
	if err != nil {
		return
	}
	for _, logTx := range rem {
		parsedPath = append(parsedPath, logTx)
		hsh := logTx.Outputs[1].Script[4:]
		var logOp []byte
		err = tx.QueryRow("SELECT data FROM hashblobs WHERE hash=$1", hsh).Scan(&logOp)
		if err != nil {
			return
		}
		ops = append(ops, logOp)
	}
	var toCheck operlog.OperLog
	for _, v := range ops {
		var parsed operlog.Operation
		err = parsed.FromBytes(v)
		if err != nil {
			panic(err)
		}
		toCheck = append(toCheck, parsed)
	}
	if !toCheck.IsValid() {
		err = operlog.ErrInvalidID
		return
	}
	for _, v := range parsedPath {
		path = append(path, v.ToBytes())
	}
	return
}
