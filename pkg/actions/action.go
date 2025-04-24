// Package actions defines the interface for runnable actions within a PoC step
// and provides a registry for different action implementations (e.g., HTTP requests,
// data generation). Each specific action type should implement the Action interface
// and register itself using the provided registry.
package actions

import (
	"sync"
	"vpr/pkg/context"
	"vpr/pkg/poc"
)

type Action interface {
	Execute(params *poc.Action, ctx *context.ExecutionContext) error
}

type actionRegistry struct {
	mu      sync.RWMutex
	actions map[string]Action
}

var registry = &actionRegistry{actions: make(map[string]Action)}

func Register(name string, action Action) {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	if _, exists := registry.actions[name]; exists {
		// Handle duplicate registration error/warning
	}
	registry.actions[name] = action
}

func GetRegistry() *actionRegistry {
	return registry
}

func (r *actionRegistry) Get(name string) (Action, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	action, exists := r.actions[name]
	return action, exists
}

// Implementations (e.g., http_request.go) would call Register in their init()
// func init() { Register("http_request", &HTTPRequestAction{}) }
