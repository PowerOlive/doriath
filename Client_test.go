package doriath

import (
	"log"
	"net/url"
	"testing"
)

func TestSyncSimple(t *testing.T) {
	u, err := url.Parse("https://bitforest.rensa.io/mock-na")
	if err != nil {
		log.Fatal(err)
	}

	//genTx := "{\"RawTx\": \"AQAAAAGbYLyckQkeJ0fer2DRod0cvlQwGuXzaK3051svop85QgAAAAAA/////wKgJaTU6AAAAAAQJwAAAAAAABl2qRRsRiTEwv3aAg/jBYeoz3H7gDi2hIisAAAAAA==\",\"BlockIdx\": 1,\"PosInBlk\": 0,\"Merkle\": null}"

	client := &Client{GenTx: nil, NaURL: u, CacheDir: "/tmp/bitforest/"}
	client.Sync()
	log.Println("sync done")
	ol, sec, err := client.GetOpLog("name-1")
	if err != nil {
		t.Error(err)
	}
	log.Println("oplog done, last is", string(ol.LastData()), "confirmed", sec)
}
