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
	
	// RetryStrategyRandom uses random delay between retries within a range
	RetryStrategyRandom RetryStrategy = "random"
)

// RetryConfig holds retry configuration
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
	
	// ConditionExpression is evaluated to determine if retry should be attempted
	ConditionExpression string
}

// DefaultRetryConfig returns the default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries: 3,
		Strategy:   RetryStrategyExponential,
		Delay:      1.0,
		MaxDelay:   60.0,
		Jitter:     0.2,
	}
}

// getRetryConfig extracts retry configuration from an action or step
func getRetryConfig(ctx *execContext.ExecutionContext, actionOrStep interface{}) (RetryConfig, error) {
	// Start with default config
	config := DefaultRetryConfig()
	
	var retries int
	var retryStrategy string
	var retryDelay string
	var retryIf string
	
	// Extract values based on type
	switch v := actionOrStep.(type) {
	case *poc.Action:
		retries = v.Retries
		retryStrategy = v.RetryStrategy
		retryDelay = v.RetryDelay
		retryIf = v.RetryIf
		
	case *poc.Step:
		retries = v.Retries
		retryStrategy = v.RetryStrategy
		retryDelay = v.RetryDelay
		retryIf = v.RetryIf
		
	case *poc.Loop:
		retries = v.Retries
		retryStrategy = v.RetryStrategy
		retryDelay = v.RetryDelay
		retryIf = v.RetryIf
		
	default:
		return config, fmt.Errorf("unsupported type for retry configuration: %T", actionOrStep)
	}
	
	// Set retries if specified
	if retries > 0 {
		config.MaxRetries = retries
	}
	
	// Set strategy if specified
	if retryStrategy != "" {
		resolvedStrategy, err := ctx.Substitute(retryStrategy)
		if err != nil {
			return config, fmt.Errorf("failed to resolve retry strategy: %w", err)
		}
		
		switch strings.ToLower(resolvedStrategy) {
		case "fixed", "constant":
			config.Strategy = RetryStrategyFixed
		case "exponential", "backoff":
			config.Strategy = RetryStrategyExponential
		case "linear", "incremental":
			config.Strategy = RetryStrategyLinear
		case "random", "jitter":
			config.Strategy = RetryStrategyRandom
		default:
			return config, fmt.Errorf("unsupported retry strategy: %s", resolvedStrategy)
		}
	}
	
	// Set delay if specified
	if retryDelay != "" {
		resolvedDelay, err := ctx.Substitute(retryDelay)
		if err != nil {
			return config, fmt.Errorf("failed to resolve retry delay: %w", err)
		}
		
		// Check if it's a duration (e.g., "2s", "500ms")
		duration, err := time.ParseDuration(resolvedDelay)
		if err == nil {
			config.Delay = duration.Seconds()
		} else {
			// Try to parse as float
			var delay float64
			_, err = fmt.Sscanf(resolvedDelay, "%f", &delay)
			if err != nil {
				return config, fmt.Errorf("invalid retry delay format: %s", resolvedDelay)
			}
			config.Delay = delay
		}
	}
	
	// Set condition if specified
	if retryIf != "" {
		config.ConditionExpression = retryIf
	}
	
	return config, nil
}

// shouldRetry determines if a retry should be attempted based on the error and configuration
func shouldRetry(ctx *execContext.ExecutionContext, config RetryConfig, actionResult interface{}, err error, attempt int) bool {
	// If max retries reached, don't retry
	if attempt >= config.MaxRetries {
		return false
	}
	
	// If no error and no condition expression, don't retry
	if err == nil && config.ConditionExpression == "" {
		return false
	}
	
	// If a retry condition is specified, evaluate it
	if config.ConditionExpression != "" {
		// Store error and result in context for condition evaluation
		ctx.SetVariable("_retry_error", err)
		ctx.SetVariable("_retry_result", actionResult)
		ctx.SetVariable("_retry_attempt", attempt)
		
		// Evaluate condition
		result, evalErr := ctx.EvaluateCondition(config.ConditionExpression)
		if evalErr != nil {
			// If condition evaluation fails, default to basic error check
			return err != nil
		}
		
		return result
	}
	
	// Default: retry if there was an error
	return err != nil
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
		
	case RetryStrategyRandom:
		// Random delay between base and max
		min := config.Delay
		max := config.MaxDelay
		if max <= min {
			max = min * 2
		}
		delaySeconds = min + rand.Float64()*(max-min)
		
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
