// Package credentials provides methods for resolving and managing authentication credentials.
// This file defines credential resolver interfaces and registry functionality.
package credentials

import (
	"fmt"
	"sync"
)

// CredentialResolver defines the interface for resolving credentials by reference
type CredentialResolver interface {
	// Resolve attempts to resolve credentials from a provided reference ID
	// Returns a map of credential attributes (e.g., username, password, token, cookie)
	Resolve(credentialRef string) (map[string]string, error)
	
	// Name returns the name of this resolver for registration and logging
	Name() string
}

// Provider manages credential resolvers
type Provider struct {
	resolvers  []CredentialResolver
	mu         sync.RWMutex
}

// NewProvider creates a new credential provider
func NewProvider() *Provider {
	return &Provider{
		resolvers: make([]CredentialResolver, 0),
	}
}

// RegisterResolver adds a credential resolver to the provider
func (p *Provider) RegisterResolver(resolver CredentialResolver) error {
	if resolver == nil {
		return fmt.Errorf("cannot register nil resolver")
	}
	
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// Check for duplicate resolver name
	for _, r := range p.resolvers {
		if r.Name() == resolver.Name() {
			return fmt.Errorf("credential resolver with name '%s' already registered", resolver.Name())
		}
	}
	
	p.resolvers = append(p.resolvers, resolver)
	return nil
}

// ResolveCredentials attempts to resolve credentials using registered resolvers
func (p *Provider) ResolveCredentials(credentialRef string) (map[string]string, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	if len(p.resolvers) == 0 {
		return nil, fmt.Errorf("no credential resolvers registered")
	}
	
	// Try each resolver in order
	var lastErr error
	for _, resolver := range p.resolvers {
		creds, err := resolver.Resolve(credentialRef)
		if err == nil && creds != nil {
			return creds, nil
		}
		lastErr = err
	}
	
	return nil, fmt.Errorf("failed to resolve credentials '%s': %w", credentialRef, lastErr)
}

// HasResolvers checks if any credential resolvers are registered
func (p *Provider) HasResolvers() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.resolvers) > 0
}
