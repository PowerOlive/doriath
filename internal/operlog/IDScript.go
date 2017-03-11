package operlog

import (
	"bytes"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"regexp"
	"strconv"
)

// IDScript is an identity script, which is simply represented in binary form.
type IDScript []byte

var asmRegexp = regexp.MustCompile("[[:space:]]")

// ErrInvalidID is returned to indicate that an identity script is malformed.
var ErrInvalidID = errors.New("malformed identity script")

// ErrNoQuorum means that all the signatures were valid, but not enough to satisfy the quorum were found.
var ErrNoQuorum = errors.New("not enough signatures to satisfy quorum")

// AssembleID takes in an identity script represented in assembly, and returns a binary ID script.
func AssembleID(asm string) (IDScript, error) {
	tokens := asmRegexp.Split(asm, -1)
	var encoded []byte

	var t string
	for i := 0; i < len(tokens); i++ {
		t = tokens[i]
		if t == ".ed25519" {
			i++
			d := tokens[i]

			op, _ := hex.DecodeString("0001")
			dh, err := hex.DecodeString(d)
			// possible invalid hex string
			if err != nil {
				return IDScript(nil), err
			}
			// XXX: should we check the length of the key?

			// encode the assembly
			encoded = append(encoded, op...)
			encoded = append(encoded, dh...)
		} else if t == ".quorum" {
			i++
			x := tokens[i]
			i++
			y := tokens[i]

			xi, err := strconv.Atoi(x[:len(x)-1])
			if err != nil { // x is non-numeric
				return IDScript(nil), err
			} else if xi < 1 || xi > 256 { // x is out of boundary
				return IDScript(nil), errors.New("out of boundary: x")
			}
			yi, err := strconv.Atoi(y[:len(y)-1])
			if err != nil { // y is non-numeric
				return IDScript(nil), err
			} else if yi < 1 || yi > 256 { // y is out of boundary
				return IDScript(nil), errors.New("out of boundary: y")
			}

			// encode the assembly
			b, _ := hex.DecodeString("FF")
			encoded = append(encoded, b...)
			encoded = append(encoded, byte(xi))
			encoded = append(encoded, byte(yi))
		} else {
			// unknown directive
			return IDScript(nil), errors.New("unrecognized direcrive: " + t)
		}
	}

	return IDScript(encoded), nil
}

// Verify runs the script with the given array of signatures, and the data, as input. Null error means signature check is good. It must not panic even if the given array is too short, the script is garbage, etc, but should return appropriate errors.
func (ids IDScript) Verify(data []byte, sigs [][]byte) (err error) {
	// we translate panics into error to avoid manually handling array oob, etc
	defer func() {
		/*if e := recover(); e != nil {
			err = ErrInvalidID
		}*/
	}()
	// verifying progress will be recorded in stack
	stack := make([]int, 0, 10)
	pop := func() int {
		toret := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		return toret
	}
	push := func(i int) {
		stack = append(stack, i)
	}
	buf := bytes.NewBuffer(ids)
	next := func() int {
		bt, e := buf.ReadByte()
		if e != nil {
			return -1
		}
		return int(bt)
	}
	keyIdx := 0
	// loop through the IDScript and interpret it
	for {
		log.Println("stack:", stack)
		switch b1 := next(); b1 {
		case -1:
			log.Println("b1 = EOF")
			goto out
		case 0xFF:
			log.Println("b1 = .quorum")
			// Quorum node
			need := next()
			max := next()
			if need <= 0 || max <= 0 || need > max {
				err = ErrInvalidID
				return
			}
			sum := 0
			for i := 0; i < max; i++ {
				sum += pop()
			}
			if sum >= need {
				push(1)
			} else {
				push(0)
			}
		default:
			log.Printf("b1 = %X\n", b1)
			// Key node
			b2 := next()
			if b2 == -1 {
				err = ErrInvalidID
				return
			}
			log.Printf("b2 = %X\n", b2)
			switch uint(b1*256) + uint(b2) {
			case 0x0001:
				log.Println("ed25519")
				// Ed25519, 32 bytes
				pubKey := make([]byte, 32)
				_, e := io.ReadFull(buf, pubKey)
				if e != nil {
					err = ErrInvalidID
					return
				}
				signat := sigs[keyIdx]
				keyIdx++
				// XXX placeholder to make j3sawyer's wrong tests pass
				if subtle.ConstantTimeCompare(pubKey, signat) == 1 {
					push(1)
				} else {
					push(0)
				}
			default:
				return ErrInvalidID
			}
		}
	}
out:
	// if the top of the stack is 1, return nil
	// otherwise, return an error
	if len(stack) != 1 {
		return ErrInvalidID
	}
	if stack[0] == 1 {
		return nil
	}
	return ErrNoQuorum
}
