package indextree

import (
	"bytes"
	"database/sql"

	"github.com/rensa-labs/doriath/internal/libkataware"
)

// treeNode creates a tree node transaction.
func (it *IndexTree) treeNode(funds []libkataware.TxInput, totalMoney int,
	data []byte) libkataware.Transaction {
	unsig := libkataware.Transaction{
		Version: 1,
		Inputs:  funds,
		Outputs: []libkataware.TxOutput{
			libkataware.TxOutput{ // spend by left child
				Value:  uint64(totalMoney/2 - 500),
				Script: funds[0].Script,
			},
			libkataware.TxOutput{ // spend by right child
				Value:  uint64(totalMoney/2 - 500),
				Script: funds[0].Script,
			},
			libkataware.TxOutput{ // spend by operation log
				Value:  1000,
				Script: funds[0].Script,
			},
			libkataware.TxOutput{ // OP_RETURN thing
				Value:  0,
				Script: append([]byte{0x6a}, data...),
			},
		},
	}
	toret, err := it.eclient.SignTx(unsig.ToBytes(), it.privKey)
	if err != nil {
		panic(err)
	}
	err = unsig.FromBytes(toret)
	if err != nil {
		panic(err)
	}
	return unsig
}

// logNode creates a log node transaction.
func (it *IndexTree) logNode(funds []libkataware.TxInput, totalMoney int,
	data []byte) libkataware.Transaction {
	unsig := libkataware.Transaction{
		Version: 1,
		Inputs:  funds,
		Outputs: []libkataware.TxOutput{
			libkataware.TxOutput{ // spend by next log entry
				Value:  uint64(totalMoney),
				Script: funds[0].Script,
			},
			libkataware.TxOutput{ // OP_RETURN encoding log entry
				Value:  0,
				Script: append(append([]byte{0x6a}, []byte("BFL")...), data...),
			},
		},
	}
	toret, err := it.eclient.SignTx(unsig.ToBytes(), it.privKey)
	if err != nil {
		panic(err)
	}
	err = unsig.FromBytes(toret)
	if err != nil {
		panic(err)
	}
	return unsig
}

func unpackNodeInfo(script []byte) (index []byte, headHash []byte) {
	script = script[3:] // skip the BFT... part
	return script[:32], script[32:]
}

func packNodeInfo(index []byte, headHash []byte) []byte {
	var toret []byte
	toret = append(toret, []byte("BFT")...)
	toret = append(toret, index...)
	toret = append(toret, headHash...)
	return toret
}

// getNode gets the node of an index.
func (it *IndexTree) getNode(tx *sql.Tx, txIndex int) (parsedTx libkataware.Transaction,
	leftChild *int, rightChild *int, err error) {
	var rawTx []byte
	err = tx.QueryRow("SELECT rawTx,leftChild,rightChild FROM indexTree WHERE txIndex=$1",
		txIndex).Scan(&rawTx, &leftChild, &rightChild)
	if err != nil {
		return
	}
	err = parsedTx.FromBytes(rawTx)
	if err != nil {
		panic(err)
	}
	return
}

// getLog finds the log in the addendum corresponding to a certain txIndex.
func (it *IndexTree) getLog(tx *sql.Tx, txIndex int) (txes []libkataware.Transaction, err error) {
	rows, err := tx.Query("SELECT rawTx FROM addendum WHERE parentIdx = $1", txIndex)
	if err != nil {
		return
	}
	for rows.Next() {
		var rawTx []byte
		err = rows.Scan(&rawTx)
		if err != nil {
			return
		}
		var realTx libkataware.Transaction
		err = realTx.FromBytes(rawTx)
		if err != nil {
			panic(err)
		}
		txes = append(txes, realTx)
	}
	return
}

