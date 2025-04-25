// Package extractors provides the core logic for executing extractors defined in a PoC.
// It handles data extraction from responses using various methods like regex, JSONPath, etc.
package extractors

import (
	"fmt"
	"log/slog" // Use structured logging
	"strings"
	"time"

	"vpr/pkg/actions" // Adjust imports based on actual structure
	"vpr/pkg/checks"
	"vpr/pkg/context"
	"vpr/pkg/poc"
)

func RunStep(step *poc.Step, phase string, ctx *context.ExecutionContext) error {
	slog.Info("Running step", "phase", phase, "dsl", step.DSL)

	// 0. Handle Manual Step
	if step.Manual {
		slog.Warn("Manual step required", "dsl", step.DSL, "description", step.Action.Description) // Assume action holds details
		fmt.Println(">>> MANUAL STEP REQUIRED <<<")
		fmt.Printf("    DSL: %s\n", step.DSL)
		if step.Action != nil {
			fmt.Printf("    Action: %s - %s\n", step.Action.Type, step.Action.Description)
		}
		if step.Check != nil {
			fmt.Printf("    Check: %s - %s\n", step.Check.Type, step.Check.Description)
		}
		fmt.Print("Press Enter to continue...")
		fmt.Scanln() // Pause execution
		return nil   // Assume manual step is 'completed'
	}

	// 1. Check Condition (`if`)
	if step.If != "" {
		conditionMet, err := ctx.EvaluateCondition(step.If)
		if err != nil {
			slog.Error("Condition evaluation failed", "if", step.If, "error", err)
			return fmt.Errorf("condition evaluation failed for step '%s': %w", step.DSL, err)
		}
		if !conditionMet {
			slog.Info("Skipping step due to unmet condition", "phase", phase, "dsl", step.DSL, "if", step.If)
			return nil // Skip this step
		}
		slog.Debug("Condition met, proceeding", "if", step.If)
	}

	// 2. Handle Loop (`loop`)
	if step.Loop != nil {
		return runLoop(step, phase, ctx) // Delegate loop logic
	}

	// 3. Execute Action or Check (if not a loop container step)
	return runSingleExecution(step, phase, ctx, nil) // Pass nil for last response initially
}

func runLoop(step *poc.Step, phase string, ctx *context.ExecutionContext) error {
	loop := step.Loop
	listVarPath := loop.Over
	loopListName := loop.VariableName

	slog.Debug("Entering loop", "over", listVarPath, "variable", loopListName)

	// Resolve the list variable from context
	listVal, err := ctx.ResolveVariable(listVarPath)
	if err != nil {
		return fmt.Errorf("loop variable '%s' not found or invalid: %w", listVarPath, err)
	}
	listSlice, ok := listVal.([]interface{}) // Assuming list is resolved as slice
	if !ok {
		return fmt.Errorf("loop variable '%s' is not a list/slice", listVarPath)
	}

	for i, item := range listSlice {
		slog.Debug("Loop iteration", "index", i, "variable", loopListName, "value", item)
		// Set loop variable in context (needs proper scoping handling)
		// Use a temporary nested context or careful Set/Unset
		loopVarPath := fmt.Sprintf("loop.%s", loopListName) // Example path for loop var
		ctx.SetVariable(loopVarPath, item)                  // Error handling needed

		if len(loop.Steps) > 0 {
			// Execute nested steps
			for _, nestedStep := range loop.Steps {
				err := RunStep(&nestedStep, phase, ctx) // Recursive call
				if err != nil {
					ctx.SetVariable(loopVarPath, nil) // Clean up loop variable
					return fmt.Errorf("error in loop step '%s' (iteration %d): %w", nestedStep.DSL, i, err)
				}
			}
		} else {
			// Execute the action/check of the main loop step itself
			err := runSingleExecution(step, phase, ctx, nil) // Pass nil response context for loop iteration
			if err != nil {
				ctx.SetVariable(loopVarPath, nil) // Clean up loop variable
				return fmt.Errorf("error in loop action/check '%s' (iteration %d): %w", step.DSL, i, err)
			}
		}
		ctx.SetVariable(loopVarPath, nil) // Clean up loop variable at end of iteration
	}
	slog.Debug("Exiting loop", "over", listVarPath)
	return nil
}

