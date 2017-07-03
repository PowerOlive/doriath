package main

import (
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"log"

	"github.com/google/subcommands"
	"github.com/rensa-labs/doriath/operlog"

	crand "crypto/rand"
)

type cmdNewop struct {
	argNextID string
	argData   string
}

func (cmd *cmdNewop) Name() string     { return "newop" }
func (cmd *cmdNewop) Synopsis() string { return "" }
func (cmd *cmdNewop) Usage() string    { return "" }

func (cmd *cmdNewop) SetFlags(f *flag.FlagSet) {
	f.StringVar(&cmd.argNextID, "nid", "", "Next ID script, in assembly")
	f.StringVar(&cmd.argData, "data", "", "Data recorded in operation")
}

func (cmd *cmdNewop) Execute(_ context.Context,
	f *flag.FlagSet,
	args ...interface{}) subcommands.ExitStatus {
	var op operlog.Operation
	op.Nonce = make([]byte, 16)
	crand.Read(op.Nonce)
	op.Data = cmd.argData
	comp, err := operlog.AssembleID(cmd.argNextID)
	if err != nil {
		log.Fatalln("cannot assemble given ID script:", err)
	}
	op.NextID = comp
	fmt.Println(base64.StdEncoding.EncodeToString(op.ToBytes()))
	return 0
}
