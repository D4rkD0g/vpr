// Package actions provides the registry and implementation of all actions supported by the VPR engine.
// This file specifically defines the registry system that allows actions to be registered,
// discovered, and executed during PoC execution.
package actions

import (
	"fmt"
	"sync"
	"vpr/pkg/context"
	"vpr/pkg/poc"
)

// ActionHandler is the function signature for action execution handlers
type ActionHandler func(ctx *context.ExecutionContext, action *poc.Action) (interface{}, error)

// ActionRegistry manages the registration and lookup of action handlers
type ActionRegistry struct {
	mu       sync.RWMutex
	handlers map[string]ActionHandler
}

// NewActionRegistry creates a new empty action registry
func NewActionRegistry() *ActionRegistry {
	return &ActionRegistry{
		handlers: make(map[string]ActionHandler),
	}
}

// Register adds a new action handler to the registry
func (r *ActionRegistry) Register(actionType string, handler ActionHandler) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.handlers[actionType]; exists {
		return fmt.Errorf("action handler for type '%s' is already registered", actionType)
	}

	r.handlers[actionType] = handler
	return nil
}

// MustRegister adds a new action handler to the registry, panicking if it fails
func (r *ActionRegistry) MustRegister(actionType string, handler ActionHandler) {
	if err := r.Register(actionType, handler); err != nil {
		panic(err)
	}
}

// Get retrieves an action handler by type
func (r *ActionRegistry) Get(actionType string) (ActionHandler, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	handler, exists := r.handlers[actionType]
	if !exists {
		return nil, fmt.Errorf("no handler registered for action type '%s'", actionType)
	}

	return handler, nil
}

// Execute runs an action using the appropriate handler
func (r *ActionRegistry) Execute(ctx *context.ExecutionContext, action *poc.Action) (interface{}, error) {
	// Sanity check
	if action == nil {
		return nil, fmt.Errorf("cannot execute nil action")
	}

	if action.Type == "" {
		return nil, fmt.Errorf("action missing required 'type' field")
	}

	// Get the handler
	handler, err := r.Get(action.Type)
	if err != nil {
		return nil, err
	}

	// Execute the action
	return handler(ctx, action)
}

// Global instance for convenience
var DefaultRegistry = NewActionRegistry()

// RegisterAction registers an action handler with the default registry
func RegisterAction(actionType string, handler ActionHandler) error {
	return DefaultRegistry.Register(actionType, handler)
}

// MustRegisterAction registers an action handler with the default registry, panicking if it fails
func MustRegisterAction(actionType string, handler ActionHandler) {
	DefaultRegistry.MustRegister(actionType, handler)
}

// ExecuteAction executes an action using the default registry
func ExecuteAction(ctx *context.ExecutionContext, action *poc.Action) (interface{}, error) {
	return DefaultRegistry.Execute(ctx, action)
}

// InitStandardActions registers all standard actions defined in the DSL specification
func InitStandardActions() {
	// HTTP Actions
	// 注：http_request已在http_request.go中注册，这里不再重复注册

	// Setup & Control Actions
	MustRegisterAction("ensure_users_exist", nil)      // TODO: Implement
	MustRegisterAction("ensure_resource_exists", nil)  // TODO: Implement
	MustRegisterAction("execute_local_commands", nil)  // TODO: Implement
	MustRegisterAction("check_target_availability", nil) // TODO: Implement
	// 注：wait已在wait.go中注册，这里不再重复注册
	// 注：generate_data已在generate_data.go中注册，这里不再重复注册
	MustRegisterAction("manual_action", nil)           // TODO: Implement
}
