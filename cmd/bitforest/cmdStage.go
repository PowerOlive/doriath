package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"log"

	"github.com/google/subcommands"
	"github.com/rensa-labs/doriath"
	"github.com/rensa-labs/doriath/operlog"
)

type cmdStage struct {
	argDbloc string
	argName  string
	argOp    string
}

func (cmd *cmdStage) Name() string     { return "stage" }
func (cmd *cmdStage) Synopsis() string { return "" }
func (cmd *cmdStage) Usage() string    { return "" }

func (cmd *cmdStage) SetFlags(f *flag.FlagSet) {
	f.StringVar(&cmd.argDbloc, "db", "", "Location of NA database")
	f.StringVar(&cmd.argName, "name", "", "Name to register/update")
	f.StringVar(&cmd.argOp, "op", "", "Operation in base64 format; omit argument to view history rather than stage changes")
}

func (cmd *cmdStage) Execute(_ context.Context,
	f *flag.FlagSet,
	args ...interface{}) subcommands.ExitStatus {
	if cmd.argDbloc == "" || cmd.argName == "" {
		log.Fatalln("-db and -name are mandatory arguments")
	}
	srv, err := doriath.NewServer(nil, "", 0, cmd.argDbloc)
	if err != nil {
		log.Fatalln("could not open database:", err)
	}
	ops, err := srv.GetOperations(cmd.argName)
	if err != nil {
		log.Fatalln("could not query name:", err)
	}
	bts, _ := json.MarshalIndent(ops, "", "    ")
	log.Println("Existing ops:", string(bts))
	if cmd.argOp == "" {
		return 0
	}
	var newop operlog.Operation
	bts, err = base64.StdEncoding.DecodeString(cmd.argOp)
	if err != nil {
		log.Fatalln("could not decode operation:", err)
	}
	err = newop.FromBytes(bts)
	if err != nil {
		log.Fatalln("could not decode operation:", err)
	}
	err = srv.StageOperation(cmd.argName, newop)
	if err != nil {
		log.Fatalln("could not stage operation:", err)
	}
	return 0
}
