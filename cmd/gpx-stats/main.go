// Command gpx-stats computes statistics about a GPX file, either from the CLI
// (given a path) or via an embedded web UI (the `serve` subcommand). It embeds
// all resources it needs and never persists data or opens external connections.
package main

import (
	"os"

	"github.com/sgaunet/gpx-stats/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
