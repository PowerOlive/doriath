package main

import (
	crand "crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"golang.org/x/crypto/ed25519"

	"github.com/rensa-labs/doriath"
	"github.com/rensa-labs/doriath/internal/libkataware"
	"github.com/rensa-labs/doriath/operlog"
)

func garbageLoop(srv *doriath.Server) {
	for i := 0; ; i++ {
		pk, sk, err := ed25519.GenerateKey(crand.Reader)
		if err != nil {
			panic(err)
		}
		idsc := fmt.Sprintf(".ed25519 %x .quorum 1. 1.", pk)
		idscBin, err := operlog.AssembleID(idsc)
		if err != nil {
			panic(err)
		}
		name := fmt.Sprintf("name-%v", i)
		newop := operlog.Operation{
			Nonce:  make([]byte, 16),
			NextID: idscBin,
			Data:   fmt.Sprintf("garbage-data-%v", name),
		}
		crand.Read(newop.Nonce)
		signature := ed25519.Sign(sk, newop.SignedPart())
		newop.Signatures = [][]byte{signature}
		srv.StageOperation(name, newop)
		time.Sleep(time.Millisecond * 100)
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())
	mbc, bogus := doriath.NewMockBitcoinClient()
	log.Printf("%x", bogus)
	var waa libkataware.Transaction
	err := waa.FromBytes(bogus)
	if err != nil {
		panic("waaaa")
	}
	bts, _ := json.MarshalIndent(waa, "", "    ")
	fmt.Println(string(bts))
	srv, err := doriath.NewServer(mbc,
		"foobar",
		time.Second*10,
		fmt.Sprintf("file::memory:?cache=shared"))
	if err != nil {
		panic(err.Error())
	}
	hserv := &http.Server{
		Addr:           "127.0.0.1:18888",
		Handler:        srv,
		MaxHeaderBytes: 1024 * 4,
		ReadTimeout:    time.Second * 2,
	}
	log.Println("MOCK SERVER STARTED at 127.0.0.1:18888, point nginx here")
	go garbageLoop(srv)
	go func() {
		for {
			bogusTx := libkataware.Transaction{
				Version: 1,
				Inputs: []libkataware.TxInput{
					libkataware.TxInput{PrevHash: make([]byte, 32)},
				},
				Outputs: []libkataware.TxOutput{
					libkataware.TxOutput{
						Value:  100000000,
						Script: make([]byte, 32),
					},
				},
			}
			crand.Read(bogusTx.Inputs[0].PrevHash)
			srv.AddFunds(bogusTx.ToBytes())
			time.Sleep(time.Second * 10)
		}
	}()
	err = hserv.ListenAndServe()
	if err != nil {
		panic(err.Error())
	}
}
