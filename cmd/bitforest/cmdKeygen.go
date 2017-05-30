package main

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"flag"
	"fmt"

	"golang.org/x/crypto/ed25519"

	"github.com/google/subcommands"
)

type cmdKeygen struct {
}

func (cmd *cmdKeygen) Name() string     { return "keygen" }
func (cmd *cmdKeygen) Synopsis() string { return "" }
func (cmd *cmdKeygen) Usage() string    { return "" }

func (cmd *cmdKeygen) SetFlags(f *flag.FlagSet) {
}

func (cmd *cmdKeygen) Execute(_ context.Context,
	f *flag.FlagSet,
	args ...interface{}) subcommands.ExitStatus {
	pk, sk, _ := ed25519.GenerateKey(nil)
	fmt.Println("Public key: .ed25519", hex.EncodeToString(pk), ".quorum 1. 1.")
	fmt.Println("Secret key:", base64.StdEncoding.EncodeToString(sk))
	return 0
}
