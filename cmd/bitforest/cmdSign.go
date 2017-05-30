package main

import (
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"log"

	"golang.org/x/crypto/ed25519"

	"github.com/google/subcommands"
	"github.com/rensa-labs/doriath/operlog"
)

type cmdSign struct {
	argInput  string
	argOutput string
	argSeckey string
}

func (cmd *cmdSign) Name() string     { return "sign" }
func (cmd *cmdSign) Synopsis() string { return "" }
func (cmd *cmdSign) Usage() string    { return "" }

func (cmd *cmdSign) SetFlags(f *flag.FlagSet) {
	f.StringVar(&cmd.argInput, "in", "", "Input operation, in Base64")
	f.StringVar(&cmd.argSeckey, "sec", "", "Secret key, in Base64, for signing the input")
}

func (cmd *cmdSign) Execute(_ context.Context,
	f *flag.FlagSet,
	args ...interface{}) subcommands.ExitStatus {
	if cmd.argInput == "" {
		log.Fatalln("input is required")
	}
	if cmd.argSeckey == "" {
		log.Fatalln("secret key is required")
	}
	in, err := base64.StdEncoding.DecodeString(cmd.argInput)
	if err != nil {
		log.Fatalln("invalid base64 in input:", err.Error())
	}
	sec, err := base64.StdEncoding.DecodeString(cmd.argSeckey)
	if err != nil {
		log.Fatalln("invalid base64 in secret key:", err.Error())
	}
	if len(sec) != ed25519.PrivateKeySize {
		log.Fatalln("secret key of wrong length")
	}
	var tosign operlog.Operation
	err = tosign.FromBytes(in)
	if err != nil {
		log.Fatalln("failed decoding input:", err.Error())
	}
	sig := ed25519.Sign(ed25519.PrivateKey(sec), tosign.SignedPart())
	tosign.Signatures = append(tosign.Signatures, sig)
	fmt.Println(base64.StdEncoding.EncodeToString(tosign.ToBytes()))
	return 0
}
