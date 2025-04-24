// Package checks provides the registry and implementation of all checks supported by the VPR engine.
// This file specifically defines the registry system that allows checks to be registered,
// discovered, and executed during PoC execution.
package checks

import (
	"fmt"
	"sync"
	"vpr/pkg/context"
	"vpr/pkg/poc"
)

// CheckHandler is the function signature for check execution handlers
type CheckHandler func(ctx *context.ExecutionContext, check *poc.Check) (bool, error)

// CheckRegistry manages the registration and lookup of check handlers
type CheckRegistry struct {
	mu       sync.RWMutex
	handlers map[string]CheckHandler
}

// NewCheckRegistry creates a new empty check registry
func NewCheckRegistry() *CheckRegistry {
	return &CheckRegistry{
		handlers: make(map[string]CheckHandler),
	}
}

// Register adds a new check handler to the registry
func (r *CheckRegistry) Register(checkType string, handler CheckHandler) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.handlers[checkType]; exists {
		return fmt.Errorf("check handler for type '%s' is already registered", checkType)
	}

	r.handlers[checkType] = handler
	return nil
}

// MustRegister adds a new check handler to the registry, panicking if it fails
func (r *CheckRegistry) MustRegister(checkType string, handler CheckHandler) {
	if err := r.Register(checkType, handler); err != nil {
		panic(err)
	}
}

// Get retrieves a check handler by type
func (r *CheckRegistry) Get(checkType string) (CheckHandler, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	handler, exists := r.handlers[checkType]
	if !exists {
		return nil, fmt.Errorf("no handler registered for check type '%s'", checkType)
	}

	return handler, nil
}

// Execute runs a check using the appropriate handler
func (r *CheckRegistry) Execute(ctx *context.ExecutionContext, check *poc.Check) (bool, error) {
	// Sanity check
	if check == nil {
		return false, fmt.Errorf("cannot execute nil check")
	}

	if check.Type == "" {
		return false, fmt.Errorf("check missing required 'type' field")
	}

	// Get the handler
	handler, err := r.Get(check.Type)
	if err != nil {
		return false, err
	}

	// Execute the check with retry logic if specified
	maxAttempts := 1
	if check.MaxAttempts > 0 {
		maxAttempts = check.MaxAttempts
	}

	var lastErr error
	var result bool

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		result, lastErr = handler(ctx, check)
		if result || lastErr == nil {
			return result, nil
		}

		// If we have more attempts and retry interval is specified, wait
		if attempt < maxAttempts && check.RetryInterval != "" {
			// TODO: Implement delay based on RetryInterval
			// This would ideally use a time.Duration parsed from RetryInterval
		}
	}

	return false, lastErr
}

// Global instance for convenience
var DefaultRegistry = NewCheckRegistry()

// RegisterCheck registers a check handler with the default registry
func RegisterCheck(checkType string, handler CheckHandler) error {
	return DefaultRegistry.Register(checkType, handler)
}

// MustRegisterCheck registers a check handler with the default registry, panicking if it fails
func MustRegisterCheck(checkType string, handler CheckHandler) {
	DefaultRegistry.MustRegister(checkType, handler)
}

// ExecuteCheck executes a check using the default registry
func ExecuteCheck(ctx *context.ExecutionContext, check *poc.Check) (bool, error) {
	return DefaultRegistry.Execute(ctx, check)
}

// InitStandardChecks registers all standard checks defined in the DSL specification
func InitStandardChecks() {
	// HTTP Response Checks
	MustRegisterCheck("http_response_status", nil)  // TODO: Implement
	MustRegisterCheck("http_response_body", nil)    // TODO: Implement
	MustRegisterCheck("http_response_header", nil)  // TODO: Implement
	
	// Variable Checks
	MustRegisterCheck("variable_equals", nil)       // TODO: Implement
	MustRegisterCheck("variable_contains", nil)     // TODO: Implement
	MustRegisterCheck("variable_regex", nil)        // TODO: Implement
	
	// Path-based checks
	MustRegisterCheck("json_path", nil)             // TODO: Implement
	MustRegisterCheck("check_remote_resource", nil) // TODO: Implement
	
	// These handlers will be implemented in separate files
}
