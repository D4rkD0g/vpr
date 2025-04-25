// Package actions defines the interface and registry for runnable actions.
// This file implements the manual_action, which pauses execution and requires
// user confirmation before continuing. This is useful for steps that require
// human verification or intervention.
package actions

import (
	"bufio"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
	
	"vpr/pkg/context"
	"vpr/pkg/poc"
)

// Common errors
var (
	errManualActionCancelled = errors.New("manual action cancelled by user")
	errManualActionTimedOut  = errors.New("manual action timed out waiting for confirmation")
)

// ManualActionResult represents the result of a manual action
type ManualActionResult struct {
	Message      string    `json:"message"`
	Response     string    `json:"response,omitempty"`
	Confirmed    bool      `json:"confirmed"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	Duration     float64   `json:"duration_seconds"`
	WasTimedOut  bool      `json:"was_timed_out,omitempty"`
}

// ManualActionHandler implements the manual_action type.
// It pauses execution and requires user confirmation before continuing.
func ManualActionHandler(ctx *context.ExecutionContext, action *poc.Action) (interface{}, error) {
	// Validation
	if action.Type != "manual_action" {
		return nil, errInvalidActionType(action.Type, "manual_action")
	}
	
	// Get the message to display to the user
	message := "Manual action required. Press Enter to continue or type 'cancel' to abort."
	if action.Description != "" {
		// If a description is provided, use it as the prompt
		message = action.Description
	}
	
	// Substitute any variables in the message
	resolvedMessage, err := ctx.Substitute(message)
	if err != nil {
		slog.Error("Failed to resolve variables in manual action message", "error", err)
		return nil, fmt.Errorf("failed to resolve variables in message: %w", err)
	}
	
	// Log the manual action request
	slog.Info("Manual action required", 
		"message", resolvedMessage)
	
	startTime := time.Now()
	
	// Create the result structure
	result := &ManualActionResult{
		Message:   resolvedMessage,
		StartTime: startTime,
		Confirmed: false,
	}
	
	// Check if a timeout was specified
	var timeout time.Duration
	if action.Timeout != "" {
		parsedTimeout, err := time.ParseDuration(action.Timeout)
		if err != nil {
			slog.Error("Invalid timeout duration", "timeout", action.Timeout, "error", err)
			return nil, fmt.Errorf("invalid timeout duration: %w", err)
		}
		timeout = parsedTimeout
	}
	
	// Create a channel for the user input
	responseChan := make(chan string, 1)
	
	// Start a goroutine to read user input
	go func() {
		fmt.Println("\n" + strings.Repeat("-", 80))
		fmt.Printf("\n MANUAL ACTION REQUIRED\n\n")
		fmt.Println(resolvedMessage)
		if timeout > 0 {
			fmt.Printf("\nTimeout: %s\n", timeout)
		}
		fmt.Printf("\nEnter 'y' to continue, 'n' to abort: ")
		
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))
		
		responseChan <- input
	}()
	
	// Wait for user input or timeout
	var response string
	if timeout > 0 {
		// Set up a timer for timeout
		timer := time.NewTimer(timeout)
		select {
		case response = <-responseChan:
			timer.Stop()
		case <-timer.C:
			// Timeout occurred
			fmt.Println("\n Manual action timed out!")
			result.WasTimedOut = true
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(startTime).Seconds()
			
			slog.Warn("Manual action timed out", 
				"timeout", timeout.String(),
				"duration", result.Duration)
			
			return result, errManualActionTimedOut
		}
	} else {
		// No timeout, wait indefinitely
		response = <-responseChan
	}
	
	// Process the response
	result.Response = response
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(startTime).Seconds()
	
	// Check if the user confirmed or cancelled
	if response == "y" || response == "yes" {
		result.Confirmed = true
		fmt.Println("\n Manual action confirmed. Continuing execution.")
		slog.Info("Manual action confirmed by user", 
			"duration", result.Duration)
	} else {
		fmt.Println("\n Manual action cancelled. Aborting execution.")
		slog.Warn("Manual action cancelled by user", 
			"response", response,
			"duration", result.Duration)
		return result, errManualActionCancelled
	}
	
	// Store the result in the target variable if specified
	if action.TargetVariable != "" {
		err := ctx.SetVariable(action.TargetVariable, result)
		if err != nil {
			slog.Error("Failed to set result variable", 
				"variable", action.TargetVariable, 
				"error", err)
			return result, fmt.Errorf("failed to store manual action result: %w", err)
		}
		
		slog.Info("Stored manual action result in variable", 
			"variable", action.TargetVariable)
	}
	
	return result, nil
}

func init() {
	// Register the manual_action handler
	MustRegisterAction("manual_action", ManualActionHandler)
}
