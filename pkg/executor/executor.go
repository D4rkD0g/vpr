// Package executor orchestrates the execution of a parsed PoC definition.
// This file contains the main Execute function which runs the different phases
// (Setup, Exploit, Assertions, Verification) in sequence, calling the step runner
// for individual steps within each phase.
package executor

import (
	"fmt"
	"log/slog"
	"time"
	
	"vpr/pkg/actions"
	"vpr/pkg/checks"
	"vpr/pkg/context"
	"vpr/pkg/poc"
)

// ExecutionResult represents the outcome of executing a PoC
type ExecutionResult struct {
	Success    bool        // Overall success/failure
	PocID      string      // PoC identifier
	StartTime  time.Time   // When execution started
	EndTime    time.Time   // When execution completed
	Duration   float64     // Total duration in seconds
	PhaseResults map[string]*PhaseResult // Results by phase
	Error      error       // Overall error (if any)
}

// PhaseResult represents the outcome of executing a specific phase
type PhaseResult struct {
	Name      string    // Phase name (setup, exploit, assertions, verification)
	Success   bool      // Whether phase succeeded
	StartTime time.Time // When phase started
	EndTime   time.Time // When phase completed
	Duration  float64   // Phase duration in seconds
	StepResults []*StepResult // Results of individual steps
	Error     error      // Phase error (if any)
}

// StepResult represents the outcome of executing an individual step
type StepResult struct {
	DSL       string      // Human-readable step description
	Success   bool        // Whether step succeeded 
	StartTime time.Time   // When step started
	EndTime   time.Time   // When step completed
	Duration  float64     // Step duration in seconds
	Error     error       // Step error (if any)
	Output    interface{} // Step output/result data
	Skipped   bool        // Whether step was skipped (due to condition)
}

// ExecutorOptions provides configuration options for the executor
type ExecutorOptions struct {
	StopOnFailure       bool
	Timeout             float64
	LogLevel            string
	CredentialResolvers map[string]interface{}
	VerboseOutput       bool
	Interactive         bool
	DryRun              bool // 设置为true将只显示执行计划而不实际执行
}

// DefaultOptions returns sensible default executor options
func DefaultOptions() *ExecutorOptions {
	return &ExecutorOptions{
		StopOnFailure: true,
		Timeout:       300.0, // 5 minutes
		LogLevel:      "info",
		VerboseOutput: true,
		Interactive:   true,
	}
}

