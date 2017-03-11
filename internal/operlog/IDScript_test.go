package operlog

import (
	"encoding/hex"
	"testing"
)

func TestSimpleAssembleId(t *testing.T) {
	_, err := AssembleID(".ed25519 d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a\n.ed25519 3d4017c3e843895a92b70aa74d1b7ebc9c982ccf2ec4968cc0cd55f12af4660c\t.quorum 1. 2.")
	if err != nil {
		t.Error("error")
	}
}

func TestAssembleIdInvalidKey(t *testing.T) {
	_, err := AssembleID(".ed25519 xxxx")
	if err == nil {
		t.Error("error")
	}
}

func TestAssembleIdInvalidDirective(t *testing.T) {
	_, err := AssembleID(".mario deadbeef")
	if err == nil {
		t.Error("error")
	}
}

func TestAssembleIdInvalidQuorumX(t *testing.T) {
	_, err := AssembleID(".quorum x. 2.")
	if err == nil {
		t.Error("error")
	}
}

func TestAssembleIdInvalidQuorumY(t *testing.T) {
	_, err := AssembleID(".quorum 1. y.")
	if err == nil {
		t.Error("error")
	}
}

func TestAssembleIdInvalidQuorumXLowerBound(t *testing.T) {
	_, err := AssembleID(".quorum 0. 2.")
	if err == nil {
		t.Error("error")
	}
}

func TestAssembleIdInvalidQuorumXUpperBound(t *testing.T) {
	_, err := AssembleID(".quorum 257. 2.")
	if err == nil {
		t.Error("error")
	}
}

func TestAssembleIdInvalidQuorumYLowerBound(t *testing.T) {
	_, err := AssembleID(".quorum 1. 0.")
	if err == nil {
		t.Error("error")
	}
}

func TestAssembleIdInvalidQuorumYUpperBound(t *testing.T) {
	_, err := AssembleID(".quorum 1. 257.")
	if err == nil {
		t.Error("error")
	}
}

func TestSimpleVerifySuccess(t *testing.T) {
	id, err := AssembleID(".ed25519 d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a\n.ed25519 3d4017c3e843895a92b70aa74d1b7ebc9c982ccf2ec4968cc0cd55f12af4660c\t.quorum 1. 2.")
	if err != nil {
		t.Error("error")
	}

	s1, _ := hex.DecodeString("d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a")
	var sigs = [][]byte{s1}
	err = id.Verify(sigs)
	if err != nil {
		t.Error("error")
	}
}

func TestSimpleVerifyFail(t *testing.T) {
	id, err := AssembleID(".ed25519 d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a\n.ed25519 3d4017c3e843895a92b70aa74d1b7ebc9c982ccf2ec4968cc0cd55f12af4660c\t.quorum 1. 2.")
	if err != nil {
		t.Error("error")
	}

	s1, _ := hex.DecodeString("d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511f")
	var sigs = [][]byte{s1}
	err = id.Verify(sigs)
	if err == nil {
		t.Error("error")
	}
}

func TestSimpleDiffTypesKeys(t *testing.T) {
	id, err := AssembleID(".ed25519 d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a\n.ed25519 3d4017c3e843895a92b70aa74d1b7ebc9c982ccf2ec4968cc0cd55f12af4660c\t.quorum 1. 2.")
	if err != nil {
		t.Error("error")
	}

	s1, _ := hex.DecodeString("d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a")
	s2, _ := hex.DecodeString("deadbeef") // suppose this is a key in different format
	var sigs = [][]byte{s1, s2}
	err = id.Verify(sigs)
	if err != nil {
		t.Error("error")
	}
}
