// Package main implements the command-line interface for the Vulnerability PoC Runner (VPR).
// It parses arguments, loads PoC definitions, initializes the execution context,
// runs the PoC using the executor, and reports the results.
package main

import (
	"flag"
	"log/slog"
	"os"

	"vpr/pkg/executor"
	"vpr/pkg/poc"
	"vpr/pkg/reporter"

	// Import action/check packages to trigger init() registration
	_ "vpr/pkg/actions"
	_ "vpr/pkg/checks"
	_ "vpr/pkg/extractors" // Import extractors for init() registration
)

func applyOrUpdateContextVar(env *[]poc.ContextEnvironment, id string, value string) {
	for i, envVar := range *env {
		if envVar.ID == id {
			(*env)[i].Value = value
			return
		}
	}
	*env = append(*env, poc.ContextEnvironment{ID: id, Value: value})
}

func main() {
	pocPath := flag.String("p", "", "Path to PoC v1.0 YAML file (required)")
	// Add flags for target overrides
	hostOverride := flag.String("host", "", "Override target host in PoC definition")
	portOverride := flag.String("port", "", "Override target port in PoC definition")
	urlOverride := flag.String("url", "", "Override target URL in PoC definition")
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
	pocDef, err := poc.LoadPocFromFile(*pocPath)
	if err != nil {
		slog.Error("Failed to load PoC", "path", *pocPath, "error", err)
		os.Exit(1)
	}
	// Basic validation (e.g., check DSL version)
	if pocDef.Metadata.DslVersion != "1.0" {
		slog.Warn("PoC DSL version mismatch", "expected", "1.0", "found", pocDef.Metadata.DslVersion)
		// Decide whether to proceed or exit
	}

	// Apply command line overrides to context if specified
	if *hostOverride != "" || *portOverride != "" || *urlOverride != "" {
		// Initialize the environment map if it doesn't exist
		if pocDef.Context.Environment == nil {
			pocDef.Context.Environment = make([]poc.ContextEnvironment, 0)
		}

		slog.Info("Applying target overrides from command line")
		
		// Apply host override
		if *hostOverride != "" {
			applyOrUpdateContextVar(&pocDef.Context.Environment, "target_host", *hostOverride)
			slog.Debug("Set target_host override", "value", *hostOverride)
		}
		
		// Apply port override
		if *portOverride != "" {
			applyOrUpdateContextVar(&pocDef.Context.Environment, "target_port", *portOverride)
			slog.Debug("Set target_port override", "value", *portOverride)
		}
		
		// Apply URL override
		if *urlOverride != "" {
			applyOrUpdateContextVar(&pocDef.Context.Environment, "target_url", *urlOverride)
			slog.Debug("Set target_url override", "value", *urlOverride)
		}
	}

	// 2. Configure execution options
	slog.Info("Preparing for execution")
	execOptions := executor.DefaultOptions()
	// TODO: Set options based on command-line flags

	// 3. Execute PoC
	slog.Info("Starting PoC execution", "id", pocDef.Metadata.ID)
	result, err := executor.Execute(pocDef, execOptions)

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
