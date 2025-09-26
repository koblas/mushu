package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/koblas/mushu/internal/cli"
	"github.com/koblas/mushu/internal/config"
	"github.com/koblas/mushu/internal/logging"
	"github.com/koblas/mushu/internal/version"
)

func main() {
	ctx := context.Background()

	// Load initial config for logging setup
	cfg, err := config.Load("config.yaml", nil)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create logger and add to context
	logger := logging.New(cfg.Logging.Level, cfg.Logging.Format)
	ctx = logging.WithLogger(ctx, logger)

	// Get version information once
	versionInfo := version.Get()
	logging.Debug(ctx, "Starting mushu",
		"version", versionInfo.Version,
		"commit", versionInfo.Commit,
		"build_time", versionInfo.BuildTime,
		"go_version", versionInfo.GoVersion,
		"log_level", cfg.Logging.Level,
		"log_format", cfg.Logging.Format)

	if err := cli.Execute(ctx); err != nil {
		logging.Error(ctx, "Application error", "error", err)
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	logging.Info(ctx, "Application completed successfully")
}
