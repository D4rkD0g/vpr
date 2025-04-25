// Package executor implements retry logic for actions and steps
package executor

import (
	"fmt"
	"math"
	"math/rand"
	"time"
	
	"vpr/pkg/actions"
	execContext "vpr/pkg/context"
	"vpr/pkg/poc"
)

// RetryStrategy defines the strategy for retrying actions
type RetryStrategy string

const (
	// RetryStrategyFixed uses a fixed delay between retries
	RetryStrategyFixed RetryStrategy = "fixed"
	
	// RetryStrategyExponential uses exponential backoff between retries
	RetryStrategyExponential RetryStrategy = "exponential"
	
	// RetryStrategyLinear uses linearly increasing delay between retries
	RetryStrategyLinear RetryStrategy = "linear"
)

// RetryConfig holds retry configuration for actions
type RetryConfig struct {
	// MaxRetries is the maximum number of retries
	MaxRetries int
	
	// Strategy is the retry strategy to use
	Strategy RetryStrategy
	
	// Delay is the base delay between retries in seconds
	Delay float64
	
	// MaxDelay is the maximum delay in seconds (for exponential backoff)
	MaxDelay float64
	
	// Jitter adds randomness to the retry delay to avoid thundering herd
	Jitter float64
}

// PollingConfig holds polling configuration for checks
type PollingConfig struct {
	// MaxAttempts is the maximum number of polling attempts
	MaxAttempts int
	
	// Interval is the time between polling attempts
	Interval time.Duration
}

// DefaultRetryConfig returns the default retry configuration for actions
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries: 3,
		Strategy:   RetryStrategyExponential,
		Delay:      1.0,
		MaxDelay:   60.0,
		Jitter:     0.2,
	}
}

// getRetryConfig extracts retry configuration from an action
func getRetryConfig(ctx *execContext.ExecutionContext, action *poc.Action) (RetryConfig, error) {
	// Start with default config
	config := DefaultRetryConfig()
	
	// Set retries if specified in the action
	if action.Retries > 0 {
		config.MaxRetries = action.Retries
	}
	
	// Parse retry delay if specified
	if action.RetryDelay != "" {
		// Try to resolve any variables in the delay string
		resolvedDelay, err := ctx.Substitute(action.RetryDelay)
		if err != nil {
			return config, fmt.Errorf("failed to resolve retry delay: %w", err)
		}
		
		// Parse delay duration (e.g., "1s", "500ms", "2.5s")
		duration, err := time.ParseDuration(resolvedDelay)
		if err != nil {
			return config, fmt.Errorf("invalid retry delay format: %w", err)
		}
		
		// Convert to seconds
		config.Delay = duration.Seconds()
	}
	
	return config, nil
}

// getPollingConfig extracts polling configuration from a check
func getPollingConfig(ctx *execContext.ExecutionContext, check *poc.Check) (PollingConfig, error) {
	config := PollingConfig{
		MaxAttempts: 1, // Default to a single attempt (no polling)
		Interval:    time.Second, // Default interval of 1 second
	}
	
	// Set max attempts if specified
	if check.MaxAttempts > 0 {
		config.MaxAttempts = check.MaxAttempts
	}
	
	// Parse retry interval if specified
	if check.RetryInterval != "" {
		// Try to resolve any variables in the interval string
		resolvedInterval, err := ctx.Substitute(check.RetryInterval)
		if err != nil {
			return config, fmt.Errorf("failed to resolve retry interval: %w", err)
		}
		
		// Parse interval duration
		interval, err := time.ParseDuration(resolvedInterval)
		if err != nil {
			return config, fmt.Errorf("invalid retry interval format: %w", err)
		}
		
		config.Interval = interval
	}
	
	return config, nil
}

// RetryDecision represents the outcome of a retry condition evaluation
type RetryDecision struct {
	ShouldRetry bool
	Reason      string
}

// RetryCondition is a function that evaluates whether to retry based on the result and error
type RetryCondition func(result interface{}, err error) RetryDecision

