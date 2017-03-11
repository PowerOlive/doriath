package operlog

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// Operation represents a single operation.
type Operation struct {
	NextID     IDScript
	Data       []byte
	Signatures [][]byte
}

// ToBytes serializes an operation to a byte array.
func (op *Operation) ToBytes() []byte {
	// Hint: make a bytes.Buffer and write to it using binary.Write
	buf := new(bytes.Buffer)

	// 4 bytes: length of next identity script; len return int
	err := binary.Write(buf, binary.BigEndian, uint32(len(op.NextID)))
	if err != nil {
		fmt.Println("failed id_len: ", err)
		return nil
	}
	// identity script
	err = binary.Write(buf, binary.BigEndian, op.NextID)
	if err != nil {
		fmt.Println("failed id: ", err)
		return nil
	}

	// 4 bytes: length of associated data; len returns int
	err = binary.Write(buf, binary.BigEndian, uint32(len(op.Data)))
	if err != nil {
		fmt.Println("failed data_len: ", err)
		return nil
	}
	// associated data
	err = binary.Write(buf, binary.BigEndian, op.Data)
	if err != nil {
		fmt.Println("failed data: ", err)
		return nil
	}

	// 4 bytes: sum of all length of signatures
	var l int
	for i := range op.Signatures {
		l += len(op.Signatures[i])
	}
	err = binary.Write(buf, binary.BigEndian, uint32(l))
	if err != nil {
		fmt.Println("failed sig_all_len: ", err)
		return nil
	}
	// all signatures
	for i := range op.Signatures {
		sig := op.Signatures[i]
		// 2 bytes: length of a signature
		err = binary.Write(buf, binary.BigEndian, uint16(len(sig)))
		if err != nil {
			fmt.Println("failed sig_len: ", err)
			return nil
		}
		// a signature
		err = binary.Write(buf, binary.BigEndian, sig)
		if err != nil {
			fmt.Println("failed sign: ", err)
			return nil
		}
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
		fmt.Println("failed to read idlen: ", err)
		return err
	}
	// init with length idlen
	op.NextID = make([]byte, idlen, idlen)
	err = binary.Read(buf, binary.BigEndian, &(op.NextID))
	if err != nil {
		fmt.Println("failed to read id: ", err)
		return err
	}

	err = binary.Read(buf, binary.BigEndian, &datalen)
	if err != nil {
		fmt.Println("failed to read datalen: ", err)
		return err
	}
	// init with length datalen
	op.Data = make([]byte, datalen, datalen)
	err = binary.Read(buf, binary.BigEndian, &(op.Data))
	if err != nil {
		fmt.Println("failed to read data: ", err)
		return err
	}

	err = binary.Read(buf, binary.BigEndian, &siglen)
	if err != nil {
		fmt.Println("failed to read siglen: ", err)
		return err
	}

	op.Signatures = make([][]byte, 0, 1)
	var slen uint16
	for siglen > 0 {
		err = binary.Read(buf, binary.BigEndian, &slen)
		if err != nil {
			fmt.Println("failed to read a slen: ", err)
			return err
		}
		op.Signatures = append(op.Signatures, make([]byte, slen, slen))
		err = binary.Read(buf, binary.BigEndian, op.Signatures[len(op.Signatures)-1])
		if err != nil {
			fmt.Println("failed to read a sig: ", err)
			return err
		}
		siglen -= uint32(slen)
	}

	return nil
}
