// Package executor orchestrates the execution of a parsed PoC definition.
// This file contains the logic for running individual steps defined within a PoC.
// It handles conditional execution (`if`), loops (`loop`), delegates execution
// to the appropriate action or check handlers, and manages step results.
package executor

import (
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"time"
	
	"vpr/pkg/checks"
	"vpr/pkg/context"
	"vpr/pkg/poc"
)

// RunStep executes a single step and handles its action or check
// This is the main entry point for step execution, handling both if conditions and loops
func RunStep(step *poc.Step, phase string, stepNumber int, ctx *context.ExecutionContext, options *ExecutorOptions) (*StepResult, error) {
	// Create the base step result
	result := &StepResult{
		DSL:       step.DSL,
		StartTime: time.Now(),
	}
	
	// Generate a meaningful step identifier
	stepID := fmt.Sprintf("%s.%v", phase, step.Step)
	if step.ID != "" {
		stepID = fmt.Sprintf("%s.%s", phase, step.ID)
	}
	
	slog.Debug("Evaluating step", "step_id", stepID, "dsl", step.DSL)
	
	// 1. Check if condition (conditional execution)
	if step.If != "" {
		shouldRun, err := evaluateCondition(step.If, ctx)
		if err != nil {
			result.Success = false
			result.Error = fmt.Errorf("condition evaluation failed: %w", err)
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime).Seconds()
			return result, result.Error
		}
		
		if !shouldRun {
			slog.Debug("Skipping step due to condition", "step_id", stepID, "condition", step.If)
			result.Success = true
			result.Skipped = true
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime).Seconds()
			return result, nil
		}
	}
	
	// 2. If this is a loop, handle it differently
	if step.Loop != nil {
		return handleLoopStep(step, phase, stepNumber, ctx, options)
	}
	
	// 3. Execute the action or check
	if step.Action != nil {
		return executeAction(step.Action, step.DSL, stepID, ctx, options)
	} else if step.Check != nil {
		return executeCheck(step.Check, step.DSL, stepID, ctx, options)
	} else {
		result.Success = false
		result.Error = errors.New("step has neither action nor check")
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime).Seconds()
		return result, result.Error
	}
}

// evaluateCondition evaluates the 'if' condition and returns whether the step should be executed
func evaluateCondition(condition string, ctx *context.ExecutionContext) (bool, error) {
	// 1. Use the context's built-in EvaluateCondition method if available
	if result, err := ctx.EvaluateCondition(condition); err == nil {
		return result, nil
	}
	
	// 2. Fallback to simpler evaluation method
	
	// Substitute any variables in the condition
	conditionStr, err := ctx.Substitute(condition)
	if err != nil {
		return false, fmt.Errorf("error substituting variables in condition: %w", err)
	}
	
	// Check if condition is a simple boolean literal
	if conditionStr == "true" {
		return true, nil
	}
	if conditionStr == "false" {
		return false, nil
	}
	
	// 3. Check if condition is a variable that holds a boolean
	varValue, err := ctx.ResolveVariable(condition)
	if err == nil && varValue != nil {
		// If variable exists, try to interpret it as boolean
		switch v := varValue.(type) {
		case bool:
			return v, nil
		case string:
			lowerStr := v
			if lowerStr == "true" {
				return true, nil
			}
			if lowerStr == "false" {
				return false, nil
			}
		case float64, int, int64:
			// In many languages, 0 is false, everything else is true
			num := reflect.ValueOf(v).Float()
			return num != 0, nil
		}
	}
	
	// 4. For complex expressions, this could be expanded in the future
	return false, fmt.Errorf("unsupported condition: %s", condition)
}

// handleLoopStep executes a loop step
func handleLoopStep(step *poc.Step, phase string, stepNumber int, ctx *context.ExecutionContext, options *ExecutorOptions) (*StepResult, error) {
	result := &StepResult{
		DSL:       step.DSL,
		StartTime: time.Now(),
	}
	
	// Create a retryable executor for loop steps
	executor := NewRetryableExecutor(RunStep, options)
	
	// Unique identifier for this loop
	loopID := fmt.Sprintf("%s.loop.%d", phase, stepNumber)
	if step.ID != "" {
		loopID = fmt.Sprintf("%s.loop.%s", phase, step.ID)
	}
	
	// Execute the loop
	_, err := executor.ExecuteLoopWithRetry(step.Loop, phase, loopID, ctx)
	
	// Finalize the result
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime).Seconds()
	
	if err != nil {
		result.Success = false
		result.Error = fmt.Errorf("loop execution failed: %w", err)
		return result, result.Error
	}
	
	result.Success = true
	return result, nil
}

