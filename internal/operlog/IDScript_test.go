package operlog

import (
	crand "crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
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

func TestAssembleIdInvalidQuorumXLowerBound(t *testing.T) {
	_, err := AssembleID(".quorum 0. 2.")
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

func TestAssembleIdInvalidQuorumYLowerBound(t *testing.T) {
	_, err := AssembleID(".quorum 1. 0.")
	if err == nil {
		t.FailNow()
	}
}

func TestAssembleIdInvalidQuorumYUpperBound(t *testing.T) {
	_, err := AssembleID(".quorum 1. 257.")
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

func TestGarbageID(t *testing.T) {
	for i := 0; i < 1000; i++ {
		garbage := IDScript(make([]byte, 1024))
		crand.Read(garbage)
		if garbage.Verify(nil, make([][]byte, 100)) != ErrInvalidID {
			t.FailNow()
		}
	}
}
