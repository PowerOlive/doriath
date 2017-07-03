package doriath

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"

	"github.com/rensa-labs/doriath/internal/libkataware"
)

var errNoFunds = errors.New("insufficient funds")

// sync with blockchain
func (srv *Server) syncChain() error {
	const mBTC = 100000
	tran, err := srv.dbHandle.Begin()
	if err != nil {
		log.Println("server: syncchain failed to start tx:", err.Error())
		return err
	}
	defer tran.Rollback()
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
		// first find funding
		var fundsum uint64
		var funding []libkataware.TxInput
		prevtx, err := func() (tn libkataware.Transaction, err error) {
			var rawtn []byte
			err = tran.QueryRow("SELECT rawtx FROM txhistory ORDER BY ROWID DESC LIMIT 1").Scan(&rawtn)
			if err != nil {
				log.Println("server: error while getting rawtn:", err.Error())
				var cnt int
				e := tran.QueryRow("SELECT count(rawtx) FROM txhistory").Scan(&cnt)
				if e != nil || cnt != 0 {
					panic("Already have txhistory but cannot find last? Corrupt DB?!")
				}
				return
			}
			err = tn.Unpack(bytes.NewReader(rawtn))
			if err != nil {
				panic("unpacked rawtn but found garbage!!!")
			}
			return
		}()
		if err == nil {
			// fund with the previous transaction
			funding = append(funding, libkataware.TxInput{
				PrevHash: prevtx.Hash256(),
				PrevIdx:  0,
				Script:   prevtx.Outputs[0].Script, // will be overwritten with real sig
				Seqno:    0xffffffff,
			})
			fundsum += prevtx.Outputs[0].Value
		}
		// also fund with everything in funds
		rows, err := tran.Query("SELECT rawtx FROM funds WHERE spent = 0")
		if err != nil {
			return err
		}
		for rows.Next() {
			var rtx []byte
			rows.Scan(&rtx)
			var tn libkataware.Transaction
			err = tn.Unpack(bytes.NewReader(rtx))
			if err != nil {
				panic("unpacked rtx but found garbage!!!")
			}
			funding = append(funding, libkataware.TxInput{
				PrevHash: tn.Hash256(),
				PrevIdx:  0,
				Script:   tn.Outputs[0].Script, // will be overwritten with real sig
				Seqno:    0xffffffff,
			})
			fundsum += tn.Outputs[0].Value
		}
		_, err = tran.Exec("UPDATE funds SET spent = 1")
		if err != nil {
			return err
		}
		if fundsum <= 10000 {
			return errNoFunds
		}
		toScript := func(bts []byte) []byte {
			if len(bts) != 20 {
				panic("not 160 bits")
			}
			toret, _ := hex.DecodeString("76A914" + hex.EncodeToString(bts) + "88AC")
			return toret
		}
		// build a new transaction based on tn
		tnp1 := libkataware.Transaction{
			Version: 1,
			Inputs:  funding,
			Outputs: []libkataware.TxOutput{
				libkataware.TxOutput{
					Value:  fundsum - 10000,
					Script: funding[0].Script,
				},
				libkataware.TxOutput{
					Value:  10000,
					Script: toScript(val[:20]),
				},
			},
		}
		fee := uint64(400 * (100 + len(tnp1.ToBytes())))
		if fundsum <= 10000+fee {
			return errNoFunds
		}
		log.Printf("%x", tnp1.ToBytes())
		log.Printf("%x", funding[0].Script)
		log.Println("server: fee is", float64(fee)/float64(mBTC), "mBTC, length", len(tnp1.ToBytes()), "remaining",
			float64(fundsum)/float64(mBTC), "mBTC")
		tnp1.Outputs[0].Value -= fee
		signed, err := srv.btcClient.SignTx(tnp1.ToBytes(), srv.btcPrivKey)
		if err != nil {
			log.Printf("server: failed to sign %x", tnp1.ToBytes())
			return err
		}
		log.Println("**** BROADCAST ****")
		bts, _ := json.MarshalIndent(tnp1, "", "    ")
		log.Println(string(bts))
		err = srv.btcClient.BroadcastTx(signed)
		if err != nil {
			return err
		}
		_, err = tran.Exec("INSERT INTO txhistory (rhash, rawtx) VALUES ($1, $2)", val, signed)
		if err != nil {
			return err
		}
	}
	return tran.Commit()
}
