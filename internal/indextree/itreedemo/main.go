package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/rensa-labs/doriath/electrumclient"
	"github.com/rensa-labs/doriath/internal/indextree"
)

type LookupResult struct {
	Ops  [][]byte
	Txes []indextree.ConfirmedTx
}

func main() {
	itree, err := indextree.NewIndexTree("DEMO-INDEX-TREE.sqlite3",
		"cV4A65Y1Vr3oLx1wh1dp2yfAcazFwTaoapVu7Z3gWJsTiik3yWR7",
		electrumclient.NewElectrumClient("electrum.akinbo.org:51001"), false)
	if err != nil {
		panic(err)
	}
	for err = itree.Initialize(); err == indextree.ErrNoFunds; err = itree.Initialize() {
		log.Println("No funds...")
		time.Sleep(time.Second)
	}
	if err != nil {
		panic(err)
	}
	for i := 0; ; i++ {
		name := fmt.Sprintf("name-%v", i)
		/*log.Println("inserting", name)
		id, err := operlog.AssembleID(".quorum 0. 0.")
		if err != nil {
			panic(err)
		}
		newOp := operlog.Operation{
			Nonce:  make([]byte, 32),
			NextID: id,
		}
		rand.Read(newOp.Nonce)
		err = itree.Insert(name, newOp)
		if err != nil {
			panic(err)
		}*/
		log.Println("looking up", name)
		ops, txes, err := itree.Lookup(name)
		if err != nil {
			panic(err)
		}
		var lres LookupResult
		for _, v := range ops {
			lres.Ops = append(lres.Ops, v.ToBytes())
		}
		lres.Txes = txes
		lol, _ := json.Marshal(lres)
		log.Println("length is", len(lol))
	}
}
