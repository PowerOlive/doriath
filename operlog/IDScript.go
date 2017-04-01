package operlog

import (
	"bytes"
	"encoding/hex"
	"errors"
	"io"
	"regexp"
	"strconv"

	"golang.org/x/crypto/ed25519"
)

// IDScript is an identity script, which is simply represented in binary form.
type IDScript []byte

var asmRegexp = regexp.MustCompile("[[:space:]]+")

// ErrInvalidID is returned to indicate that an identity script is malformed.
var ErrInvalidID = errors.New("malformed identity scritpt")

// ErrNoQuorum means that all the signatures were valid, but not enough to satisfy the quorum were found.
var ErrNoQuorum = errors.New("not enough signatures to satisfy quorum")

// AssembleID takes in an identity script represented in assembly, and returns a binary ID script.
func AssembleID(asm string) (IDScript, error) {
	tokens := asmRegexp.Split(asm, -1)
	var encoded []byte

	assemTok := func(tok string) []byte {
		if len(tok) < 2 {
			return nil
		}
		if tok[0] == '.' {
			switch tok {
			case ".ed25519":
				return []byte{0x00, 0x01}
			case ".quorum":
				return []byte{0xFF}
			default:
				return nil
			}
		} else if tok[len(tok)-1] == '.' {
			numstr := tok[:len(tok)-1]
			num, err := strconv.Atoi(numstr)
			if err != nil || num >= 256 {
				return nil
			}
			return []byte{byte(num)}
		}
		// it must be hex then
		bts, err := hex.DecodeString(tok)
		if err != nil {
			return nil
		}
		return bts
	}

	if len(tokens) == 0 {
		return nil, ErrInvalidID
	}

	for i := 0; i < len(tokens); i++ {
		if len(tokens[i]) > 0 {
			frag := assemTok(tokens[i])
			if frag == nil {
				return IDScript(nil), errors.New("invalid token")
			}
			encoded = append(encoded, frag...)
		}
	}

	return IDScript(encoded), nil
}

// Verify runs the script with the given array of signatures, and the data, as input. Null error means signature check is good. It must not panic even if the given array is too short, the script is garbage, etc, but should return appropriate errors.
func (ids IDScript) Verify(data []byte, sigs [][]byte) (err error) {
	// we translate panics into error to avoid manually handling array oob, etc
	defer func() {
		if e := recover(); e != nil {
			err = ErrInvalidID
		}
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
		switch b1 := next(); b1 {
		case -1:
			goto out
		case 0xFF:
			// Quorum node
			need := next()
			max := next()
			if need < 0 || max < 0 || need > max {
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
			// Key node
			b2 := next()
			if b2 == -1 {
				err = ErrInvalidID
				return
			}
			switch uint(b1<<8) | uint(b2) {
			case 0x0001:
				// Ed25519, 32 bytes
				pubKey := make([]byte, 32)
				_, e := io.ReadFull(buf, pubKey)
				if e != nil {
					err = ErrInvalidID
					return
				}
				signat := sigs[keyIdx]
				keyIdx++
				// verify signature against key and data
				if !ed25519.Verify(pubKey, data, signat) {
					push(0)
				} else {
					push(1)
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
