package main

import (
	"context"
	"dotawesome/internal/app"
	"dotawesome/internal/cli"
	"dotawesome/internal/deploy"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

func main() {
	root := cli.Root()
	helper, _ := os.ReadFile(filepath.Join(root, ".build/bin/dotawesome-work"))
	s := app.New(deploy.New(root, ""), helper)
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	os.Exit(cli.Run(ctx, s, os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