// getConfirm gets the confirmation for a transaction, using cached values if possible
func (it *IndexTree) getConfirm(tx *sql.Tx, parsedTx libkataware.Transaction) (blkidx int,
	merkle [][]byte, posn int, err error) {
	var concatMerkle []byte
	err = tx.QueryRow(
		"SELECT block,merkle,posn FROM txConfCache WHERE txHash=$1", parsedTx.Hash256()).
		Scan(&blkidx, &concatMerkle, &posn)
	if err == nil {
		for i := 0; i < len(concatMerkle)/32; i++ {
			merkle = append(merkle, concatMerkle[:i*32][:32])
		}
		goto END
	}
	// we have to query the network
	blkidx, err = it.eclient.LocateTx(parsedTx.Hash256())
	if err != nil {
		return
	}
	merkle, posn, err = it.eclient.GetMerkle(parsedTx.Hash256(), blkidx)
	if err != nil {
		return
	}
	for _, v := range merkle {
		concatMerkle = append(concatMerkle, v...)
	}
	_, err = tx.Exec("INSERT INTO txConfCache VALUES ($1,$2,$3,$4)",
		parsedTx.Hash256(), blkidx, concatMerkle, posn)
	if err != nil {
		return
	}
END:
	return
}

// findPath finds a path to the "closest" node to a key
func (it *IndexTree) findPath(tx *sql.Tx, targetIndex []byte) (nids []int, err error) {
	nids = []int{0} // root
	// iterate through tree, descending downwards
	for {
		tip := nids[len(nids)-1]
		parsedTx, leftChild, rightChild, e := it.getNode(tx, tip)
		if err != nil {
			err = e
			return
		}
		currIndex, _ := unpackNodeInfo(parsedTx.Outputs[3].Script[1:])
		// compare currIndex to targetIndex
		switch bytes.Compare(targetIndex, currIndex) {
		case -1:
			if leftChild == nil {
				return
			}
			nids = append(nids, *leftChild)
		case 1:
			if rightChild == nil {
				return
			}
			nids = append(nids, *rightChild)
		default:
			return
		}
	}
}

// getFunds tries to obtain funding at least of a given amount of satoshis, returning amt of satoshis actually gotten
func (it *IndexTree) getFunds(tx *sql.Tx, desired int) (txes []libkataware.TxInput,
	totsum int, err error) {
	rows, err := tx.Query("SELECT rawTx,outIdx FROM funding")
	if err != nil {
		return
	}
	defer rows.Close()
	// scan the funds until we have enough satoshis
	for rows.Next() {
		var rawtx []byte
		var outid int
		err = rows.Scan(&rawtx, &outid)
		if err != nil {
			return
		}
		var realtx libkataware.Transaction
		err = realtx.FromBytes(rawtx)
		if err != nil {
			panic("malformed transaction in funding!")
		}
		totsum += int(realtx.Outputs[outid].Value)
		txes = append(txes, libkataware.TxInput{
			PrevHash: realtx.Hash256(),
			PrevIdx:  outid,
			Script:   realtx.Outputs[outid].Script,
			Seqno:    0xffffffff,
		})
	}
	fee := int(it.eclient.EstimateFee(25) * 300)
	// if we didn't gather enough, give up without touching anything
	if totsum < desired+fee {
		err = ErrNoFunds
		return
	}
	// otherwise we create a new transaction, first output funds the request, second output goes back
	unsig := libkataware.Transaction{
		Version: 1,
		Inputs:  txes,
		Outputs: []libkataware.TxOutput{
			libkataware.TxOutput{
				Value:  uint64(desired),
				Script: txes[0].Script,
			},
			libkataware.TxOutput{
				Value:  uint64(totsum - desired - fee),
				Script: txes[0].Script,
			},
		},
	}
	signed, err := it.eclient.SignTx(unsig.ToBytes(), it.privKey)
	if err != nil {
		return
	}
	if it.bcast {
		err = it.eclient.BroadcastTx(signed)
		if err != nil {
			return
		}
	}
	_, err = tx.Exec("DELETE FROM funding")
	if err != nil {
		return
	}
	_, err = tx.Exec("INSERT INTO funding VALUES ($1, $2)", signed, 1)
	if err != nil {
		return
	}
	txes = nil
	txes = append(txes, libkataware.TxInput{
		PrevHash: libkataware.DoubleSHA256(signed),
		PrevIdx:  0,
		Script:   unsig.Outputs[0].Script,
		Seqno:    0xffffffff,
	})
	totsum = desired
	return
}
