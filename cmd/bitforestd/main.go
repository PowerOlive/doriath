package main

import (
	"encoding/hex"
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/rensa-labs/doriath"
)

var glstate struct {
	srv    *doriath.Server
	logger *log.Logger
}

func main() {
	glstate.logger = log.New(os.Stderr, "", log.LstdFlags)
	apiaddr := flag.String("apiaddr", "127.0.0.1:8899", "host and port for incoming clients")
	btcaddr := flag.String("btcaddr", "localhost:8332", "location of Bitcoin Core RPC server")
	btcuser := flag.String("btcuser", "user", "username for Bitcoin Core RPC")
	btcpwd := flag.String("btcpwd", "pwd", "password for Bitcoin Core RPC")
	txinterval := flag.Int("txinterval", 86400, "time in seconds between transactions")
	dbloc := flag.String("dbloc", "DORIATH-SERVER.db", "location of database")
	initfund := flag.String("initfund", "", "initial funding, given as a transaction hexcode")
	btcpriv := flag.String("btcpriv", "", "Bitcoin private key, WIF format")
	flag.Parse()
	if *initfund == "" {
		glstate.logger.Println("initfund must be given")
		return
	}
	if *btcpriv == "" {
		glstate.logger.Println("btcpriv must be given")
		return
	}

	btcclient := doriath.NewBitcoinCoreClient(*btcaddr, *btcuser, *btcpwd)
	res, err := btcclient.GetBlockCount()
	if err != nil {
		glstate.logger.Println("error connecting to Bitcoin Core:", err.Error())
		return
	}
	itxbts, err := hex.DecodeString(*initfund)
	if err != nil {
		glstate.logger.Println("could not decode itxhash:", err.Error())
		return
	}
	log.Println("successfully connected to the Bitcoin network:", res, "blocks")
	glstate.srv, err = doriath.NewServer(btcclient,
		*btcpriv, time.Second*time.Duration(*txinterval), *dbloc)
	glstate.srv.AddFunds(itxbts)
	if err != nil {
		glstate.logger.Println("failed to construct new server:", err.Error())
		return
	}
	hserv := &http.Server{
		Addr:           *apiaddr,
		Handler:        glstate.srv,
		MaxHeaderBytes: 1024 * 4,
		ReadTimeout:    time.Second * 2,
	}
	go func() {
		time.Sleep(time.Second)
		log.Println("STARTED SERVER at", *apiaddr)
	}()
	err = hserv.ListenAndServe()
	if err != nil {
		glstate.logger.Println("error starting API server:", err.Error())
		return
	}
}
