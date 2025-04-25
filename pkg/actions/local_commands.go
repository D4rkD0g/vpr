// Package actions defines the interface and registry for runnable actions.
// This file implements the specific action handler for executing local system commands
// (`type: execute_local_commands`). This action is useful for interacting with the
// local environment, but comes with security considerations.
package actions

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"
	
	"vpr/pkg/context"
	"vpr/pkg/poc"
)

// Common errors
var (
	errNoCommands = errors.New("no commands specified")
)

// LocalCommandResult represents the result of executing a local command
type LocalCommandResult struct {
	Commands  []string `json:"commands"`
	Output    string   `json:"output"`
	ExitCode  int      `json:"exit_code"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Duration  float64  `json:"duration_seconds"`
	Success   bool     `json:"success"`
}

// LocalCommandsHandler implements the execute_local_commands action type.
// It executes one or more local system commands and captures their output.
func LocalCommandsHandler(ctx *context.ExecutionContext, action *poc.Action) (interface{}, error) {
	// Validation
	if action.Type != "execute_local_commands" {
		return nil, errInvalidActionType(action.Type, "execute_local_commands")
	}
	
	// Check for commands
	if len(action.Commands) == 0 {
		return nil, errNoCommands
	}
	
	// Log detailed information about the command execution
	slog.Info("Executing local commands", 
		"command_count", len(action.Commands),
		"first_command", action.Commands[0])
		
	// Process each command, we'll collect results in a composite output
	var combinedOutput bytes.Buffer
	startTime := time.Now()
	success := true
	exitCode := 0
	
	// Execute each command in sequence
	for i, cmdStr := range action.Commands {
		// Substitute variables in the command string
		resolvedCmd, err := ctx.Substitute(cmdStr)
		if err != nil {
			slog.Error("Failed to resolve variables in command", 
				"command", cmdStr, 
				"error", err)
			return nil, fmt.Errorf("failed to resolve variables in command: %w", err)
		}
		
		// Prepare the command
		slog.Info("Executing command", "index", i+1, "command", resolvedCmd)
		
		// Split the command into executable and arguments
		parts := strings.Fields(resolvedCmd)
		if len(parts) == 0 {
			slog.Warn("Empty command after processing", "original", cmdStr)
			continue
		}
		
		cmd := exec.Command(parts[0], parts[1:]...)
		
		// Capture command output
		output, err := cmd.CombinedOutput()
		
		// Append to combined output
		combinedOutput.WriteString(fmt.Sprintf("Command %d: %s\n", i+1, resolvedCmd))
		combinedOutput.WriteString(fmt.Sprintf("Output:\n%s\n", string(output)))
		
		if err != nil {
			// Command failed
			success = false
			if exitError, ok := err.(*exec.ExitError); ok {
				exitCode = exitError.ExitCode()
				combinedOutput.WriteString(fmt.Sprintf("Exit code: %d\n", exitCode))
			} else {
				exitCode = -1
				combinedOutput.WriteString(fmt.Sprintf("Error: %s\n", err.Error()))
			}
			
			// Log the failure
			slog.Error("Command execution failed", 
				"command", resolvedCmd,
				"exit_code", exitCode,
				"error", err)
			
			// Break on first error unless we need to execute all commands
			// regardless of errors (we could add a parameter for this later)
			break
		} else {
			combinedOutput.WriteString("Exit code: 0\n")
			slog.Info("Command executed successfully", "command", resolvedCmd)
		}
		
		combinedOutput.WriteString("\n")
	}
	
	// Construct the result
	endTime := time.Now()
	result := &LocalCommandResult{
		Commands:  action.Commands,
		Output:    combinedOutput.String(),
		ExitCode:  exitCode,
		StartTime: startTime,
		EndTime:   endTime,
		Duration:  endTime.Sub(startTime).Seconds(),
		Success:   success,
	}
	
	// Store the output in the target variable if specified
	if action.TargetVariable != "" {
		err := ctx.SetVariable(action.TargetVariable, result)
		if err != nil {
			slog.Error("Failed to set result variable", 
				"variable", action.TargetVariable, 
				"error", err)
			return result, fmt.Errorf("failed to store command result: %w", err)
		}
		
		slog.Info("Stored command result in variable", 
			"variable", action.TargetVariable,
			"success", success,
			"exit_code", exitCode)
	}
	
	return result, nil
}

func init() {
	// Register the execute_local_commands action handler
	MustRegisterAction("execute_local_commands", LocalCommandsHandler)
}
