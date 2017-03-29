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

	client := &Client{GenTx: nil, NaURL: u, CacheDir: "/Users/w3kim/tmp/bitforest"}
	client.Sync()
	log.Println("sync done")
	client.GetOpLog("name-1")
}
