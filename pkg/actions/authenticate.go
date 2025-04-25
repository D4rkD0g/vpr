// Package actions defines the interface and registry for runnable actions.
// This file implements the specific action handler for performing authentication
// (`type: authenticate`), such as submitting login forms or handling specific
// auth flows, storing the resulting session context (e.g., cookies, tokens).
package actions

import (
	"fmt"
	"log/slog"

	"vpr/pkg/context"
	"vpr/pkg/poc"
)

// AuthenticateHandler implements the authenticate action type as defined in the DSL specification.
// It supports multiple authentication types:
// - form: Traditional web form login
// - oauth2_client_credentials: OAuth2 client credentials flow
// - oauth2_password: OAuth2 password grant flow
// - basic: HTTP Basic authentication
// - api_key: API key authentication
func AuthenticateHandler(ctx *context.ExecutionContext, action *poc.Action) (interface{}, error) {
	// Validation
	if action.Type != "authenticate" {
		return nil, fmt.Errorf("invalid action type for AuthenticateHandler: %s", action.Type)
	}
	
	// The existing authenticationHandler in authentication.go has the core implementation
	// Just ensure we're logging the attempt properly
	slog.Info("Executing authentication",
		"auth_type", action.AuthType,
		"user_context", action.AuthenticationContext)
		
	// Call the implementation from authentication.go
	result, err := authenticationHandler(ctx, action)
	
	if err != nil {
		slog.Error("Authentication failed",
			"auth_type", action.AuthType,
			"error", err)
		return nil, fmt.Errorf("authentication failed: %w", err)
	}
	
	slog.Info("Authentication successful",
		"auth_type", action.AuthType,
		"user_context", action.AuthenticationContext)
		
	return result, nil
}

func init() {
	// Register the authenticate action handler
	MustRegisterAction("authenticate", AuthenticateHandler)
}
