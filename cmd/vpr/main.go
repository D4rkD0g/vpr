// Package main implements the command-line interface for the Vulnerability PoC Runner (VPR).
// It parses arguments, loads PoC definitions, initializes the execution context,
// runs the PoC using the executor, and reports the results.
package main

import (
	"flag"
	"log/slog"
	"os"

	"vpr/pkg/context" // Adjust imports
	"vpr/pkg/executor"
	"vpr/pkg/poc"

	// Import action/check packages to trigger init() registration
	_ "vpr/pkg/actions"
	_ "vpr/pkg/checks"
)

func main() {
	pocPath := flag.String("p", "", "Path to PoC v1.0 YAML file (required)")
	// Add flags for target overrides (e.g., -host, -port)
	// Add flags for credential handling strategy/source
	logLevel := flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	flag.Parse()

	// Setup structured logging
	var level slog.Level
	switch *logLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)

	if *pocPath == "" {
		slog.Error("PoC file path is required (-p)")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// 1. Load and Parse PoC
	slog.Info("Loading PoC", "path", *pocPath)
	pocDef, err := poc.LoadPoc(*pocPath)
	if err != nil {
		slog.Error("Failed to load PoC", "path", *pocPath, "error", err)
		os.Exit(1)
	}
	// Basic validation (e.g., check DSL version)
	if pocDef.Metadata.DslVersion != "1.0" {
		slog.Warn("PoC DSL version mismatch", "expected", "1.0", "found", pocDef.Metadata.DslVersion)
		// Decide whether to proceed or exit
	}

	// 2. Initialize Context
	// TODO: Pass target overrides, credential strategy to context initialization
	slog.Info("Initializing execution context")
	execCtx, err := context.NewExecutionContext(&pocDef.Context)
	if err != nil {
		slog.Error("Failed to initialize context", "error", err)
		os.Exit(1)
	}

	// 3. Execute PoC
	slog.Info("Starting PoC execution", "id", pocDef.Metadata.ID)
	result, err := executor.Execute(pocDef, execCtx) // Execute returns final result & overall exec error

	// 4. Report Results
	slog.Info("PoC execution finished", "id", pocDef.Metadata.ID)
	reporter.PrintResult(result, os.Stdout) // Implement reporting function

	if err != nil {
		slog.Error("Execution encountered an error", "error", err)
		os.Exit(1) // Exit with error if overall execution failed
	}
	if !result.Success { // Check the result status
		os.Exit(1) // Exit with error if PoC logic determined failure/no match
	}
	os.Exit(0) // Success
}