// Execute runs a complete PoC through all phases
func Execute(pocDef *poc.Poc, options *ExecutorOptions) (*ExecutionResult, error) {
	if options == nil {
		options = DefaultOptions()
	}
	
	// 1. Initialize result
	result := &ExecutionResult{
		PocID:        pocDef.Metadata.ID,
		StartTime:    time.Now(),
		PhaseResults: make(map[string]*PhaseResult),
	}
	
	// 2. Set up logging
	slog.Info("Starting PoC execution", 
		"id", pocDef.Metadata.ID, 
		"title", pocDef.Metadata.Title,
		"dsl_version", pocDef.Metadata.DslVersion)
	
	// 3. Create execution context
	ctx, err := context.NewExecutionContext(&pocDef.Context)
	if err != nil {
		return nil, fmt.Errorf("failed to create execution context: %w", err)
	}
	
	// 4. Run phases in sequence according to DSL specification
	
	// 4.1 Setup Phase (Given)
	setupResult, err := executePhase("setup", pocDef.Setup, ctx, options)
	result.PhaseResults["setup"] = setupResult
	if err != nil {
		result.Error = fmt.Errorf("setup phase failed: %w", err)
		if options.StopOnFailure {
			slog.Error("Setup phase failed, stopping execution", "error", err)
			finalizeResult(result)
			return result, result.Error
		}
	}
	
	// 4.2 Exploit Phase (When)
	// 4.2.1 Exploit Setup (Optional)
	if len(pocDef.Exploit.Setup) > 0 {
		exploitSetupResult, err := executePhase("exploit_setup", pocDef.Exploit.Setup, ctx, options)
		result.PhaseResults["exploit_setup"] = exploitSetupResult
		if err != nil && options.StopOnFailure {
			result.Error = fmt.Errorf("exploit setup failed: %w", err)
			slog.Error("Exploit setup failed, stopping execution", "error", err)
			finalizeResult(result)
			return result, result.Error
		}
	}
	
	// 4.2.2 Exploit Steps
	exploitResult, err := executePhase("exploit", pocDef.Exploit.Steps, ctx, options)
	result.PhaseResults["exploit"] = exploitResult
	if err != nil && options.StopOnFailure {
		result.Error = fmt.Errorf("exploit phase failed: %w", err)
		slog.Error("Exploit phase failed, stopping execution", "error", err)
		finalizeResult(result)
		return result, result.Error
	}
	
	// 4.2.3 Exploit Teardown (Optional) - Always attempt even if exploit steps failed
	if len(pocDef.Exploit.Teardown) > 0 {
		teardownResult, _ := executePhase("exploit_teardown", pocDef.Exploit.Teardown, ctx, options)
		result.PhaseResults["exploit_teardown"] = teardownResult
		// Ignore teardown errors for overall success/failure
	}
	
	// 4.3 Assertions Phase (Then)
	assertionsResult, err := executePhase("assertions", pocDef.Assertions, ctx, options)
	result.PhaseResults["assertions"] = assertionsResult
	if err != nil {
		// Assertions failing is a normal part of PoC execution - it means the vulnerability was not exploitable
		if result.Error == nil {
			result.Error = fmt.Errorf("assertions phase failed: %w", err)
		}
		if options.StopOnFailure {
			slog.Info("Assertions phase failed - vulnerability not exploitable", "error", err)
			finalizeResult(result)
			return result, result.Error
		}
	}
	
	// 4.4 Verification Phase (Impact Confirmation) - Optional
	if len(pocDef.Verification) > 0 {
		verificationResult, err := executePhase("verification", pocDef.Verification, ctx, options)
		result.PhaseResults["verification"] = verificationResult
		if err != nil && options.StopOnFailure && result.Error == nil {
			result.Error = fmt.Errorf("verification phase failed: %w", err)
			slog.Error("Verification phase failed", "error", err)
		}
	}
	
	// 5. Finalize result
	finalizeResult(result)
	
	// 6. Determine overall success
	// A successful PoC means the vulnerability was successfully exploited
	// This means all phases executed without error
	if result.Error == nil {
		result.Success = true
		slog.Info("PoC execution successful", 
			"id", pocDef.Metadata.ID, 
			"duration", result.Duration,
			"timestamp", result.EndTime.Format(time.RFC3339))
	} else {
		result.Success = false
		slog.Warn("PoC execution failed", 
			"id", pocDef.Metadata.ID, 
			"error", result.Error,
			"duration", result.Duration,
			"timestamp", result.EndTime.Format(time.RFC3339))
	}
	
	return result, result.Error
}

// executePhase runs all steps in a specific phase
func executePhase(name string, steps []poc.Step, ctx *context.ExecutionContext, options *ExecutorOptions) (*PhaseResult, error) {
	// Skip if no steps
	if len(steps) == 0 {
		return &PhaseResult{
			Name:     name,
			Success:  true,
			StartTime: time.Now(),
			EndTime:  time.Now(),
		}, nil
	}
	
	slog.Info("Starting phase", "phase", name, "steps", len(steps))
	
	// Initialize phase result
	result := &PhaseResult{
		Name:      name,
		StartTime: time.Now(),
		StepResults: make([]*StepResult, 0, len(steps)),
	}
	
	// Execute steps in sequence
	for i, step := range steps {
		stepResult, err := executeStep(&step, name, i+1, ctx, options)
		result.StepResults = append(result.StepResults, stepResult)
		
		if err != nil {
			result.Error = fmt.Errorf("step %d failed: %w", i+1, err)
			break
		}
	}
	
	// Finalize phase result
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime).Seconds()
	result.Success = result.Error == nil
	
	slog.Info("Phase completed", 
		"phase", name, 
		"success", result.Success, 
		"duration", result.Duration)
	
	return result, result.Error
}

