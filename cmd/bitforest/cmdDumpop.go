package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"

	"github.com/google/subcommands"
	"github.com/rensa-labs/doriath/operlog"
)

type cmdDumpop struct {
	argInput string
}

func (cmd *cmdDumpop) Name() string     { return "dumpop" }
func (cmd *cmdDumpop) Synopsis() string { return "" }
func (cmd *cmdDumpop) Usage() string    { return "" }

func (cmd *cmdDumpop) SetFlags(f *flag.FlagSet) {
	f.StringVar(&cmd.argInput, "in", "", "Input operation, in Base64")
}

func (cmd *cmdDumpop) Execute(_ context.Context,
	f *flag.FlagSet,
	args ...interface{}) subcommands.ExitStatus {
	in, err := base64.StdEncoding.DecodeString(cmd.argInput)
	if err != nil {
		log.Fatalln("invalid base64 in input:", err.Error())
	}
	var todump operlog.Operation
	err = todump.FromBytes(in)
	if err != nil {
		log.Fatalln("failed decoding input:", err.Error())
	}
	bts, _ := json.MarshalIndent(todump, "", "    ")
	fmt.Println(string(bts))
	return 0
}
