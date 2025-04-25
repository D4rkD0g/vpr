// Package checks implements handlers for verifying expected errors in PoC execution.
package checks

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
	"time"
	
	execContext "vpr/pkg/context"
	"vpr/pkg/poc"
)

// ErrorContext stores additional information about an error that occurred during execution
type ErrorContext struct {
	Message      string                 `json:"message"`
	StatusCode   int                    `json:"status_code,omitempty"`
	ErrorType    string                 `json:"error_type,omitempty"`
	Timestamp    time.Time              `json:"timestamp"`
	Source       string                 `json:"source,omitempty"`
	ResponseBody string                 `json:"response_body,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

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
	
	// Build enhanced error context
	errorContext, err := buildErrorContext(ctx, lastError)
	if err != nil {
		// Non-critical error, log but continue
		fmt.Printf("Warning: failed to build complete error context: %v\n", err)
	}
	
	// Store error context in execution context for later reference
	if err := ctx.SetVariable("_last_error_context", errorContext); err != nil {
		fmt.Printf("Warning: failed to store error context in execution context: %v\n", err)
	}
	
	// Check message_contains if specified
	if expectedError.MessageContains != "" {
		// Resolve any variables in the expected message
		expectedMessage, err := ctx.Substitute(expectedError.MessageContains)
		if err != nil {
			return false, fmt.Errorf("failed to resolve variables in expected_error.message_contains: %w", err)
		}
		
		if !strings.Contains(errorStr, expectedMessage) {
			return false, fmt.Errorf("error message '%s' does not contain expected substring '%s'", errorStr, expectedMessage)
		}
	}
	
	// Check status_matches if specified
	if expectedError.StatusMatches != "" {
		// Try to extract status code from context or error
		statusCode := extractStatusCode(ctx, lastError)
		
		// Resolve any variables in the expected status pattern
		expectedPattern, err := ctx.Substitute(expectedError.StatusMatches)
		if err != nil {
			return false, fmt.Errorf("failed to resolve variables in expected_error.status_matches: %w", err)
		}
		
		// Match using regex
		matched, err := regexp.MatchString(expectedPattern, statusCode)
		if err != nil {
			return false, fmt.Errorf("invalid status pattern '%s': %w", expectedPattern, err)
		}
		if !matched {
			return false, fmt.Errorf("status code '%s' does not match pattern '%s'", statusCode, expectedPattern)
		}
	}
	
	// Check error_type_matches if specified
	if expectedError.ErrorTypeMatches != "" {
		// Try to extract error type from error
		errorType := extractErrorType(lastError)
		
		// Resolve any variables in the expected error type pattern
		expectedPattern, err := ctx.Substitute(expectedError.ErrorTypeMatches)
		if err != nil {
			return false, fmt.Errorf("failed to resolve variables in expected_error.error_type_matches: %w", err)
		}
		
		// Match using regex
		matched, err := regexp.MatchString(expectedPattern, errorType)
		if err != nil {
			return false, fmt.Errorf("invalid error type pattern '%s': %w", expectedPattern, err)
		}
		if !matched {
			return false, fmt.Errorf("error type '%s' does not match pattern '%s'", errorType, expectedPattern)
		}
	}
	
	// All specified criteria passed
	return true, nil
}

// buildErrorContext creates a comprehensive ErrorContext from the error and execution context
func buildErrorContext(ctx *execContext.ExecutionContext, err error) (ErrorContext, error) {
	errorContext := ErrorContext{
		Message:   err.Error(),
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}
	
	// Extract status code
	statusCode := ctx.GetLastStatusCode()
	if statusCode > 0 {
		errorContext.StatusCode = statusCode
	}
	
	// Extract error type
	errorContext.ErrorType = extractErrorType(err)
	
	// Extract source of error (e.g., action name, step ID)
	if lastStep := ctx.GetLastStep(); lastStep != "" {
		errorContext.Source = lastStep
	}
	
	// Extract response body if available
	lastResponse := ctx.GetLastResponse()
	if lastResponse != nil {
		// Only store response body if it's not too large (< 10KB)
		if len(lastResponse) < 10240 {
			errorContext.ResponseBody = string(lastResponse)
			
			// Try to parse as JSON for additional metadata
			var jsonData map[string]interface{}
			if err := json.Unmarshal(lastResponse, &jsonData); err == nil {
				// If successful, extract useful metadata
				for _, key := range []string{"error", "message", "code", "type", "reason"} {
					if val, ok := jsonData[key]; ok {
						errorContext.Metadata[key] = val
					}
				}
			}
		} else {
			errorContext.Metadata["response_size"] = len(lastResponse)
			errorContext.ResponseBody = string(lastResponse[:1024]) + "... [truncated]"
		}
	}
	
	// Collect HTTP-specific information
	if httpErr, ok := err.(*HTTPError); ok {
		errorContext.StatusCode = httpErr.StatusCode
		errorContext.Metadata["url"] = httpErr.URL
		errorContext.Metadata["method"] = httpErr.Method
	}
	
	return errorContext, nil
}

// HTTPError represents an HTTP-specific error with additional context
type HTTPError struct {
	StatusCode int
	URL        string
	Method     string
	Message    string
}

// Error implements the error interface
func (e *HTTPError) Error() string {
	return e.Message
}

// extractStatusCode attempts to extract a status code from the error
func extractStatusCode(ctx *execContext.ExecutionContext, err error) string {
	// Try to get from context first
	if statusCode := ctx.GetLastStatusCode(); statusCode > 0 {
		return fmt.Sprintf("%d", statusCode)
	}
	
	// Check if it's an HTTP error
	if httpErr, ok := err.(*HTTPError); ok {
		return fmt.Sprintf("%d", httpErr.StatusCode)
	}
	
	// Try to parse from error message (this is implementation-specific)
	// Look for standard HTTP status code patterns
	errMsg := err.Error()
	
	// Pattern 1: "status code: 404"
	re := regexp.MustCompile(`status(?:\s+code)?[:\s]+(\d+)`)
	matches := re.FindStringSubmatch(errMsg)
	if len(matches) > 1 {
		return matches[1]
	}
	
	// Pattern 2: "HTTP 404" or "404 Not Found"
	re = regexp.MustCompile(`(?:HTTP\s+)?(\d{3})(?:\s+[A-Za-z\s]+)?`)
	matches = re.FindStringSubmatch(errMsg)
	if len(matches) > 1 {
		return matches[1]
	}
	
	// Check for common HTTP status codes by name
	statusMap := map[string]string{
		"not found":          "404",
		"unauthorized":       "401",
		"forbidden":          "403",
		"bad request":        "400",
		"internal server":    "500",
		"service unavailable": "503",
		"gateway timeout":    "504",
	}
	
	for text, code := range statusMap {
		if strings.Contains(strings.ToLower(errMsg), text) {
			return code
		}
	}
	
	// Default to empty string if not found
	return ""
}

// extractErrorType attempts to determine the type of error
func extractErrorType(err error) string {
	// First, check for specific error types
	switch err.(type) {
	case *HTTPError:
		return "HTTPError"
	case *json.SyntaxError:
		return "JSONSyntaxError"
	case *url.Error:
		return "URLError"
	case *net.OpError:
		return "NetworkError"
	}
	
	// Otherwise return the Go type name
	typeName := fmt.Sprintf("%T", err)
	
	// Clean up common prefixes for readability
	typeName = strings.TrimPrefix(typeName, "*")
	typeName = strings.TrimPrefix(typeName, "vpr/pkg/")
	
	return typeName
}

// init registers the error check handler
func init() {
	MustRegisterCheck("expected_error", errorCheckHandler)
}
