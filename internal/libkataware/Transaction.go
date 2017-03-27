package libkataware

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
)

// Transaction is a Bitcoin transaction.
type Transaction struct {
	Version  int
	Inputs   []TxInput
	Outputs  []TxOutput
	LockTime uint32
}

// Hash256 returns the double-SHA256 hash of the transaction.
func (tx *Transaction) Hash256() []byte {
	buf := new(bytes.Buffer)
	tx.Pack(buf)
	return DoubleSHA256(buf.Bytes())
}

// ToBytes is a convenience function.
func (tx *Transaction) ToBytes() []byte {
	buf := new(bytes.Buffer)
	tx.Pack(buf)
	return buf.Bytes()
}

// FromBytes is a convenience function.
func (tx *Transaction) FromBytes(b []byte) error {
	return tx.Unpack(bytes.NewReader(b))
}

// Pack serializes a transaction.
func (tx *Transaction) Pack(out io.Writer) error {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, uint32(tx.Version))
	WriteVarint(buf, uint64(len(tx.Inputs)))
	for _, txi := range tx.Inputs {
		txi.Pack(buf)
	}
	WriteVarint(buf, uint64(len(tx.Outputs)))
	for _, txo := range tx.Outputs {
		txo.Pack(buf)
	}
	binary.Write(buf, binary.LittleEndian, tx.LockTime)
	_, err := io.Copy(out, buf)
	return err
}

// Unpack deserializes a transaction.
func (tx *Transaction) Unpack(in io.Reader) error {
	var u32 uint32
	err := binary.Read(in, binary.LittleEndian, &u32)
	if err != nil {
		return err
	}
	tx.Version = int(u32)
	numin, err := ReadVarint(in)
	if err != nil {
		return err
	}
	if numin > 128*1024 {
		return errors.New("unreasonable numin")
	}
	tx.Inputs = make([]TxInput, numin)
	for i := range tx.Inputs {
		err = tx.Inputs[i].Unpack(in)
		if err != nil {
			return err
		}
	}
	numout, err := ReadVarint(in)
	if err != nil {
		return err
	}
	if numout > 128*1024 {
		return errors.New("unreasonable numout")
	}
	tx.Outputs = make([]TxOutput, numout)
	for i := range tx.Outputs {
		err = tx.Outputs[i].Unpack(in)
		if err != nil {
			return err
		}
	}
	err = binary.Read(in, binary.LittleEndian, &tx.LockTime)
	if err != nil {
		return err
	}
	return nil
}

// TxInput is an input to a transaction.
type TxInput struct {
	PrevHash []byte
	PrevIdx  int
	Script   []byte
	Seqno    uint32
}

// Pack packs a transaction input.
func (txi *TxInput) Pack(out io.Writer) error {
	buf := new(bytes.Buffer)
	buf.Write(txi.PrevHash)
	binary.Write(buf, binary.LittleEndian, uint32(txi.PrevIdx))
	WriteVarint(buf, uint64(len(txi.Script)))
	buf.Write(txi.Script)
	binary.Write(buf, binary.LittleEndian, txi.Seqno)
	_, err := io.Copy(out, buf)
	return err
}

// Unpack unpacks a transaction input.
func (txi *TxInput) Unpack(in io.Reader) error {
	txi.PrevHash = make([]byte, 32)
	_, err := io.ReadFull(in, txi.PrevHash)
	if err != nil {
		return err
	}
	var u32 uint32
	err = binary.Read(in, binary.LittleEndian, &u32)
	if err != nil {
		return err
	}
	txi.PrevIdx = int(u32)
	scrlen, err := ReadVarint(in)
	if err != nil {
		return err
	}
	if scrlen > 128*1024 {
		return errors.New("unreasonable scrlen")
	}
	txi.Script = make([]byte, scrlen)
	_, err = io.ReadFull(in, txi.Script)
	if err != nil {
		return err
	}
	err = binary.Read(in, binary.LittleEndian, &txi.Seqno)
	return err
}

// TxOutput is an output to a transaction.
type TxOutput struct {
	Value  uint64
	Script []byte
}

// Pack packs a transaction output.
func (txo *TxOutput) Pack(out io.Writer) error {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, txo.Value)
	WriteVarint(buf, uint64(len(txo.Script)))
	buf.Write(txo.Script)
	_, err := io.Copy(out, buf)
	return err
}

// Unpack unpacks a transaction output.
func (txo *TxOutput) Unpack(in io.Reader) error {
	err := binary.Read(in, binary.LittleEndian, &txo.Value)
	if err != nil {
		return err
	}
	scrlen, err := ReadVarint(in)
	if err != nil {
		return err
	}
	if scrlen > 128*1024 {
		return errors.New("unreasonable scrlen")
	}
	txo.Script = make([]byte, scrlen)
	_, err = io.ReadFull(in, txo.Script)
	return err
}
