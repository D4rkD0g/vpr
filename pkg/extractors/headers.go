// Package extractors provides the implementation for data extraction from responses.
// This file specifically implements header-based extraction from HTTP responses.
package extractors

import (
	"fmt"
	"strings"
	
	"vpr/pkg/context"
	"vpr/pkg/poc"
)

// extractFromHeaderHandler extracts data from HTTP response headers
func extractFromHeaderHandler(ctx *context.ExecutionContext, action *poc.HTTPResponseAction, data interface{}) (interface{}, error) {
	if action.Type != "extract_from_header" {
		return nil, fmt.Errorf("invalid extractor type for extractFromHeaderHandler: %s", action.Type)
	}
	
	// Validate required fields
	if action.HeaderName == "" {
		return nil, fmt.Errorf("extract_from_header requires header_name field")
	}
	
	if action.TargetVariable == "" {
		return nil, fmt.Errorf("extract_from_header requires target_variable field")
	}
	
	// Get the HTTP response
	httpResp, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("data is not an HTTP response")
	}
	
	// Get headers from response
	headers, ok := httpResp["headers"].(map[string][]string)
	if !ok {
		return nil, fmt.Errorf("HTTP response headers not found or invalid type")
	}
	
	// Resolve any variables in the header name
	headerName, err := ctx.Substitute(action.HeaderName)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve header_name: %w", err)
	}
	
	// Look for the header (case-insensitive)
	var headerValue []string
	for name, values := range headers {
		if strings.EqualFold(name, headerName) {
			headerValue = values
			break
		}
	}
	
	// Check if header exists
	if headerValue == nil || len(headerValue) == 0 {
		return nil, fmt.Errorf("header '%s' not found in response", headerName)
	}
	
	// Determine what to return
	if action.ExtractAll {
		// Return all values as an array
		return headerValue, nil
	} else {
		// Return just the first value
		return headerValue[0], nil
	}
}

// init registers header extractor handler
func init() {
	// Register the handler with the extractor registry
	MustRegisterExtractor("extract_from_header", extractFromHeaderHandler)
}
