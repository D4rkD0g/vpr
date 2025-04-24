// Package checks defines the interface and registry for performing checks.
// This file implements the specific check handler for verifying the state or existence
// of a remote resource (`type: remote_resource`), potentially via API calls or other
// indirect methods defined by the PoC.
package checks

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	
	execContext "vpr/pkg/context"
	"vpr/pkg/poc"
	"vpr/pkg/utils"
)

// remoteResourceCheck implements the check_remote_resource check
func remoteResourceCheck(ctx *execContext.ExecutionContext, check *poc.Check) (bool, error) {
	if check.Type != "check_remote_resource" {
		return false, fmt.Errorf("invalid check type for remoteResourceCheck: %s", check.Type)
	}

	// Extract parameters based on Check structure fields
	endpoint := check.Path
	if endpoint == "" {
		return false, fmt.Errorf("check_remote_resource requires 'path' field for endpoint URL")
	}

	// Resolve endpoint URL
	resolvedEndpoint, err := ctx.Substitute(endpoint)
	if err != nil {
		return false, fmt.Errorf("failed to resolve endpoint: %w", err)
	}

	// Get method (use default GET)
	method := "GET"

	// Get expected status (optional)
	expectedStatus := 0 // 0 means any 2xx status
	if check.ExpectedStatus != nil {
		switch v := check.ExpectedStatus.(type) {
		case int:
			expectedStatus = v
		case float64:
			expectedStatus = int(v)
		case string:
			resolvedStatus, err := ctx.Substitute(v)
			if err != nil {
				return false, fmt.Errorf("failed to resolve expected_status: %w", err)
			}
			_, err = fmt.Sscanf(resolvedStatus, "%d", &expectedStatus)
			if err != nil {
				return false, fmt.Errorf("invalid expected_status format: %s", resolvedStatus)
			}
		}
	}

	// Get content validation based on Check fields
	contentContains := check.ContentContains
	contentEquals := check.ContentEquals
	
	// Create HTTP client
	client, err := utils.GetHTTPClient(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get HTTP client: %w", err)
	}

	// Create request
	req, err := http.NewRequest(method, resolvedEndpoint, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("User-Agent", "VPR-PoCRunner/1.0")

	// Apply authentication if context is set
	authContext := ctx.GetAuthenticationContext()
	err = ctx.ApplyAuthentication(req, authContext)
	if err != nil {
		return false, fmt.Errorf("failed to apply authentication: %w", err)
	}

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("resource check request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read response body: %w", err)
	}

	// Store status code in context for potential error checking
	ctx.SetLastStatusCode(resp.StatusCode)

	// Validate expected status
	if expectedStatus > 0 && resp.StatusCode != expectedStatus {
		return false, fmt.Errorf("status code %d does not match expected %d", resp.StatusCode, expectedStatus)
	}

	// If no specific status is specified, check if it's a success (2xx)
	if expectedStatus == 0 && (resp.StatusCode < 200 || resp.StatusCode >= 300) {
		return false, fmt.Errorf("status code %d is not a success status", resp.StatusCode)
	}

	// Validate content contains if specified
	if contentContains != "" {
		resolvedContent, err := ctx.Substitute(contentContains)
		if err != nil {
			return false, fmt.Errorf("failed to resolve content_contains: %w", err)
		}

		if !strings.Contains(string(body), resolvedContent) {
			return false, fmt.Errorf("response does not contain expected content: %s", resolvedContent)
		}
	}

	// Validate content equals if specified
	if contentEquals != "" {
		resolvedContent, err := ctx.Substitute(contentEquals)
		if err != nil {
			return false, fmt.Errorf("failed to resolve content_equals: %w", err)
		}

		if string(body) != resolvedContent {
			return false, fmt.Errorf("response does not equal expected content: %s", resolvedContent)
		}
	}

	return true, nil
}

// init registers the check_remote_resource check handler
func init() {
	MustRegisterCheck("check_remote_resource", remoteResourceCheck)
}