// DefaultRetryCondition returns the default retry condition that retries on errors only
func DefaultRetryCondition() RetryCondition {
	return func(result interface{}, err error) RetryDecision {
		if err != nil {
			return RetryDecision{
				ShouldRetry: true,
				Reason:      fmt.Sprintf("error occurred: %v", err),
			}
		}
		return RetryDecision{
			ShouldRetry: false,
			Reason:      "operation succeeded",
		}
	}
}

// OrRetryCondition combines multiple retry conditions with OR logic
func OrRetryCondition(conditions ...RetryCondition) RetryCondition {
	return func(result interface{}, err error) RetryDecision {
		for _, condition := range conditions {
			decision := condition(result, err)
			if decision.ShouldRetry {
				return decision
			}
		}
		return RetryDecision{
			ShouldRetry: false,
			Reason:      "no conditions indicated retry is needed",
		}
	}
}

// ResultContainsRetryCondition creates a condition that checks if a result map contains a specific key/value
func ResultContainsRetryCondition(key string, expectedValue interface{}) RetryCondition {
	return func(result interface{}, err error) RetryDecision {
		// If there's an error, let the default handler deal with it
		if err != nil {
			return RetryDecision{
				ShouldRetry: false,
				Reason:      "error occurred, deferring to error condition",
			}
		}
		
		// Check if result is a map
		resultMap, ok := result.(map[string]interface{})
		if !ok {
			return RetryDecision{
				ShouldRetry: false,
				Reason:      fmt.Sprintf("result is not a map, got %T", result),
			}
		}
		
		// Check if key exists
		value, exists := resultMap[key]
		if !exists {
			return RetryDecision{
				ShouldRetry: false,
				Reason:      fmt.Sprintf("key '%s' not found in result", key),
			}
		}
		
		// Check if value matches expected
		if value == expectedValue {
			return RetryDecision{
				ShouldRetry: true,
				Reason:      fmt.Sprintf("key '%s' has expected value '%v'", key, expectedValue),
			}
		}
		
		return RetryDecision{
			ShouldRetry: false,
			Reason:      fmt.Sprintf("key '%s' has value '%v', not matching expected '%v'", key, value, expectedValue),
		}
	}
}

// shouldRetry determines if a retry should be attempted based on the error and configuration
func shouldRetry(ctx *execContext.ExecutionContext, config RetryConfig, actionResult interface{}, err error, attempt int, condition RetryCondition) bool {
	// If max retries reached, don't retry
	if attempt >= config.MaxRetries {
		return false
	}
	
	// If no error, don't retry unless condition indicates otherwise
	if err == nil {
		// Apply custom condition if provided
		if condition != nil {
			decision := condition(actionResult, err)
			return decision.ShouldRetry
		}
		return false
	}
	
	// Default: retry if there was an error
	return true
}

// calculateRetryDelay calculates the delay before the next retry attempt
func calculateRetryDelay(config RetryConfig, attempt int) time.Duration {
	var delaySeconds float64
	
	switch config.Strategy {
	case RetryStrategyFixed:
		delaySeconds = config.Delay
		
	case RetryStrategyExponential:
		// Base^attempt with max limit
		delaySeconds = math.Min(config.Delay*math.Pow(2, float64(attempt)), config.MaxDelay)
		
	case RetryStrategyLinear:
		// Base * attempt with max limit
		delaySeconds = math.Min(config.Delay*float64(attempt+1), config.MaxDelay)
		
	default:
		// Default to fixed
		delaySeconds = config.Delay
	}
	
	// Apply jitter to avoid thundering herd
	if config.Jitter > 0 {
		jitterRange := delaySeconds * config.Jitter
		delaySeconds = delaySeconds - (jitterRange / 2) + (rand.Float64() * jitterRange)
	}
	
	// Ensure delay is never negative
	if delaySeconds < 0 {
		delaySeconds = 0
	}
	
	return time.Duration(delaySeconds * float64(time.Second))
}

