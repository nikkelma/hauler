package main

import (
	"context"
	"os"

	"github.com/nikkelma/hauler/cmd/hauler/cli"
	"github.com/nikkelma/hauler/pkg/log"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := log.NewLogger(os.Stdout)
	ctx = logger.WithContext(ctx)

	if err := cli.New().ExecuteContext(ctx); err != nil {
		logger.Errorf("%v", err)
	}
}
