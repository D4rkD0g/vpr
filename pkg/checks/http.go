// Package checks implements check handlers for the VPR engine.
// This file contains HTTP response check implementations, including
// status code check, body content check, and header check.
package checks

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	
	"vpr/pkg/context"
	"vpr/pkg/poc"
)

// httpResponseStatusCheck checks if the HTTP response status code matches expected values
func httpResponseStatusCheck(ctx *context.ExecutionContext, check *poc.Check) (bool, error) {
	if check.Type != "http_response_status" {
		return false, fmt.Errorf("invalid check type for httpResponseStatusCheck: %s", check.Type)
	}
	
	// Get the last HTTP response from context
	responseObj, err := ctx.ResolveVariable("last_http_response")
	if err != nil {
		return false, fmt.Errorf("failed to resolve last_http_response: %w", err)
	}
	
	// Cast to our expected response type - using type assertion with validation
	httpResp, ok := responseObj.(map[string]interface{})
	if !ok {
		return false, fmt.Errorf("last_http_response is not a valid HTTP response object")
	}
	
	// Extract status code
	statusCode, ok := httpResp["status_code"].(int)
	if !ok {
		return false, fmt.Errorf("status_code not found in HTTP response or not an integer")
	}
	
	// Different ways to check HTTP status
	switch {
	// Case 1: ExpectedStatus is a single integer
	case check.ExpectedStatus != nil:
		// Handle single value, range, or array of acceptable values
		switch expected := check.ExpectedStatus.(type) {
		case int:
			return statusCode == expected, nil
		case float64: // JSON numbers often come as float64
			return statusCode == int(expected), nil
		case string:
			// Range like "200-299" or array like "200,201,204"
			if strings.Contains(expected, "-") {
				// Range case: "min-max"
				parts := strings.Split(expected, "-")
				if len(parts) != 2 {
					return false, fmt.Errorf("invalid status range format: %s", expected)
				}
				
				min, err := strconv.Atoi(strings.TrimSpace(parts[0]))
				if err != nil {
					return false, fmt.Errorf("invalid min status in range: %s", parts[0])
				}
				
				max, err := strconv.Atoi(strings.TrimSpace(parts[1]))
				if err != nil {
					return false, fmt.Errorf("invalid max status in range: %s", parts[1])
				}
				
				return statusCode >= min && statusCode <= max, nil
			} else if strings.Contains(expected, ",") {
				// List case: "status1,status2,status3"
				validCodes := strings.Split(expected, ",")
				for _, codeStr := range validCodes {
					code, err := strconv.Atoi(strings.TrimSpace(codeStr))
					if err != nil {
						return false, fmt.Errorf("invalid status code in list: %s", codeStr)
					}
					
					if statusCode == code {
						return true, nil
					}
				}
				return false, nil
			} else {
				// Single value as string
				code, err := strconv.Atoi(strings.TrimSpace(expected))
				if err != nil {
					return false, fmt.Errorf("invalid status code: %s", expected)
				}
				return statusCode == code, nil
			}
		case []interface{}:
			// Array of values
			for _, val := range expected {
				switch exp := val.(type) {
				case int:
					if statusCode == exp {
						return true, nil
					}
				case float64:
					if statusCode == int(exp) {
						return true, nil
					}
				case string:
					code, err := strconv.Atoi(strings.TrimSpace(exp))
					if err != nil {
						return false, fmt.Errorf("invalid status code in array: %s", exp)
					}
					if statusCode == code {
						return true, nil
					}
				default:
					return false, fmt.Errorf("unsupported status code type in array")
				}
			}
			return false, nil
		default:
			return false, fmt.Errorf("unsupported expected_status format")
		}
	
	default:
		// No expected status specified, consider any 2XX as success
		return statusCode >= 200 && statusCode < 300, nil
	}
}

// httpResponseBodyCheck checks if the HTTP response body matches the expected content
func httpResponseBodyCheck(ctx *context.ExecutionContext, check *poc.Check) (bool, error) {
	if check.Type != "http_response_body" {
		return false, fmt.Errorf("invalid check type for httpResponseBodyCheck: %s", check.Type)
	}
	
	// Get the last HTTP response from context
	responseObj, err := ctx.ResolveVariable("last_http_response")
	if err != nil {
		return false, fmt.Errorf("failed to resolve last_http_response: %w", err)
	}
	
	// Cast to our expected response type
	httpResp, ok := responseObj.(map[string]interface{})
	if !ok {
		return false, fmt.Errorf("last_http_response is not a valid HTTP response object")
	}
	
	// Extract body content
	body, ok := httpResp["body"].(string)
	if !ok {
		return false, fmt.Errorf("body not found in HTTP response or not a string")
	}
	
	// Different ways to check body content
	switch {
	// Case 1: Body contains a specific string
	case check.Contains != nil:
		// Handle different data types for 'contains'
		switch expected := check.Contains.(type) {
		case string:
			// Simple string contains check
			resolvedExpected, err := ctx.Substitute(expected)
			if err != nil {
				return false, fmt.Errorf("failed to resolve contains value: %w", err)
			}
			return strings.Contains(body, resolvedExpected), nil
		case []interface{}:
			// Must contain all strings in array
			for _, item := range expected {
				itemStr, ok := item.(string)
				if !ok {
					return false, fmt.Errorf("contains array must only contain strings")
				}
				
				resolvedItemStr, err := ctx.Substitute(itemStr)
				if err != nil {
					return false, fmt.Errorf("failed to resolve contains value: %w", err)
				}
				
				if !strings.Contains(body, resolvedItemStr) {
					return false, nil
				}
			}
			return true, nil
		default:
			return false, fmt.Errorf("unsupported contains format")
		}
	
	// Case 2: Body exactly equals expected string
	case check.Equals != nil:
		expectedStr, ok := check.Equals.(string)
		if !ok {
			return false, fmt.Errorf("equals must be a string for body comparison")
		}
		
		resolvedExpected, err := ctx.Substitute(expectedStr)
		if err != nil {
			return false, fmt.Errorf("failed to resolve equals value: %w", err)
		}
		
		return body == resolvedExpected, nil
	
	// Case 3: Body matches a regular expression
	case check.Regex != "":
		resolvedRegex, err := ctx.Substitute(check.Regex)
		if err != nil {
			return false, fmt.Errorf("failed to resolve regex: %w", err)
		}
		
		re, err := regexp.Compile(resolvedRegex)
		if err != nil {
			return false, fmt.Errorf("invalid regex pattern %s: %w", resolvedRegex, err)
		}
		
		return re.MatchString(body), nil
	
	// Case 4: JSON path check
	case check.JSONPath != nil:
		// This requires a JSON parser; for now, we'll delegate to a separate function
		return jsonPathCheck(ctx, check, body)
	
	default:
		// No specific check criteria specified
		return false, fmt.Errorf("http_response_body check requires at least one of: contains, equals, regex, or json_path")
	}
}

