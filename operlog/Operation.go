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
	Nonce      []byte
	NextID     IDScript
	Data       string
	Signatures [][]byte
}

func (op *Operation) fixNonce() {
	if len(op.Nonce) != 16 {
		nnc := make([]byte, 16)
		copy(nnc, op.Nonce)
		op.Nonce = nnc
	}
}

// SignedPart extracts the part that's signed by the signatures.
func (op *Operation) SignedPart() []byte {
	op.fixNonce()
	buf := new(bytes.Buffer)

	buf.Write(op.Nonce)

	// 4 bytes: length of next identity script; len return int
	binary.Write(buf, binary.BigEndian, uint32(len(op.NextID)))
	// identity script
	binary.Write(buf, binary.BigEndian, op.NextID)

	// 4 bytes: length of associated data; len returns int
	binary.Write(buf, binary.BigEndian, uint32(len(op.Data)))
	// associated data
	binary.Write(buf, binary.BigEndian, []byte(op.Data))
	return buf.Bytes()
}

// FromBytes is a convenience wrapper around Unpack.
func (op *Operation) FromBytes(b []byte) error {
	nr := bytes.NewReader(b)
	err := op.Unpack(nr)
	if err != nil {
		return err
	}
	if nr.Len() != 0 {
		return ErrInvalidOp
	}
	return nil
}

// ToBytes is a convenience wrapper around Pack.
func (op *Operation) ToBytes() []byte {
	op.fixNonce()
	buf := new(bytes.Buffer)
	op.Pack(buf)
	return buf.Bytes()
}

// Pack serializes an operation to an output.
func (op *Operation) Pack(out io.Writer) error {
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

	_, err := io.Copy(out, buf)
	if err != nil {
		return err
	}
	return nil
}

// Unpack fills an operation struct by deserializing an input.
func (op *Operation) Unpack(in io.Reader) error {
	// Note that op is passed by reference.
	// Hint: make a bytes.Buffer and read from it using binary.Read
	op.fixNonce()
	_, err := io.ReadFull(in, op.Nonce)
	if err != nil {
		return ErrInvalidOp
	}
	var idlen, datalen, siglen uint32
	err = binary.Read(in, binary.BigEndian, &idlen)
	if err != nil {
		return ErrInvalidOp
	}
	if idlen > 1024*32 {
		return ErrInvalidOp
	}
	// init with length idlen
	op.NextID = make([]byte, idlen)
	err = binary.Read(in, binary.BigEndian, &(op.NextID))
	if err != nil {
		return ErrInvalidOp
	}

	err = binary.Read(in, binary.BigEndian, &datalen)
	if err != nil {
		return ErrInvalidOp
	}
	if datalen > 1024*128 {
		return ErrInvalidOp
	}
	// init with length datalen
	zzz := make([]byte, datalen)
	err = binary.Read(in, binary.BigEndian, &zzz)
	if err != nil {
		return ErrInvalidOp
	}
	op.Data = string(zzz)

	err = binary.Read(in, binary.BigEndian, &siglen)
	if err != nil {
		return ErrInvalidOp
	}
	if siglen > 1024*32 {
		return ErrInvalidOp
	}
	// the last siglen bytes contain all the signatures
	sigbuf := make([]byte, siglen)
	_, err = io.ReadFull(in, sigbuf)
	if err != nil {
		return ErrInvalidOp
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
			return ErrInvalidOp
		}
	}
	return nil
}
