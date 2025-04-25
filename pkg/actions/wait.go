// Package actions defines the interface and registry for runnable actions.
// This file implements the specific action handler for pausing execution
// for a specified duration (`type: wait`).
package actions

import (
	"log/slog"
	
	"vpr/pkg/context"
	"vpr/pkg/poc"
)

// WaitHandler implements the wait action type as defined in DSL specification.
// It pauses execution for a specified duration before proceeding to the next step.
// This is a wrapper around the core implementation in utilities.go
func WaitHandler(ctx *context.ExecutionContext, action *poc.Action) (interface{}, error) {
	// Validation
	if action.Type != "wait" {
		return nil, errInvalidActionType(action.Type, "wait")
	}
	
	// Log the wait attempt with proper structured logging
	slog.Info("Executing wait action")
	
	// Call the core implementation from utilities.go
	result, err := waitHandler(ctx, action)
	
	if err != nil {
		slog.Error("Wait action failed", "error", err)
		return nil, err
	}
	
	// Log success with duration
	if resultMap, ok := result.(map[string]interface{}); ok {
		slog.Info("Wait completed", 
			"duration", resultMap["duration"],
			"duration_ms", resultMap["duration_ms"])
	} else {
		slog.Info("Wait completed")
	}
	
	return result, nil
}

func init() {
	// Register the wait action handler
	MustRegisterAction("wait", WaitHandler)
}