// httpResponseHeaderCheck checks if an HTTP response header matches expected values
func httpResponseHeaderCheck(ctx *context.ExecutionContext, check *poc.Check) (bool, error) {
	if check.Type != "http_response_header" {
		return false, fmt.Errorf("invalid check type for httpResponseHeaderCheck: %s", check.Type)
	}
	
	// Ensure we have a header name to check
	if check.HeaderName == "" {
		return false, fmt.Errorf("http_response_header check requires header_name")
	}
	
	// Get the last HTTP response from context
	responseObj, err := ctx.ResolveVariable("last_http_response")
	if err != nil {
		return false, fmt.Errorf("failed to resolve last_http_response: %w", err)
	}
	
	// Cast to our expected response type
	httpResp, ok := responseObj.(map[string]interface{})
	if !ok {
		return false, fmt.Errorf("last_http_response is not a valid HTTP response object")
	}
	
	// Extract headers
	headers, ok := httpResp["headers"].(map[string][]string)
	if !ok {
		return false, fmt.Errorf("headers not found in HTTP response or not a map")
	}
	
	// Get the header value (headers are case-insensitive in HTTP)
	headerName := check.HeaderName
	var headerValue string
	for name, values := range headers {
		if strings.EqualFold(name, headerName) {
			if len(values) > 0 {
				headerValue = values[0]
			}
			break
		}
	}
	
	// If header doesn't exist
	if headerValue == "" {
		// If we're checking for non-existence, that's actually a success
		if check.Equals != nil && check.Equals == "" {
			return true, nil
		}
		return false, nil
	}
	
	// Different ways to check header
	switch {
	// Case 1: Header contains a substring
	case check.Contains != nil:
		containsStr, ok := check.Contains.(string)
		if !ok {
			return false, fmt.Errorf("contains must be a string for header comparison")
		}
		
		resolvedExpected, err := ctx.Substitute(containsStr)
		if err != nil {
			return false, fmt.Errorf("failed to resolve contains value: %w", err)
		}
		
		return strings.Contains(headerValue, resolvedExpected), nil
	
	// Case 2: Header equals expected value
	case check.Equals != nil:
		equalsStr, ok := check.Equals.(string)
		if !ok {
			return false, fmt.Errorf("equals must be a string for header comparison")
		}
		
		resolvedExpected, err := ctx.Substitute(equalsStr)
		if err != nil {
			return false, fmt.Errorf("failed to resolve equals value: %w", err)
		}
		
		return headerValue == resolvedExpected, nil
	
	// Case 3: Header matches regex
	case check.Regex != "":
		resolvedRegex, err := ctx.Substitute(check.Regex)
		if err != nil {
			return false, fmt.Errorf("failed to resolve regex: %w", err)
		}
		
		re, err := regexp.Compile(resolvedRegex)
		if err != nil {
			return false, fmt.Errorf("invalid regex pattern %s: %w", resolvedRegex, err)
		}
		
		return re.MatchString(headerValue), nil
	
	default:
		// No specific check criteria specified
		return false, fmt.Errorf("http_response_header check requires at least one of: contains, equals, or regex")
	}
}

// jsonPathCheck is a helper function for JSON path evaluation (placeholder for now)
func jsonPathCheck(ctx *context.ExecutionContext, check *poc.Check, body string) (bool, error) {
	// To be implemented when we add JSON path support
	// For now, return error
	if check.JSONPath == nil {
		return false, fmt.Errorf("json_path check is missing json_path section")
	}
	
	// Placeholder for JSON path evaluation
	return false, fmt.Errorf("json_path check not yet implemented")
}

// init registers all HTTP response checks
func init() {
	// Register the HTTP response check handlers
	MustRegisterCheck("http_response_status", httpResponseStatusCheck)
	MustRegisterCheck("http_response_body", httpResponseBodyCheck)
	MustRegisterCheck("http_response_header", httpResponseHeaderCheck)
}
