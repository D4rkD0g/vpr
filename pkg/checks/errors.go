// Package checks implements handlers for verifying expected errors in PoC execution.
package checks

import (
	"fmt"
	"regexp"
	"strings"
	
	execContext "vpr/pkg/context"
	"vpr/pkg/poc"
)

// errorCheckHandler implements the expected_error check
func errorCheckHandler(ctx *execContext.ExecutionContext, check *poc.Check) (bool, error) {
	// Get latest error from context
	lastError := ctx.GetLastError()
	if lastError == nil {
		return false, fmt.Errorf("expected error but none occurred")
	}
	
	// Grab the expected error details from the check
	expectedError := check.ExpectedError
	
	// If no specific error details are provided, any error passes
	if expectedError == nil {
		return true, nil
	}
	
	// Convert error to string
	errorStr := lastError.Error()
	
	// Check message_contains if specified
	if expectedError.MessageContains != "" {
		if !strings.Contains(errorStr, expectedError.MessageContains) {
			return false, fmt.Errorf("error message '%s' does not contain expected substring '%s'", errorStr, expectedError.MessageContains)
		}
	}
	
	// Check status_matches if specified
	if expectedError.StatusMatches != "" {
		// Try to extract status code from context or error
		statusCode := extractStatusCode(ctx, lastError)
		
		// Match using regex
		matched, err := regexp.MatchString(expectedError.StatusMatches, statusCode)
		if err != nil {
			return false, fmt.Errorf("invalid status pattern '%s': %w", expectedError.StatusMatches, err)
		}
		if !matched {
			return false, fmt.Errorf("status code '%s' does not match pattern '%s'", statusCode, expectedError.StatusMatches)
		}
	}
	
	// Check error_type_matches if specified
	if expectedError.ErrorTypeMatches != "" {
		// Try to extract error type from error
		errorType := extractErrorType(lastError)
		
		// Match using regex
		matched, err := regexp.MatchString(expectedError.ErrorTypeMatches, errorType)
		if err != nil {
			return false, fmt.Errorf("invalid error type pattern '%s': %w", expectedError.ErrorTypeMatches, err)
		}
		if !matched {
			return false, fmt.Errorf("error type '%s' does not match pattern '%s'", errorType, expectedError.ErrorTypeMatches)
		}
	}
	
	// All specified criteria passed
	return true, nil
}

// extractStatusCode attempts to extract a status code from the error
func extractStatusCode(ctx *execContext.ExecutionContext, err error) string {
	// Try to get from context first
	if statusCode := ctx.GetLastStatusCode(); statusCode > 0 {
		return fmt.Sprintf("%d", statusCode)
	}
	
	// Try to parse from error message (this is implementation-specific)
	// Example: look for patterns like "status code: 404" in the error message
	errMsg := err.Error()
	re := regexp.MustCompile(`status(?:\s+code)?[:\s]+(\d+)`)
	matches := re.FindStringSubmatch(errMsg)
	if len(matches) > 1 {
		return matches[1]
	}
	
	// Default to empty string if not found
	return ""
}

// extractErrorType attempts to determine the type of error
func extractErrorType(err error) string {
	// Implementation-specific logic to extract error type
	// For now, just return the type name of the error
	return fmt.Sprintf("%T", err)
}

// init registers the error check handler
func init() {
	MustRegisterCheck("expected_error", errorCheckHandler)
}