// executeAction runs an action through the action registry
func executeAction(action *poc.Action, dsl, stepID string, ctx *context.ExecutionContext, options *ExecutorOptions) (*StepResult, error) {
	result := &StepResult{
		DSL:       dsl,
		StartTime: time.Now(),
	}
	
	// Log action start
	slog.Info("Executing action", 
		"step_id", stepID, 
		"type", action.Type, 
		"description", action.Description)
	
	// Execute with retry capability
	actionResult, err := executeActionWithRetry(ctx, action, DefaultRetryCondition())
	
	// Finalize the result
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime).Seconds()
	result.Output = actionResult
	
	if err != nil {
		result.Success = false
		result.Error = fmt.Errorf("action execution failed: %w", err)
		slog.Error("Action execution failed", 
			"step_id", stepID, 
			"type", action.Type, 
			"error", err)
		return result, result.Error
	}
	
	result.Success = true
	slog.Info("Action execution successful", 
		"step_id", stepID, 
		"type", action.Type, 
		"duration", result.Duration)
	return result, nil
}

// executeCheck runs a check through the check registry
func executeCheck(check *poc.Check, dsl, stepID string, ctx *context.ExecutionContext, options *ExecutorOptions) (*StepResult, error) {
	result := &StepResult{
		DSL:       dsl,
		StartTime: time.Now(),
	}
	
	// Log check start
	slog.Info("Executing check", 
		"step_id", stepID, 
		"type", check.Type, 
		"description", check.Description)
	
	// Get polling configuration
	pollingConfig, err := getPollingConfig(ctx, check)
	if err != nil {
		result.Success = false
		result.Error = fmt.Errorf("failed to get polling configuration: %w", err)
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime).Seconds()
		return result, result.Error
	}
	
	// Initialize variables for polling
	var attempts int
	var checkResult bool
	var checkErr error
	
	// Polling loop
	for attempts = 1; attempts <= pollingConfig.MaxAttempts; attempts++ {
		// Execute the check with correct parameter order
		checkResult, checkErr = checks.ExecuteCheck(ctx, check)
		
		// If successful or we're on the last attempt, exit the loop
		if checkErr == nil || attempts == pollingConfig.MaxAttempts {
			break
		}
		
		// If expected error check and ExpectedError is defined,
		// we should not retry, as this is a special case
		if check.Type == "expected_error" && check.ExpectedError != nil {
			// For expected_error checks, the error might be the expected outcome
			// Don't treat as failure, let it be processed by the check handler
			break
		}
		
		// For other checks, if ExpectedError is defined, we should handle the error differently
		if check.ExpectedError != nil {
			// Special handling for expected errors in other check types
			slog.Info("Check failed with potential expected error", 
				"step_id", stepID, 
				"type", check.Type, 
				"error", checkErr,
				"attempt", attempts)
			
			// Create a special expected_error check to validate this error
			errorCheck := &poc.Check{
				Type:          "expected_error",
				ExpectedError: check.ExpectedError,
			}
			
			// Execute the error check
			isExpectedError, _ := checks.ExecuteCheck(ctx, errorCheck)
			
			// If this is indeed the expected error, consider the check successful
			if isExpectedError {
				slog.Info("Confirmed expected error condition", 
					"step_id", stepID, 
					"type", check.Type)
				checkErr = nil
				// For expected errors, we consider the check successful even though it "failed"
				// This matches the DSL specification for expected_error behavior
				checkResult = true
				break
			}
		}
		
		// Log retry attempt
		slog.Debug("Check failed, retrying", 
			"step_id", stepID, 
			"type", check.Type, 
			"error", checkErr,
			"attempt", attempts,
			"max_attempts", pollingConfig.MaxAttempts)
		
		// Wait before next attempt
		time.Sleep(pollingConfig.Interval)
	}
	
	// Finalize the result
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime).Seconds()
	result.Output = checkResult
	
	if checkErr != nil {
		// Special case for explicit expected_error checks
		// For these, an error means the expected error didn't occur or didn't match
		if check.Type == "expected_error" {
			result.Success = false
			result.Error = fmt.Errorf("expected error check failed: %w", checkErr)
			slog.Error("Expected error check failed", 
				"step_id", stepID, 
				"error", checkErr)
			return result, result.Error
		}
		
		// For other checks with ExpectedError, we already handled them in the loop
		if check.ExpectedError != nil {
			// Check one final time if this error matches the expected pattern
			errorCheck := &poc.Check{
				Type:          "expected_error",
				ExpectedError: check.ExpectedError,
			}
			isExpectedError, _ := checks.ExecuteCheck(ctx, errorCheck)
			
			if isExpectedError {
				// This is the expected error - consider check successful
				result.Success = true
				slog.Info("Check completed with expected error", 
					"step_id", stepID, 
					"type", check.Type, 
					"attempts", attempts)
				return result, nil
			}
		}
		
		// Normal error case - check failed unexpectedly
		result.Success = false
		result.Error = fmt.Errorf("check failed after %d attempts: %w", attempts, checkErr)
		slog.Error("Check execution failed", 
			"step_id", stepID, 
			"type", check.Type, 
			"attempts", attempts,
			"error", checkErr)
		return result, result.Error
	}
	
	result.Success = true
	slog.Info("Check execution successful", 
		"step_id", stepID, 
		"type", check.Type, 
		"attempts", attempts,
		"duration", result.Duration)
	return result, nil
}
