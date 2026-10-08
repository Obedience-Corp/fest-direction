// Command fest-direction computes, anchors, and attests stable direction hashes
// for Festival work units. All behavior lives in internal/cli and the domain
// packages it calls; main only owns process lifecycle and the exit code.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Obedience-Corp/fest-direction/internal/cli"
	"github.com/Obedience-Corp/fest-direction/internal/errs"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := cli.Execute(ctx); err != nil {
		if !errors.Is(err, errs.ErrAlreadyPrinted) {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}
}
