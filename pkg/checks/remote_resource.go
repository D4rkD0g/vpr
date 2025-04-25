// Package checks defines the interface and registry for performing checks.
// This file implements the specific check handler for verifying the state or existence
// of a remote resource (`type: check_remote_resource`), potentially via API calls or other
// indirect methods defined by the PoC.
package checks

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
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

	// Extract required parameters
	resourceType := check.ResourceType
	if resourceType == "" {
		// Default to file if not specified
		resourceType = "file"
	}
	
	path := check.Path
	if path == "" {
		return false, fmt.Errorf("check_remote_resource requires 'path' field")
	}

	// Resolve path
	resolvedPath, err := ctx.Substitute(path)
	if err != nil {
		return false, fmt.Errorf("failed to resolve path: %w", err)
	}
	
	// Extract state parameter (exists, not_exists)
	state := check.State
	if state == "" {
		// Default to exists if not specified
		state = "exists"
	}
	
	// Create result map for optional output variable
	result := map[string]interface{}{
		"resource_type": resourceType,
		"path":          resolvedPath,
		"state":         state,
		"exists":        false, // Will be updated based on check result
		"valid":         false, // Will be updated based on check result
	}
	
	// Perform the check based on resource type
	switch strings.ToLower(resourceType) {
	case "file":
		return checkRemoteFile(ctx, check, resolvedPath, state, result)
	case "directory":
		return checkRemoteDirectory(ctx, check, resolvedPath, state, result)
	default:
		return false, fmt.Errorf("unsupported resource_type: %s", resourceType)
	}
}

