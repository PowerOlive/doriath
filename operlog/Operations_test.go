package operlog

import (
	"encoding/hex"
	"io/ioutil"
	"testing"
)

func TestToBytes(t *testing.T) {
	id, err := AssembleID(".ed25519 d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a\n.ed25519 3d4017c3e843895a92b70aa74d1b7ebc9c982ccf2ec4968cc0cd55f12af4660c\t.quorum 1. 2.")
	if err != nil {
		t.Error(err)
		return
	}

	s1, _ := hex.DecodeString("d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a")
	var sigs = [][]byte{s1}
	op := Operation{NextID: id, Data: string([]byte{1, 2, 3, 4, 5, 6}), Signatures: sigs}

	op.Pack(ioutil.Discard)
}

func TestFromBytes(t *testing.T) {
	id, err := AssembleID(".ed25519 d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a\n.ed25519 3d4017c3e843895a92b70aa74d1b7ebc9c982ccf2ec4968cc0cd55f12af4660c\t.quorum 1. 2.")
	if err != nil {
		t.Error(err)
		return
	}

	s1, _ := hex.DecodeString("d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a")
	var sigs = [][]byte{s1}
	op := Operation{NextID: id, Data: string([]byte{1, 2, 3, 4, 5, 6}), Signatures: sigs}

	source := op.ToBytes()

	another := Operation{}
	err = another.FromBytes(source)
	if err != nil {
		t.Error(err)
		return
	}

	constructed := another.ToBytes()
	if len(source) != len(constructed) {
		t.Error(err)
		return
	}

	for i := range constructed {
		if source[i] != constructed[i] {
			t.Error(err)
			return
		}
	}
}

func TestFromBytesBadIdScriptLength(t *testing.T) {
	source := []byte{1}
	another := Operation{}
	err := another.FromBytes(source)
	if err == nil {
		t.Error(err)
		return
	}
}

func TestFromBytesBadIdScript(t *testing.T) {
	source := []byte{0, 0, 0, 1}
	another := Operation{}
	err := another.FromBytes(source)
	if err == nil {
		t.Error(err)
		return
	}
}

func TestFromBytesBadDataLen(t *testing.T) {
	source := []byte{0, 0, 0, 1, 1, 1}
	another := Operation{}
	err := another.FromBytes(source)
	if err == nil {
		t.Error(err)
		return
	}
}

func TestFromBytesBadData(t *testing.T) {
	source := []byte{0, 0, 0, 1, 1, 0, 0, 0, 1}
	another := Operation{}
	err := another.FromBytes(source)
	if err == nil {
		t.Error(err)
		return
	}
}

func TestFromBytesBadSigLen(t *testing.T) {
	source := []byte{0, 0, 0, 1, 1, 0, 0, 0, 1, 1}
	another := Operation{}
	err := another.FromBytes(source)
	if err == nil {
		t.Error(err)
		return
	}
}

func TestFromBytesBadSlen(t *testing.T) {
	source := []byte{0, 0, 0, 1, 1, 0, 0, 0, 1, 1, 0, 0, 0, 1}
	another := Operation{}
	err := another.FromBytes(source)
	if err == nil {
		t.Error(err)
		return
	}
}

func TestFromBytesBadSig(t *testing.T) {
	source := []byte{0, 0, 0, 1, 1, 0, 0, 0, 1, 1, 0, 0, 0, 1, 0, 1}
	another := Operation{}
	err := another.FromBytes(source)
	if err == nil {
		t.Error(err)
		return
	}
}
