// Package extractors provides the implementation for data extraction from responses.
// This file specifically implements header-based extraction.
package extractors

import (
	"fmt"
	"log"
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
	
	log.Printf("DEBUG: Extracting from header - target_variable='%s', header_name='%s'",
		action.TargetVariable, action.HeaderName)
	
	// Get the HTTP response data
	httpResp, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("data is not an HTTP response")
	}
	
	// Get headers from the response
	headers, ok := httpResp["headers"].(map[string][]string)
	if !ok {
		return nil, fmt.Errorf("HTTP response headers are not in expected format")
	}
	
	// Headers are typically case-insensitive, so we need to search case-insensitively
	headerName := strings.ToLower(action.HeaderName)
	var headerValue interface{}
	var found bool
	
	for name, values := range headers {
		if strings.ToLower(name) == headerName {
			if len(values) > 0 {
				if action.ExtractAll {
					// Return all values as an array
					headerValue = values
				} else {
					// Return just the first value
					headerValue = values[0]
				}
				found = true
				break
			}
		}
	}
	
	if !found {
		return nil, fmt.Errorf("header '%s' not found in response", action.HeaderName)
	}
	
	// Create a proper variable structure that matches the ContextVariable format
	varStruct := &poc.ContextVariable{
		ID:    action.TargetVariable,
		Value: headerValue,
	}
	
	// Store the extracted data in the variables map
	varsPath := "variables." + action.TargetVariable
	if err := ctx.SetVariable(varsPath, varStruct); err != nil {
		return nil, fmt.Errorf("failed to set target variable: %w", err)
	}
	
	log.Printf("DEBUG: Header extraction successful - target_variable='%s', extracted_value='%v'", 
		action.TargetVariable, headerValue)
	
	return headerValue, nil
}

// init registers header extractor handler
func init() {
	// Register the handler with the extractor registry
	MustRegisterExtractor("extract_from_header", extractFromHeaderHandler)
}
