package libkataware

import (
	"bytes"
	"math/rand"
	"testing"
	"time"
)

func TestSHA256(t *testing.T) {
	DoubleSHA256([]byte(""))
}

func TestVarint(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	var tvect []uint64
	for i := 0; i < 1000; i++ {
		lol := uint64(rand.Uint32())<<32 + uint64(rand.Uint32())
		switch i % 4 {
		case 0:
			tvect = append(tvect, lol%256)
		case 1:
			tvect = append(tvect, uint64(uint16(lol)))
		case 2:
			tvect = append(tvect, uint64(uint32(lol)))
		case 3:
			tvect = append(tvect, lol)
		}
	}
	buf := new(bytes.Buffer)
	for _, v := range tvect {
		WriteVarint(buf, v)
	}
	for _, v := range tvect {
		val, err := ReadVarint(buf)
		if err != nil || val != v {
			t.FailNow()
		}
	}
	_, err := ReadVarint(buf)
	if err == nil {
		t.FailNow()
	}
}
