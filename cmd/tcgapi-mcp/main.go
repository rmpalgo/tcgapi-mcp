package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/rmpalgo/tcgapi-mcp/internal/buildinfo"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	os.Exit(run())
}

func run() int {
	info := buildinfo.Info{
		Version: version,
		Commit:  commit,
		Date:    date,
	}

	if len(os.Args) == 2 && os.Args[1] == "--version" {
		fmt.Fprintln(os.Stdout, info.String())
		return 0
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	opts := runOptions{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Args:   os.Args[1:],
		Build:  info,
	}

	if err := runServer(ctx, opts); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	return 0
}
