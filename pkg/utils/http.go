// Package utils provides common utility functions used across the VPR engine.
// This file specifically implements HTTP-related utilities including a common
// HTTP client with consistent configuration.
package utils

import (
	"fmt"
	"net/http"
	"time"
	
	"vpr/pkg/context"
)

// GetHTTPClient returns a configured HTTP client, either from the context or creating a new one
func GetHTTPClient(ctx *context.ExecutionContext) (*http.Client, error) {
	// Check if context already has an HTTP client
	if ctx != nil {
		if client := ctx.GetHTTPClient(); client != nil {
			return client, nil
		}
		
		// Create a new HTTP client with default settings if not present
		client := &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// Allow up to 10 redirects
				if len(via) >= 10 {
					return fmt.Errorf("stopped after 10 redirects")
				}
				return nil
			},
		}
		
		// Store in context
		ctx.SetHTTPClient(client)
		return client, nil
	}
	
	// If no context is provided, return a default client
	return &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Allow up to 10 redirects
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			return nil
		},
	}, nil
}