// runSingleExecution handles executing the action or check for a step, including polling for checks.
func runSingleExecution(step *poc.Step, phase string, ctx *context.ExecutionContext, lastResponse interface{}) error {
	var err error
	maxAttempts := 1
	var retryInterval time.Duration

	if step.Check != nil && step.Check.MaxAttempts > 1 {
		maxAttempts = step.Check.MaxAttempts
		if step.Check.RetryInterval != "" {
			retryInterval, err = time.ParseDuration(step.Check.RetryInterval)
			if err != nil {
				slog.Error("Invalid retry_interval duration", "interval", step.Check.RetryInterval, "error", err)
				// Default or return error? Let's default for now.
				retryInterval = time.Second
			}
		} else {
			retryInterval = time.Second // Default interval if not specified
		}
	}

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if step.Action != nil {
			handler, exists := actions.GetRegistry().Get(step.Action.Type)
			if !exists {
				err = fmt.Errorf("unknown action type: %s", step.Action.Type)
			} else {
				slog.Debug("Executing action", "type", step.Action.Type, "attempt", attempt)
				// TODO: Pass lastResponse if action needs it? Usually not.
				err = handler.Execute(step.Action, ctx)
			}
		} else if step.Check != nil {
			handler, err := checks.DefaultRegistry.Get(step.Check.Type)
			if err != nil {
				err = fmt.Errorf("unknown check type: %s", step.Check.Type)
			} else {
				slog.Debug("Evaluating check", "type", step.Check.Type, "attempt", attempt)
				
				// Execute the check
				success, checkErr := handler(ctx, step.Check)
				if checkErr != nil {
					err = fmt.Errorf("check execution failed: %w", checkErr)
				} else if !success {
					err = fmt.Errorf("check failed")
				}
			}
		} else {
			// Should not happen if validation is done, but good to handle.
			slog.Warn("Step has neither action nor check", "dsl", step.DSL)
			return nil // Nothing to do
		}

		if err == nil {
			slog.Info("Step completed successfully", "phase", phase, "dsl", step.DSL, "attempt", attempt)
			return nil // Success
		}

		// Check if it's an expected error
		if step.Check != nil && step.Check.ExpectedError != nil {
			if checkExpectedError(step.Check.ExpectedError, err) {
				slog.Info("Step completed with expected error", "phase", phase, "dsl", step.DSL, "error", err)
				return nil // Expected error occurred, treat as success for flow control
			}
		}

		// If error occurred and it's the last attempt or no retries configured
		if attempt == maxAttempts {
			slog.Error("Step failed", "phase", phase, "dsl", step.DSL, "attempt", attempt, "error", err)
			return fmt.Errorf("step '%s' failed: %w", step.DSL, err) // Final failure
		}

		// If retrying, log and wait
		slog.Warn("Step failed, retrying...", "phase", phase, "dsl", step.DSL, "attempt", attempt, "max_attempts", maxAttempts, "error", err)
		time.Sleep(retryInterval)
	}
	return err // Should technically be unreachable if loop logic is correct
}

// checkExpectedError compares an actual error against the expected error conditions
func checkExpectedError(expected *poc.ExpectedError, actual error) bool {
	if actual == nil {
		return false
	} // No error occurred

	actualStr := actual.Error()
	// TODO: Check status code if actual error wraps an HTTP response error
	// TODO: Check error type string if possible

	if expected.StatusMatches != "" {
		// Logic to extract status code if available and match regex
	}
	if expected.MessageContains != "" {
		if !strings.Contains(actualStr, expected.MessageContains) {
			return false // Does not contain expected message
		}
	}
	// Add more checks for TypeMatches etc.

	// If any defined expected condition is met (could be OR or AND logic depending on spec refinement)
	// Simple OR logic: if MessageContains matches, it's considered expected.
	if expected.MessageContains != "" && strings.Contains(actualStr, expected.MessageContains) {
		return true
	}
	// Add other conditions check here...

	return false // No expected condition matched
}