// retryAction retries an action with the specified configuration
func retryAction(ctx *execContext.ExecutionContext, action *poc.Action, attempt int, config RetryConfig, condition RetryCondition) (interface{}, error) {
	// Get the action registry
	actionRegistry := actions.DefaultRegistry
	
	// Get action handler
	handler, err := actionRegistry.Get(action.Type)
	if err != nil {
		return nil, fmt.Errorf("action handler not found for type '%s': %w", action.Type, err)
	}
	
	// Execute action
	result, err := handler(ctx, action)
	
	// Check if we should retry
	if shouldRetry(ctx, config, result, err, attempt, condition) && attempt < config.MaxRetries {
		// Calculate delay before retry
		delay := calculateRetryDelay(config, attempt)
		
		// Log retry attempt
		fmt.Printf("[Retry] Attempt %d/%d for action '%s' after %s delay\n", 
			attempt+1, config.MaxRetries, action.Type, delay)
		
		time.Sleep(delay)
		
		// Retry with incremented attempt counter
		return retryAction(ctx, action, attempt+1, config, condition)
	}
	
	return result, err
}

// executeActionWithRetry executes an action with retry logic
func executeActionWithRetry(ctx *execContext.ExecutionContext, action *poc.Action, condition RetryCondition) (interface{}, error) {
	// Get retry configuration
	config, err := getRetryConfig(ctx, action)
	if err != nil {
		return nil, fmt.Errorf("failed to get retry config: %w", err)
	}
	
	// Start with attempt 0
	return retryAction(ctx, action, 0, config, condition)
}

// StepRunner is a function type for executing steps
type StepRunner func(step *poc.Step, phase string, stepNumber int, ctx *execContext.ExecutionContext, options *ExecutorOptions) (*StepResult, error)

// RetryableExecutor wraps a step execution function with retry capability
type RetryableExecutor struct {
	StepExecutor   StepRunner
	Options        *ExecutorOptions
	RetryCondition RetryCondition
}

// NewRetryableExecutor creates a new retryable executor wrapper
func NewRetryableExecutor(executor StepRunner, options *ExecutorOptions) *RetryableExecutor {
	return &RetryableExecutor{
		StepExecutor:   executor,
		Options:        options,
		RetryCondition: DefaultRetryCondition(),
	}
}

// ExecuteStepWithRetry executes a step with retry logic
func (e *RetryableExecutor) ExecuteStepWithRetry(step *poc.Step, phase string, stepNumber int, ctx *execContext.ExecutionContext) (*StepResult, error) {
	// Step only has retry capability if it contains an action
	if step.Action == nil {
		// For steps without actions, just execute normally
		return e.StepExecutor(step, phase, stepNumber, ctx, e.Options)
	}
	
	// For steps with actions, use action retry logic first
	result := &StepResult{
		StartTime: time.Now(),
	}
	
	// Execute action with retry
	actionResult, err := executeActionWithRetry(ctx, step.Action, e.RetryCondition)
	
	// Update result
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime).Seconds()
	result.Output = actionResult
	
	if err != nil {
		result.Error = err
		result.Success = false
	} else {
		result.Success = true
	}
	
	return result, err
}

// ExecuteLoopWithRetry executes a loop with retry capabilities
func (e *RetryableExecutor) ExecuteLoopWithRetry(loop *poc.Loop, phase string, loopID string, ctx *execContext.ExecutionContext) (interface{}, error) {
	// Get the collection to iterate over
	collection, err := ctx.ResolveVariable(loop.Over)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve loop collection '%s': %w", loop.Over, err)
	}
	
	// Convert to slice if needed
	var items []interface{}
	switch v := collection.(type) {
	case []interface{}:
		items = v
	case map[string]interface{}:
		// For maps, iterate over keys
		for k := range v {
			items = append(items, k)
		}
	default:
		return nil, fmt.Errorf("loop collection must be a slice or map, got %T", collection)
	}
	
	// Iterate over items
	var results []interface{}
	for i, item := range items {
		// Create a loop context with the current item
		loopCtx, err := ctx.CreateLoopContext(loop.VariableName, item)
		if err != nil {
			return nil, fmt.Errorf("failed to create loop context: %w", err)
		}
		
		// Execute each step in the loop
		for j, step := range loop.Steps {
			stepResult, err := e.ExecuteStepWithRetry(&step, phase, i*len(loop.Steps)+j, loopCtx)
			if err != nil && e.Options.StopOnFailure {
				return results, fmt.Errorf("loop '%s' step %d failed: %w", loopID, j+1, err)
			}
			results = append(results, stepResult)
		}
	}
	
	return results, nil
}
