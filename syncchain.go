package doriath

import (
	"bytes"
	"encoding/hex"
	"log"

	"github.com/rensa-labs/doriath/internal/libkataware"
)

// sync with blockchain
func (srv *Server) syncChain() error {
	tran, err := srv.dbHandle.Begin()
	if err != nil {
		log.Println("server: syncchain failed to start tx:", err.Error())
		return err
	}
	defer tran.Rollback()
	var tn libkataware.Transaction
	var txcount int
	err = tran.QueryRow("SELECT COUNT(*) FROM txhistory").Scan(&txcount)
	if err != nil {
		log.Println("server: error while selecting txcount:", err.Error())
		return err
	}
	if txcount == 0 {
		tn = srv.funding
	} else {
		var rawtn []byte
		err = tran.QueryRow("SELECT rawtx FROM txhistory ORDER BY ROWID DESC LIMIT 1").Scan(&rawtn)
		if err != nil {
			log.Println("server: error while getting rawtn:", err.Error())
			return err
		}
		err = tn.Unpack(bytes.NewReader(rawtn))
		if err != nil {
			panic("unpacked rawtn but found garbage!!!")
		}
	}
	fee := uint64(50000)
	if tn.Outputs[0].Value <= fee+10000 {
		// TODO do something intelligent
		panic("OUT OF $$$")
	}
	toScript := func(bts []byte) []byte {
		if len(bts) != 20 {
			panic("not 160 bits")
		}
		toret, e := hex.DecodeString("76A914" + hex.EncodeToString(bts) + "88AC")
		if e != nil {
			panic("WAT")
		}
		return toret
	}
	// select all uncommitted tree roots
	rowz, err := tran.Query(`SELECT rhash FROM treeroots WHERE
		rhash NOT IN (SELECT rhash FROM txhistory)`)
	if err != nil {
		return err
	}
	var tocommit [][]byte
	for rowz.Next() {
		var one []byte
		err = rowz.Scan(&one)
		if err != nil {
			return err
		}
		tocommit = append(tocommit, one)
	}
	// loop through the uncommitted tree roots, broadcasting txes into the blockchain
	for _, val := range tocommit {
		// build a new transaction based on tn
		tnp1 := libkataware.Transaction{
			Version: 1,
			Inputs: []libkataware.TxInput{
				libkataware.TxInput{
					PrevHash: tn.Hash256(),
					PrevIdx:  0,
					Script:   tn.Outputs[0].Script, // will be overwritten with real sig
					Seqno:    0xffffffff,
				},
			},
			Outputs: []libkataware.TxOutput{
				libkataware.TxOutput{
					Value:  tn.Outputs[0].Value - 10000 - fee,
					Script: tn.Outputs[0].Script,
				},
				libkataware.TxOutput{
					Value:  10000,
					Script: toScript(val[:20]),
				},
			},
		}
		signed, err := srv.btcClient.SignTx(tnp1.ToBytes(), srv.btcPrivKey)
		if err != nil {
			log.Printf("server: failed to sign %x", tnp1.ToBytes())
			return err
		}
		if tn.FromBytes(signed) != nil {
			panic("WAT")
		}
		err = srv.btcClient.BroadcastTx(signed)
		if err != nil {
			return err
		}
		_, err = tran.Exec("INSERT INTO txhistory VALUES ($1, $2)", val, tn.ToBytes())
		if err != nil {
			return err
		}
	}
	return tran.Commit()
}
