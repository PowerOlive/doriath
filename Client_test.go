package doriath

import (
	"log"
	"net/url"
	"testing"
)

func TestSyncSimple(t *testing.T) {
	u, err := url.Parse("https://bitforest.rensa.io/real-test-na")
	//u, err := url.Parse("http://tirion.rensa.io:8899")
	if err != nil {
		log.Fatal(err)
	}
	client := &Client{GenTx: nil, NaURL: u, CacheDir: "/tmp/bitforest/"}
resync:
	ol, sec, err := client.GetOpLog("_natime")
	if err == ErrOutOfSync {
		log.Println("need to resync!")
		client.Sync()
		log.Println("sync done")
		goto resync
	}
	if err != nil {
		t.Error(err)
		return
	}
	log.Println("oplog done, last is", string(ol.LastData()), "confirmed", sec)
}
