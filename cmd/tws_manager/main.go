package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"tws_manager/internal/app"
)

func main() {
	os.Exit(runMain(os.Args[1:]))
}

func runMain(args []string) int {
	fs := flag.NewFlagSet("tws_manager", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	flags := app.RegisterFlags(fs, app.ProfileCLI)
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	cfg, err := app.ConfigFromFlags(flags)
	if err != nil {
		fatalf("invalid flags: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := app.Run(ctx, cfg, run); err != nil && ctx.Err() == nil {
		fatalf("%v", err)
	}
	return 0
}

func run(ctx context.Context, rt *app.Runtime) error {
	services, err := app.WireServices(ctx, rt)
	if err != nil {
		return err
	}
	return app.RunCLI(ctx, rt, services)
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	exitFn(1)
}

var exitFn = os.Exit
