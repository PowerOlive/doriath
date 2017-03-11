package operlog

import (
	"bytes"
	crand "crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"math/rand"
	"testing"

	"golang.org/x/crypto/ed25519"
)

func TestSimpleAssembleId(t *testing.T) {
	_, err := AssembleID(".ed25519 d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a\n.ed25519 3d4017c3e843895a92b70aa74d1b7ebc9c982ccf2ec4968cc0cd55f12af4660c\t.quorum 1. 2.")
	if err != nil {
		t.Error(err)
	}
}

func TestAssembleIdInvalidKey(t *testing.T) {
	_, err := AssembleID(".ed25519 xxxx")
	if err == nil {
		t.FailNow()
	}
}

func TestAssembleIdInvalidDirective(t *testing.T) {
	_, err := AssembleID(".mario deadbeef")
	if err == nil {
		t.FailNow()
	}
}

func TestAssembleIdInvalidQuorumX(t *testing.T) {
	_, err := AssembleID(".quorum x. 2.")
	if err == nil {
		t.FailNow()
	}
}

func TestAssembleIdInvalidQuorumY(t *testing.T) {
	_, err := AssembleID(".quorum 1. y.")
	if err == nil {
		t.FailNow()
	}
}

func TestAssembleIdInvalidQuorumXUpperBound(t *testing.T) {
	_, err := AssembleID(".quorum 257. 2.")
	if err == nil {
		t.FailNow()
	}
}

func TestSimpleVerifySuccess(t *testing.T) {
	PK, SK, _ := ed25519.GenerateKey(crand.Reader)

	id, err := AssembleID(
		fmt.Sprintf(".ed25519 %x\n.ed25519 3d4017c3e843895a92b70aa74d1b7ebc9c982ccf2ec4968cc0cd55f12af4660c\t.quorum 1. 2.",
			PK))
	if err != nil {
		t.Error(err)
		return
	}

	log.Printf("result of assembly: %X\n", id)

	sig := ed25519.Sign(SK, nil)
	var sigs = [][]byte{sig, nil}
	err = id.Verify(nil, sigs)
	if err != nil {
		t.Error(err)
	}
}

func TestSimpleVerifyFail(t *testing.T) {
	id, err := AssembleID(".ed25519 d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a\n.ed25519 3d4017c3e843895a92b70aa74d1b7ebc9c982ccf2ec4968cc0cd55f12af4660c\t.quorum 1. 2.")
	if err != nil {
		t.Error(err)
		return
	}

	s1, _ := hex.DecodeString("d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511f")
	var sigs = [][]byte{s1, nil}
	err = id.Verify(nil, sigs)
	if err == nil {
		t.Error(err)
	}
}

func TestGarbageFuzz(t *testing.T) {
	for i := 0; i < 100; i++ {
		// generate a garbage assembly by appending strings
		lol := new(bytes.Buffer)
		for i := 0; i < 50; i++ {
			switch rand.Int() % 10 {
			case 0:
				fmt.Fprintf(lol, ".quorum %v. %v.\n", rand.Int()%3+1, rand.Int()%10+1)
			default:
				lawl := make([]byte, 32)
				crand.Read(lawl)
				fmt.Fprintf(lol, ".ed25519 %x\n", lawl)
			}
		}
		garbage, err := AssembleID(string(lol.Bytes()))
		if err != nil {
			panic(err.Error())
		}
		crand.Read(garbage)
		if garbage.Verify(nil, make([][]byte, 100)) != ErrInvalidID {
			t.FailNow()
		}
	}
}