// checkRemoteFile verifies a remote file using HTTP(S)
func checkRemoteFile(ctx *execContext.ExecutionContext, check *poc.Check, path, state string, result map[string]interface{}) (bool, error) {
	// Create HTTP client
	client, err := utils.GetHTTPClient(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get HTTP client: %w", err)
	}
	
	// Resolve validation parameters
	contentContains, contentEquals, regexPattern, err := resolveValidationParameters(ctx, check)
	if err != nil {
		return false, err
	}
	
	// Add validation parameters to result if present
	if contentContains != "" {
		result["content_contains"] = contentContains
	}
	if contentEquals != "" {
		result["content_equals"] = contentEquals
	}
	if regexPattern != "" {
		result["regex"] = regexPattern
	}
	
	// Create request - use HEAD first to check existence
	method := "HEAD"
	req, err := http.NewRequest(method, path, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set headers
	req.Header.Set("User-Agent", "VPR-PoCRunner/1.0")
	
	// Apply authentication if context is set
	authContext := ctx.GetAuthenticationContext()
	if err = ctx.ApplyAuthentication(req, authContext); err != nil {
		return false, fmt.Errorf("failed to apply authentication: %w", err)
	}
	
	// Send HEAD request to check existence
	resp, err := client.Do(req)
	exists := err == nil && (resp.StatusCode >= 200 && resp.StatusCode < 404)
	
	// If we got a response, close it
	if resp != nil {
		resp.Body.Close()
	}
	
	// Update result with existence information
	result["exists"] = exists
	
	// Handle state check
	if state == "not_exists" {
		isValid := !exists
		result["valid"] = isValid
		
		// Store result if output variable is specified
		storeResult(ctx, check, result)
		
		if !isValid {
			return false, fmt.Errorf("resource exists but expected not to exist: %s", path)
		}
		return true, nil
	}
	
	// For "exists" state, if file doesn't exist, return error
	if !exists {
		storeResult(ctx, check, result)
		return false, fmt.Errorf("resource does not exist: %s", path)
	}
	
	// If we need to check content, we need to make a GET request
	if contentContains != "" || contentEquals != "" || regexPattern != "" {
		getReq, err := http.NewRequest("GET", path, nil)
		if err != nil {
			return false, fmt.Errorf("failed to create GET request: %w", err)
		}
		
		// Set headers
		getReq.Header.Set("User-Agent", "VPR-PoCRunner/1.0")
		
		// Apply authentication
		if err = ctx.ApplyAuthentication(getReq, authContext); err != nil {
			return false, fmt.Errorf("failed to apply authentication: %w", err)
		}
		
		// Send GET request
		getResp, err := client.Do(getReq)
		if err != nil {
			return false, fmt.Errorf("failed to send GET request: %w", err)
		}
		defer getResp.Body.Close()
		
		// Read response body
		body, err := io.ReadAll(getResp.Body)
		if err != nil {
			return false, fmt.Errorf("failed to read response body: %w", err)
		}
		
		// Store status code in context for potential error checking
		ctx.SetLastStatusCode(getResp.StatusCode)
		
		// Check content validation
		isValid, err := validateContent(string(body), contentContains, contentEquals, regexPattern)
		
		// Update result
		result["valid"] = isValid
		if !isValid && err != nil {
			result["error"] = err.Error()
		}
		
		// Store result
		storeResult(ctx, check, result)
		
		if !isValid {
			return false, err
		}
	} else {
		// If no content validation needed, the resource exists and is valid
		result["valid"] = true
		storeResult(ctx, check, result)
	}
	
	return true, nil
}

// checkRemoteDirectory verifies a remote directory
func checkRemoteDirectory(ctx *execContext.ExecutionContext, check *poc.Check, path, state string, result map[string]interface{}) (bool, error) {
	// For directories, we need some way to check if it exists
	// This is highly dependent on the server configuration and may not work in all cases
	
	// If path doesn't end with /, add it to ensure we're requesting a directory
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}
	
	// Create HTTP client
	client, err := utils.GetHTTPClient(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get HTTP client: %w", err)
	}
	
	// Create request
	req, err := http.NewRequest("HEAD", path, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set headers
	req.Header.Set("User-Agent", "VPR-PoCRunner/1.0")
	
	// Apply authentication if context is set
	authContext := ctx.GetAuthenticationContext()
	if err = ctx.ApplyAuthentication(req, authContext); err != nil {
		return false, fmt.Errorf("failed to apply authentication: %w", err)
	}
	
	// Send request
	resp, err := client.Do(req)
	exists := err == nil && (resp.StatusCode >= 200 && resp.StatusCode < 404)
	
	// If we got a response, close it
	if resp != nil {
		resp.Body.Close()
	}
	
	// Update result with existence information
	result["exists"] = exists
	
	// Handle state check
	if state == "not_exists" {
		isValid := !exists
		result["valid"] = isValid
		
		// Store result
		storeResult(ctx, check, result)
		
		if !isValid {
			return false, fmt.Errorf("directory exists but expected not to exist: %s", path)
		}
		return true, nil
	}
	
	// For "exists" state, if directory doesn't exist, return error
	if !exists {
		storeResult(ctx, check, result)
		return false, fmt.Errorf("directory does not exist: %s", path)
	}
	
	// If we reach here, the directory exists and is valid
	result["valid"] = true
	storeResult(ctx, check, result)
	
	return true, nil
}

// resolveValidationParameters extracts and resolves validation parameters
func resolveValidationParameters(ctx *execContext.ExecutionContext, check *poc.Check) (string, string, string, error) {
	var contentContains, contentEquals, regexPattern string
	var err error
	
	// Resolve content_contains if specified
	if check.ContentContains != "" {
		contentContains, err = ctx.Substitute(check.ContentContains)
		if err != nil {
			return "", "", "", fmt.Errorf("failed to resolve content_contains: %w", err)
		}
	}
	
	// Resolve content_equals if specified
	if check.ContentEquals != "" {
		contentEquals, err = ctx.Substitute(check.ContentEquals)
		if err != nil {
			return "", "", "", fmt.Errorf("failed to resolve content_equals: %w", err)
		}
	}
	
	// Resolve regex if specified
	if check.Regex != "" {
		regexPattern, err = ctx.Substitute(check.Regex)
		if err != nil {
			return "", "", "", fmt.Errorf("failed to resolve regex: %w", err)
		}
		
		// Validate regex pattern
		_, err = regexp.Compile(regexPattern)
		if err != nil {
			return "", "", "", fmt.Errorf("invalid regex pattern: %w", err)
		}
	}
	
	return contentContains, contentEquals, regexPattern, nil
}

// validateContent checks if content matches the validation criteria
func validateContent(content, contains, equals, regex string) (bool, error) {
	// Check content_contains
	if contains != "" && !strings.Contains(content, contains) {
		return false, fmt.Errorf("content does not contain expected text: %s", contains)
	}
	
	// Check content_equals
	if equals != "" && content != equals {
		return false, fmt.Errorf("content does not equal expected text: %s", equals)
	}
	
	// Check regex
	if regex != "" {
		matched, err := regexp.MatchString(regex, content)
		if err != nil {
			return false, fmt.Errorf("regex matching error: %w", err)
		}
		if !matched {
			return false, fmt.Errorf("content does not match regex pattern: %s", regex)
		}
	}
	
	return true, nil
}

// storeResult stores the check result in a context variable if specified
func storeResult(ctx *execContext.ExecutionContext, check *poc.Check, result map[string]interface{}) {
	// Try to determine target variable
	targetVar := ""
	
	// Check if path can be used as variable name base
	if check.Path != "" {
		targetVar = check.Path + "_result"
	} else {
		// Use default name
		targetVar = "check_remote_resource_result"
	}
	
	// Store the result
	if targetVar != "" {
		_ = ctx.SetVariable(targetVar, result)
	}
}

// init registers the check_remote_resource check handler
func init() {
	MustRegisterCheck("check_remote_resource", remoteResourceCheck)
}
