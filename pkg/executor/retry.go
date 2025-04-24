// Package executor implements the PoC execution engine.
// This file implements enhanced retry strategies for handling transient failures.
package executor

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"
	
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

// shouldRetry determines if a retry should be attempted based on the error and configuration
func shouldRetry(ctx *execContext.ExecutionContext, config RetryConfig, actionResult interface{}, err error, attempt int) bool {
	// If max retries reached, don't retry
	if attempt >= config.MaxRetries {
		return false
	}
	
	// If no error, don't retry
	if err == nil {
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
func retryAction(ctx *execContext.ExecutionContext, action *poc.Action, handler ActionHandler, attempt int, config RetryConfig) (interface{}, error) {
	// Execute the action
	result, err := handler(ctx, action)
	
	// Check if we should retry
	if shouldRetry(ctx, config, result, err, attempt) {
		// Calculate delay
		delay := calculateRetryDelay(config, attempt)
		
		// Log retry attempt
		fmt.Printf("Retrying action '%s' (attempt %d/%d) after %v\n", action.Type, attempt+1, config.MaxRetries, delay)
		
		// Wait before retry
		time.Sleep(delay)
		
		// Increment attempt counter and retry
		return retryAction(ctx, action, handler, attempt+1, config)
	}
	
	// Return final result
	return result, err
}

// executeActionWithRetry executes an action with retry logic
func executeActionWithRetry(ctx *execContext.ExecutionContext, action *poc.Action) (interface{}, error) {
	// Get handler for action type
	handler, err := GetActionHandler(action.Type)
	if err != nil {
		return nil, err
	}
	
	// Get retry configuration
	config, err := getRetryConfig(ctx, action)
	if err != nil {
		return nil, err
	}
	
	// If retries are specified, use retry logic
	if config.MaxRetries > 0 {
		return retryAction(ctx, action, handler, 0, config)
	}
	
	// Otherwise, execute directly
	return handler(ctx, action)
}

// executeStepWithRetry executes a step with retry logic
func executeStepWithRetry(ctx *execContext.ExecutionContext, step *poc.Step) (interface{}, error) {
	// Get retry configuration
	config, err := getRetryConfig(ctx, step)
	if err != nil {
		return nil, err
	}
	
	attempt := 0
	var stepResult interface{}
	var stepErr error
	
	// Execute step with retries
	for attempt <= config.MaxRetries {
		if attempt > 0 {
			// Calculate delay for retry
			delay := calculateRetryDelay(config, attempt-1)
			fmt.Printf("Retrying step '%s' (attempt %d/%d) after %v\n", step.Name, attempt, config.MaxRetries, delay)
			time.Sleep(delay)
		}
		
		// Execute step
		stepResult, stepErr = executeStep(ctx, step)
		
		// Check if we should retry
		if !shouldRetry(ctx, config, stepResult, stepErr, attempt) {
			break
		}
		
		attempt++
	}
	
	return stepResult, stepErr
}

// executeLoopWithRetry executes a loop with retry logic
func executeLoopWithRetry(ctx *execContext.ExecutionContext, loop *poc.Loop) (interface{}, error) {
	// Get retry configuration
	config, err := getRetryConfig(ctx, loop)
	if err != nil {
		return nil, err
	}
	
	attempt := 0
	var loopResult interface{}
	var loopErr error
	
	// Execute loop with retries
	for attempt <= config.MaxRetries {
		if attempt > 0 {
			// Calculate delay for retry
			delay := calculateRetryDelay(config, attempt-1)
			fmt.Printf("Retrying loop '%s' (attempt %d/%d) after %v\n", loop.Name, attempt, config.MaxRetries, delay)
			time.Sleep(delay)
		}
		
		// Execute loop
		loopResult, loopErr = executeLoop(ctx, loop)
		
		// Check if we should retry
		if !shouldRetry(ctx, config, loopResult, loopErr, attempt) {
			break
		}
		
		attempt++
	}
	
	return loopResult, loopErr
}