// executeStep runs a single step
func executeStep(step *poc.Step, phase string, stepNumber int, ctx *context.ExecutionContext, options *ExecutorOptions) (*StepResult, error) {
	// Initialize step result
	result := &StepResult{
		DSL:      step.DSL,
		StartTime: time.Now(),
	}
	
	// Log step execution
	slog.Info("Executing step", 
		"phase", phase, 
		"step", stepNumber, 
		"dsl", step.DSL)
	
	// 1. Check if step should be executed (condition)
	if step.If != "" {
		conditionMet, err := ctx.EvaluateCondition(step.If)
		if err != nil {
			result.Error = fmt.Errorf("condition evaluation failed: %w", err)
			result.Success = false
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime).Seconds()
			return result, result.Error
		}
		
		if !conditionMet {
			// Skip step if condition not met
			slog.Info("Skipping step - condition not met", 
				"phase", phase, 
				"step", stepNumber, 
				"condition", step.If)
			result.Skipped = true
			result.Success = true
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime).Seconds()
			return result, nil
		}
	}
	
	// 2. Handle manual steps
	if step.Manual {
		slog.Warn("Manual step", "phase", phase, "step", stepNumber, "dsl", step.DSL)
		fmt.Printf("\n-----\nMANUAL STEP REQUIRED:\n%s\n", step.DSL)
		
		// Describe the step details
		if step.Action != nil {
			fmt.Printf("Action Type: %s\n", step.Action.Type)
			if step.Action.Description != "" {
				fmt.Printf("Description: %s\n", step.Action.Description)
			}
		}
		
		if step.Check != nil {
			fmt.Printf("Check Type: %s\n", step.Check.Type)
			if step.Check.Description != "" {
				fmt.Printf("Description: %s\n", step.Check.Description)
			}
		}
		
		// If interactive, wait for user confirmation
		if options.Interactive {
			fmt.Print("\nPress Enter when completed...\n")
			fmt.Scanln()
		}
		
		result.Success = true
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime).Seconds()
		return result, nil
	}
	
	// 3. Handle loop
	if step.Loop != nil {
		// Load collection to iterate over
		collectionPath := step.Loop.Over
		collection, err := ctx.ResolveVariable(collectionPath)
		if err != nil {
			result.Error = fmt.Errorf("failed to resolve loop collection: %w", err)
			result.Success = false
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime).Seconds()
			return result, result.Error
		}
		
		// Get the collection as a slice
		var items []interface{}
		switch v := collection.(type) {
		case []interface{}:
			items = v
		case map[string]interface{}:
			// For maps, iterate over keys
			items = make([]interface{}, 0, len(v))
			for k := range v {
				items = append(items, k)
			}
		default:
			// Try to handle other types
			result.Error = fmt.Errorf("loop collection must be a slice or map, got %T", collection)
			result.Success = false
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime).Seconds()
			return result, result.Error
		}
		
		slog.Info("Starting loop", 
			"phase", phase, 
			"step", stepNumber, 
			"collection", collectionPath, 
			"items", len(items))
		
		// Execute steps for each item
		for i, item := range items {
			// Create a child context with the loop variable
			loopCtx, err := ctx.CreateLoopContext(step.Loop.VariableName, item)
			if err != nil {
				result.Error = fmt.Errorf("failed to create loop context: %w", err)
				break
			}
			
			slog.Info("Loop iteration", 
				"phase", phase, 
				"step", stepNumber, 
				"iteration", i+1, 
				"variable", step.Loop.VariableName)
			
			// Execute loop steps
			loopPhase := fmt.Sprintf("%s_loop_%d_%d", phase, stepNumber, i+1)
			loopResult, loopErr := executePhase(loopPhase, step.Loop.Steps, loopCtx, options)
			_ = loopResult // We're not using the detailed result, just checking for errors
			
			if loopErr != nil && options.StopOnFailure {
				result.Error = fmt.Errorf("loop iteration %d failed: %w", i+1, loopErr)
				break
			}
		}
		
		result.Success = result.Error == nil
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime).Seconds()
		return result, result.Error
	}
	
	// 4. Execute action or check
	if step.Action != nil {
		// Execute action
		output, err := actions.ExecuteAction(ctx, step.Action)
		if err != nil {
			result.Error = fmt.Errorf("action execution failed: %w", err)
			result.Success = false
		} else {
			result.Output = output
			result.Success = true
		}
	} else if step.Check != nil {
		// Execute check
		success, err := checks.ExecuteCheck(ctx, step.Check)
		if err != nil {
			result.Error = fmt.Errorf("check execution failed: %w", err)
			result.Success = false
		} else {
			result.Success = success
			if !success {
				result.Error = fmt.Errorf("check condition not satisfied")
			}
		}
	} else {
		// Neither action nor check specified
		result.Error = fmt.Errorf("step must have either an action or a check")
		result.Success = false
	}
	
	// Finalize step result
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime).Seconds()
	
	// Log step result
	slog.Info("Step completed", 
		"phase", phase, 
		"step", stepNumber, 
		"success", result.Success, 
		"duration", result.Duration)
	
	return result, result.Error
}

// finalizeResult completes the result structure with timing information
func finalizeResult(result *ExecutionResult) {
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime).Seconds()
}
