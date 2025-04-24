// Package credentials provides methods for resolving and managing authentication credentials.
// This file implements basic credential resolver functionality.
package credentials

import (
	"fmt"
	"sync"
)

// BasicResolver is a simple in-memory credential resolver
// It's primarily used for testing or simple use cases
type BasicResolver struct {
	name      string
	store     map[string]map[string]string
	mu        sync.RWMutex
}

// NewBasicResolver creates a new basic credential resolver
func NewBasicResolver(name string) *BasicResolver {
	return &BasicResolver{
		name:  name,
		store: make(map[string]map[string]string),
	}
}

// Name returns the resolver's name
func (r *BasicResolver) Name() string {
	return r.name
}

// Resolve implements the CredentialResolver interface
func (r *BasicResolver) Resolve(credentialRef string) (map[string]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	creds, ok := r.store[credentialRef]
	if !ok {
		return nil, fmt.Errorf("credentials for reference '%s' not found", credentialRef)
	}
	
	// Return a copy to prevent modification of the original
	result := make(map[string]string, len(creds))
	for k, v := range creds {
		result[k] = v
	}
	
	return result, nil
}

// AddCredentials adds or updates credentials in the store
func (r *BasicResolver) AddCredentials(credentialRef string, credentials map[string]string) error {
	if credentialRef == "" {
		return fmt.Errorf("credential reference cannot be empty")
	}
	
	if credentials == nil {
		return fmt.Errorf("credentials cannot be nil")
	}
	
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// Store a copy to prevent modification of the original
	credsMap := make(map[string]string, len(credentials))
	for k, v := range credentials {
		credsMap[k] = v
	}
	
	r.store[credentialRef] = credsMap
	return nil
}

// RemoveCredentials removes credentials from the store
func (r *BasicResolver) RemoveCredentials(credentialRef string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	delete(r.store, credentialRef)
}

// Clear removes all credentials from the store
func (r *BasicResolver) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.store = make(map[string]map[string]string)
}
