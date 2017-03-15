package operlog

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
)

// ErrInvalidOp signals and invalid operation
var ErrInvalidOp = errors.New("invalid operation")

// Operation represents a single operation.
type Operation struct {
	NextID     IDScript
	Data       []byte
	Signatures [][]byte
}

// SignedPart extracts the part that's signed by the signatures.
func (op *Operation) SignedPart() []byte {
	buf := new(bytes.Buffer)

	// XXX writing to buf cannot fail since it's not a socket

	// 4 bytes: length of next identity script; len return int
	binary.Write(buf, binary.BigEndian, uint32(len(op.NextID)))
	// identity script
	binary.Write(buf, binary.BigEndian, op.NextID)

	// 4 bytes: length of associated data; len returns int
	binary.Write(buf, binary.BigEndian, uint32(len(op.Data)))
	// associated data
	binary.Write(buf, binary.BigEndian, op.Data)
	return buf.Bytes()
}

// ToBytes serializes an operation to a byte array.
func (op *Operation) ToBytes() []byte {
	// Hint: make a bytes.Buffer and write to it using binary.Write
	buf := new(bytes.Buffer)
	buf.Write(op.SignedPart())

	// 4 bytes: sum of all length of signatures plus the 2 bytes for the length
	var l int
	for i := range op.Signatures {
		l += len(op.Signatures[i]) + 2
	}
	binary.Write(buf, binary.BigEndian, uint32(l))
	// all signatures
	for i := range op.Signatures {
		sig := op.Signatures[i]
		// 2 bytes: length of a signature
		binary.Write(buf, binary.BigEndian, uint16(len(sig)))
		// a signature
		binary.Write(buf, binary.BigEndian, sig)
	}

	return buf.Bytes()
}

// FromBytes fills an operation struct by parsing a byte array.
func (op *Operation) FromBytes(barr []byte) error {
	// Note that op is passed by reference.
	// Hint: make a bytes.Buffer and read from it using binary.Read
	buf := bytes.NewReader(barr)

	var idlen, datalen, siglen uint32
	err := binary.Read(buf, binary.BigEndian, &idlen)
	if err != nil {
		return err
	}
	if idlen > 1024*32 {
		return ErrInvalidOp
	}
	// init with length idlen
	op.NextID = make([]byte, idlen)
	err = binary.Read(buf, binary.BigEndian, &(op.NextID))
	if err != nil {
		return err
	}

	err = binary.Read(buf, binary.BigEndian, &datalen)
	if err != nil {
		return err
	}
	if datalen > 1024*128 {
		return ErrInvalidOp
	}
	// init with length datalen
	op.Data = make([]byte, datalen)
	err = binary.Read(buf, binary.BigEndian, &(op.Data))
	if err != nil {
		return err
	}

	err = binary.Read(buf, binary.BigEndian, &siglen)
	if err != nil {
		return err
	}
	if siglen > 1024*32 {
		return ErrInvalidOp
	}
	// the last siglen bytes contain all the signatures
	sigbuf := make([]byte, siglen)
	_, err = io.ReadFull(buf, sigbuf)
	if err != nil {
		return err
	}
	sbread := bytes.NewReader(sigbuf)
	// nil is an empty array
	op.Signatures = nil
	var slen uint16
	for {
		err = binary.Read(sbread, binary.BigEndian, &slen)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if slen > 1024*2 {
			return ErrInvalidOp
		}
		op.Signatures = append(op.Signatures, make([]byte, slen, slen))
		err = binary.Read(sbread, binary.BigEndian, op.Signatures[len(op.Signatures)-1])
		if err != nil {
			return err
		}
	}
	if buf.Len() != 0 {
		return ErrInvalidOp
	}
	return nil
}
