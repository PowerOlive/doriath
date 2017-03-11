package operlog

import (
	"encoding/hex"
	"errors"
	"regexp"
	"strconv"
)

// IDScript is an identity script, which is simply represented in binary form.
type IDScript []byte

var asmRegexp = regexp.MustCompile("[[:space:]]")

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

// Verify runs the script with the given array of signatures as input. Null error means signature check is good. It must not panic even if the given array is too short, the script is garbage, etc, but should return appropriate errors.
func (ids IDScript) Verify(sigs [][]byte) error {
	// verifying progress will be recorded in stack
	stack := make([]int, 0, 10)
	// loop through the IDScript and interpret it
	for i := 0; i < len(ids); i++ {
		if ids[i] == 0 && ids[i+1] == 1 { // .ed25519
			// increment i accordingly
			i += 2
			// set a flag 'found' to false
			found := false
			// read 32 bytes and compare it to sigs
			asig := ids[i : i+32]
			for j := range sigs {
				sigi := sigs[j]
				// different types of signatures; move on to the next sig
				if len(asig) != len(sigi) {
					break
				}
				// set found to true
				found = true
				// compare asig with sigi; if there's a difference, set found to false and break
				for k := range asig {
					if asig[k] != sigi[k] {
						found = false
						break
					}
				}
				// if there was a match, break this loop
				if found {
					break
				}
			}
			// increment i accordingly
			i += 31
			// if found a match, push 1; otherwise, push 0
			if found {
				stack = append(stack, 1)
			} else {
				stack = append(stack, 0)
			}
			// endif .ed25519
		} else if ids[i] == 255 { // .quorum
			// read two bytes; the first byte is x and the second byte is y
			i++
			x := int(ids[i])
			i++
			y := int(ids[i])
			// pop y elements from the stack, sum them up to s
			s := 0
			for i := 0; i < y; i++ {
				s = s + stack[len(stack)-1]
				stack = stack[:len(stack)-1]
			}
			// if s > x then push 1; otherwise, push 0
			if s >= x {
				stack = append(stack, 1)
			} else {
				stack = append(stack, 0)
			}
			// endif .quorum
		}
	} // endfor

	// if the top of the stack is 1, return nil
	// otherwise, return an error
	if stack[len(stack)-1] == 1 {
		return nil
	}
	return errors.New("verification failed")
}
