package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"crawler-cli/internal/config"
	"crawler-cli/internal/crawler"
	"crawler-cli/internal/fetcher"
	"crawler-cli/internal/logger"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Parse(os.Args[1:])
	if err != nil {
		return err
	}

	log, err := logger.New(cfg.LogFile)
	if err != nil {
		return err
	}
	defer log.Close()

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	f := fetcher.New(cfg.RequestTimeout)
	c := crawler.New(cfg.Depth, f, log)

	pages, err := c.Run(ctx, cfg.URLs)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(pages, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cfg.Output, data, 0o644)
}
