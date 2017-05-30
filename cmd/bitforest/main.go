package main

import (
	"context"
	"flag"
	"os"

	"github.com/google/subcommands"
)

func main() {
	subcommands.Register(&cmdSign{}, "")
	subcommands.Register(&cmdKeygen{}, "")
	subcommands.Register(&cmdNewop{}, "")
	subcommands.Register(&cmdDumpop{}, "")
	flag.Parse()
	ctx := context.Background()
	os.Exit(int(subcommands.Execute(ctx)))
}
