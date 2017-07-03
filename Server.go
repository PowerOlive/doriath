package doriath

import (
	crand "crypto/rand"
	"database/sql"
	"log"
	"net/http"
	"sync"
	"time"

	"gopkg.in/tomb.v2"

	"github.com/rensa-labs/doriath/internal/sqliteforest"
	"github.com/rensa-labs/doriath/operlog"
)

// Server represents a Bitforest server.
type Server struct {
	btcClient  BitcoinClient
	btcPrivKey string // WIF
	interval   time.Duration
	forest     *sqliteforest.Forest
	dbHandle   *sql.DB

	hdrcache [][]byte
	hclock   sync.RWMutex

	smux *http.ServeMux

	death tomb.Tomb
}

// force sync for the Server
func (srv *Server) syncHeaders() error {
	srv.hclock.Lock()
	defer srv.hclock.Unlock()
	if len(srv.hdrcache) > 100 {
		srv.hdrcache = srv.hdrcache[:len(srv.hdrcache)-101]
	}
	// sync from blockchain
	curblcount, err := srv.btcClient.GetBlockCount()
	if err != nil {
		return err
	}
	curlen := len(srv.hdrcache)
	workcount := 15
	workchan := make(chan int)
	tmb := new(tomb.Tomb)
	for i := 0; i < workcount; i++ {
		tmb.Go(func() error {
			for {
				var todo int
				var ok bool
				select {
				case <-tmb.Dying():
					return tmb.Err()
				case todo, ok = <-workchan:
					if !ok {
						return nil
					}
				}
				hdr, e := srv.btcClient.GetHeader(todo)
				if err != nil {
					log.Println("server: unexpected error in GetHeader:", e.Error())
					return e
				}
				srv.hdrcache[todo] = hdr
			}
		})
	}
	srv.hdrcache = append(srv.hdrcache, make([][]byte, curblcount-curlen)...)
	for i := curlen; i < curblcount; i++ {
		if i%10000 == 0 {
			log.Println("syncing headers", int(100.0*float64(i)/float64(curblcount)), "%")
		}
		select {
		case workchan <- i:
		case <-tmb.Dying():
			goto OUT
		}
	}
	close(workchan)
OUT:
	err = tmb.Wait()
	if err != nil {
		srv.hdrcache = srv.hdrcache[:curlen]
		return err
	}
	return nil
}

func (srv *Server) dummyOp() operlog.Operation {
	idsc, err := operlog.AssembleID(".quorum 0. 0.")
	if err != nil {
		panic(err.Error())
	}
	newop := operlog.Operation{
		Nonce:  make([]byte, 16),
		NextID: idsc,
		Data:   time.Now().String(),
	}
	crand.Read(newop.Nonce)
	return newop
}

// background routine for the Server
func (srv *Server) bkgRoutine() error {
	for cnt := 0; ; cnt++ {
		nextUnrounded := time.Now().Add(srv.interval).UnixNano()
		nextRounded := (nextUnrounded / srv.interval.Nanoseconds()) * srv.interval.Nanoseconds()
		realnext := time.Unix(0, nextRounded)
		log.Println("server: NEXT PUBLISH IS", realnext)
		nxtCommit := time.After(realnext.Sub(time.Now()))
		select {
		case <-srv.death.Dying():
			log.Println("server: dying due to tomb:", srv.death.Err())
			return srv.death.Err()
		case <-nxtCommit:
			log.Println("server: committing staging area", cnt)
			for {
				err := srv.forest.Stage("_natime", srv.dummyOp())
				if err != nil {
					log.Println("server: adding natime", cnt, "FAILED:", err.Error())
					continue
				}
				err = srv.forest.Commit()
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

// GetOperations obtains all the operations for a name, both staging and confirmed.
func (srv *Server) GetOperations(name string) (ops operlog.OperLog, err error) {
	return srv.forest.FindOperations(name)
}

// StageOperation stages an operation for a name.
func (srv *Server) StageOperation(name string, operation operlog.Operation) error {
	err := srv.forest.Stage(name, operation)
	return err
}

// AddFunds adds funds from the first output of the given transaction. TODO does not do ANY sanity checking!
func (srv *Server) AddFunds(tx []byte) error {
	_, err := srv.dbHandle.Exec("INSERT INTO funds VALUES ($1, 0)", tx)
	return err
}

// NewServer creates a new Bitforest server with the given parameters. Set btcClient to null to use simply as an interface to the database.
func NewServer(btcClient BitcoinClient,
	btcPrivKey string,
	interval time.Duration,
	dbPath string) (srv *Server, err error) {
	srv = &Server{
		btcClient:  btcClient,
		btcPrivKey: btcPrivKey,
		interval:   interval,
		smux:       http.NewServeMux(),
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
						rawtx BLOB NOT NULL,
						blkidx INTEGER,
						posinblk INTEGER,
						merkle BLOB)`)
	if err != nil {
		return
	}
	_, err = srv.dbHandle.Exec(`CREATE TABLE IF NOT EXISTS funds (
		rawtx BLOB PRIMARY KEY,
		spent INTEGER)`)
	if err != nil {
		return
	}
	if btcClient != nil {
		srv.death.Go(func() error {
			for {
				//log.Println("server: syncing headers...")
				err := srv.syncHeaders()
				if err != nil {
					log.Println("server: syncing headers failed! trying next time")
				}
				select {
				case <-srv.death.Dying():
					return srv.death.Err()
				case <-time.After(time.Minute):
				}
			}
		})
		srv.death.Go(srv.bkgRoutine)
		srv.smux.HandleFunc("/blockchain_headers", srv.handBlockchainHeaders)
		srv.smux.HandleFunc("/txchain.json", srv.handTxchain)
		srv.smux.HandleFunc("/oplogs/", srv.handOplog)
	}
	return
}
