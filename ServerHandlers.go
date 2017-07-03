package doriath

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/rensa-labs/doriath/internal/libkataware"
)

func (srv *Server) handBlockchainHeaders(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	srv.hclock.RLock()
	defer srv.hclock.RUnlock()
	w.Header().Add("cache-control", "max-age=60")
	w.Header().Add("content-length", fmt.Sprintf("%v", len(srv.hdrcache)*80))
	for _, b := range srv.hdrcache {
		w.Write(b)
	}
}

func (srv *Server) handTxchain(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	w.Header().Add("cache-control", fmt.Sprintf("max-age=60"))
	var towrite []struct {
		RawTx    []byte
		BlockIdx interface{}
		PosInBlk interface{}
		Merkle   [][]byte
	}
	dbtx, err := srv.dbHandle.Begin()
	if err != nil {
		log.Println("server: failed to lock db for txchain")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer func() {
		dbtx.Commit()
	}()
	rows, err := dbtx.Query("SELECT rawtx, blkidx, posinblk, merkle FROM txhistory")
	if err != nil {
		log.Println("server: failed selecting rawtx from txhistory")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var toadd struct {
			RawTx    []byte
			BlockIdx interface{}
			PosInBlk interface{}
			Merkle   [][]byte
		}
		var ccmerkle []byte
		err = rows.Scan(&toadd.RawTx, &toadd.BlockIdx, &toadd.PosInBlk, &ccmerkle)
		if err != nil {
			log.Println("server: failed scanning rawtx:", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if ccmerkle == nil {
			txhash := libkataware.DoubleSHA256(toadd.RawTx)
			toadd.BlockIdx, err = srv.btcClient.LocateTx(txhash)
			if err != nil {
				toadd.BlockIdx = -1
				toadd.PosInBlk = 0
				towrite = append(towrite, toadd)
				continue
			}
			var blk []byte
			blk, err = srv.btcClient.GetBlock(toadd.BlockIdx.(int))
			if err != nil {
				log.Println("server: failed locating block from txhistory")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			var fullblock libkataware.Block
			err = fullblock.Deserialize(blk)
			if err != nil {
				panic("Garbage in fullblock?!")
			}
			toadd.Merkle, toadd.PosInBlk = fullblock.GenMerkle(txhash)
			_, err = dbtx.Exec(`UPDATE txhistory SET blkidx = $1,
				 posinblk = $2, merkle = $3 WHERE rawtx = $4`,
				toadd.BlockIdx, toadd.PosInBlk, func() (res []byte) {
					for _, v := range toadd.Merkle {
						res = append(res, v...)
					}
					return
				}(), toadd.RawTx)
			if err != nil {
				log.Println("server: failed to update tx cache:", err.Error())
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		} else {
			buf := bytes.NewReader(ccmerkle)
			for buf.Len() > 0 {
				mkbr := make([]byte, 32)
				buf.Read(mkbr)
				toadd.Merkle = append(toadd.Merkle, mkbr)
			}
		}
		towrite = append(towrite, toadd)
	}
	encoded, err := json.MarshalIndent(towrite, "", "    ")
	if err != nil {
		panic(err)
	}
	w.Header().Add("content-type", "application/json")
	w.Write(encoded)
}

var jsonRgxp = regexp.MustCompile("\\.json$")

func (srv *Server) handOplog(w http.ResponseWriter, r *http.Request) {
	fname := path.Base(r.URL.Path)
	if !jsonRgxp.MatchString(fname) {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	name := strings.Replace(fname, ".json", "", -1)
	startTime := time.Now()
	_, proofs, values, err := srv.forest.FindAllProof(name)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var towrite []struct {
		RawOps []byte
		Proof  [][]byte
	}
	for i := 0; i < len(proofs); i++ {
		var current struct {
			RawOps []byte
			Proof  [][]byte
		}
		for _, op := range values[i] {
			current.RawOps = append(current.RawOps, op.ToBytes()...)
		}
		current.Proof = proofs[i].ToBytes()
		towrite = append(towrite, current)
	}
	log.Println("one search took", time.Now().Sub(startTime))
	// add staging info if possible
	stagOps, err := srv.forest.SearchStaging(name)
	if err == nil {
		var last struct {
			RawOps []byte
			Proof  [][]byte
		}
		for _, o := range stagOps {
			last.RawOps = append(last.RawOps, o.ToBytes()...)
		}
		towrite = append(towrite, last)
	}
	encoded, err := json.MarshalIndent(towrite, "", "    ")
	if err != nil {
		panic(err)
	}
	w.Header().Add("content-type", "application/json")
	w.Header().Add("cache-control", "max-age=10")
	w.Write(encoded)
}
