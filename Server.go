package doriath

import (
	"bytes"
	"database/sql"
	"log"
	"net/http"
	"time"

	"gopkg.in/tomb.v2"

	"github.com/rensa-labs/doriath/internal/libkataware"
	"github.com/rensa-labs/doriath/internal/sqliteforest"
	"github.com/rensa-labs/doriath/operlog"
)

// Server represents a Bitforest server.
type Server struct {
	btcClient  BitcoinClient
	btcPrivKey string // WIF
	funding    libkataware.Transaction
	interval   time.Duration
	forest     *sqliteforest.Forest
	dbHandle   *sql.DB

	smux *http.ServeMux

	death tomb.Tomb
}

// background routine for the Server
func (srv *Server) bkgRoutine() error {
	for cnt := 0; ; cnt++ {
		nxtCommit := time.After(srv.interval)
		select {
		case <-srv.death.Dying():
			log.Println("server: dying due to tomb:", srv.death.Err())
			return srv.death.Err()
		case <-nxtCommit:
			log.Println("server: committing staging area", cnt)
			for {
				err := srv.forest.Commit()
				if err != nil {
					log.Println("server: committing staging area", cnt,
						"FAILED:", err.Error())
					time.Sleep(time.Second)
				} else {
					break
				}
			}
			for {
				if err := srv.syncChain(); err != nil {
					log.Println("server: syncing chain", cnt, "FAILED:", err.Error())
					time.Sleep(time.Second)
				} else {
					break
				}
			}
		}
	}
}

// ServeHTTP implements the http.Handler interface, and responds according to the standard REST-based protocol.
func (srv *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	srv.smux.ServeHTTP(w, r)
}

// StageOperation stages an operation for a name.
// TODO: no sanity checking currently done!!!
func (srv *Server) StageOperation(name string, operation operlog.Operation) error {
	err := srv.forest.Stage(name, operation)
	return err
}

// NewServer creates a new Bitforest server with the given parameters.
func NewServer(btcClient BitcoinClient,
	btcPrivKey string,
	fundingTx []byte,
	interval time.Duration,
	dbPath string) (srv *Server, err error) {
	srv = &Server{
		btcClient:  btcClient,
		btcPrivKey: btcPrivKey,
		interval:   interval,
		smux:       http.NewServeMux(),
	}
	err = srv.funding.Unpack(bytes.NewReader(fundingTx))
	if err != nil {
		log.Println("could not unpack funding tx")
		return
	}
	srv.dbHandle, err = sql.Open("sqlite3_with_fk", dbPath)
	if err != nil {
		return
	}
	srv.forest, err = sqliteforest.OpenForest(dbPath)
	if err != nil {
		return
	}
	// initialize db
	_, err = srv.dbHandle.Exec(`CREATE TABLE IF NOT EXISTS txhistory (
						rhash BLOB NOT NULL, --- not FK since possibly not unique
						rawtx BLOB NOT NULL)`)
	if err != nil {
		return
	}
	srv.death.Go(srv.bkgRoutine)
	srv.smux.HandleFunc("/blockchain_headers", srv.handBlockchainHeaders)
	srv.smux.HandleFunc("/txchain.json", srv.handTxchain)
	srv.smux.HandleFunc("/oplogs/", srv.handOplog)
	return
}
