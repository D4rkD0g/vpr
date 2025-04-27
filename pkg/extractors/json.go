// Package extractors provides the implementation for data extraction from responses.
// This file specifically implements JSON-based extraction using JSONPath.
package extractors

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	
	"vpr/pkg/context"
	"vpr/pkg/poc"
)

// extractFromJSONHandler extracts data from JSON responses using JSONPath
func extractFromJSONHandler(ctx *context.ExecutionContext, action *poc.HTTPResponseAction, data interface{}) (interface{}, error) {
	if action.Type != "extract_from_json" {
		return nil, fmt.Errorf("invalid extractor type for extractFromJSONHandler: %s", action.Type)
	}
	
	// Validate required fields
	if action.JSONPath == "" {
		return nil, fmt.Errorf("extract_from_json requires json_path field")
	}
	
	if action.TargetVariable == "" {
		return nil, fmt.Errorf("extract_from_json requires target_variable field")
	}
	
	// Get the source data
	var jsonData map[string]interface{}
	
	// Determine source of JSON data
	if action.Source != "" {
		// Source specified, try to resolve from context
		sourceData, err := ctx.ResolveVariable(action.Source)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve source variable '%s': %w", action.Source, err)
		}
		
		// Convert source to map if it's already a map
		if mapData, ok := sourceData.(map[string]interface{}); ok {
			jsonData = mapData
		} else if strData, ok := sourceData.(string); ok {
			// Try to parse string as JSON
			if err := json.Unmarshal([]byte(strData), &jsonData); err != nil {
				return nil, fmt.Errorf("failed to parse source as JSON: %w", err)
			}
		} else {
			return nil, fmt.Errorf("source data is neither a map nor a JSON string")
		}
	} else if httpResp, ok := data.(map[string]interface{}); ok {
		// No source specified, use the data from HTTP response
		// Try to parse the response body as JSON
		if bodyStr, ok := httpResp["body"].(string); ok {
			if err := json.Unmarshal([]byte(bodyStr), &jsonData); err != nil {
				return nil, fmt.Errorf("failed to parse response body as JSON: %w", err)
			}
		} else {
			return nil, fmt.Errorf("HTTP response body is not a string")
		}
	} else {
		return nil, fmt.Errorf("data is not an HTTP response")
	}
	
	// Resolve any variables in the JSONPath
	jsonPath, err := ctx.Substitute(action.JSONPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve json_path: %w", err)
	}
	
	// Extract data using JSONPath
	result, err := extractJSONPath(jsonData, jsonPath)
	if err != nil {
		return nil, fmt.Errorf("failed to extract data using JSONPath: %w", err)
	}
	
	// Create a proper variable structure that matches the ContextVariable format
	varStruct := &poc.ContextVariable{
		ID:    action.TargetVariable,
		Value: result,
	}
	
	// Store the extracted data in the variables map
	varsPath := "variables." + action.TargetVariable
	if err := ctx.SetVariable(varsPath, varStruct); err != nil {
		return nil, fmt.Errorf("failed to set target variable: %w", err)
	}
	
	log.Printf("DEBUG: JSON extraction successful - target_variable='%s', extracted_value='%v'", 
		action.TargetVariable, result)
	
	return result, nil
}

// extractJSONPath extracts data from a JSON object using a JSONPath expression
// This is a simplified implementation that supports basic path components
func extractJSONPath(data map[string]interface{}, path string) (interface{}, error) {
	// Normalize path
	path = strings.TrimPrefix(path, "$")
	path = strings.TrimPrefix(path, ".")
	
	// Split path into components
	components := strings.Split(path, ".")
	
	// Navigate through the components
	var current interface{} = data
	
	for i, component := range components {
		// Handle array indexing
		if strings.Contains(component, "[") && strings.Contains(component, "]") {
			// Extract array name and index
			openBracket := strings.Index(component, "[")
			closeBracket := strings.Index(component, "]")
			
			if openBracket == -1 || closeBracket == -1 || closeBracket <= openBracket {
				return nil, fmt.Errorf("invalid array index format in component: %s", component)
			}
			
			arrayName := component[:openBracket]
			indexStr := component[openBracket+1 : closeBracket]
			
			// Get the array
			var array []interface{}
			
			// Handle empty array name (root array)
			if arrayName == "" {
				if arr, ok := current.([]interface{}); ok {
					array = arr
				} else {
					return nil, fmt.Errorf("expected array at path component %d", i)
				}
			} else {
				// Get named array from object
				if obj, ok := current.(map[string]interface{}); ok {
					val, exists := obj[arrayName]
					if !exists {
						return nil, fmt.Errorf("array '%s' not found at path component %d", arrayName, i)
					}
					
					if arr, ok := val.([]interface{}); ok {
						array = arr
					} else {
						return nil, fmt.Errorf("'%s' is not an array at path component %d", arrayName, i)
					}
				} else {
					return nil, fmt.Errorf("expected object at path component %d", i)
				}
			}
			
			// Parse the index
			var index int
			if indexStr == "*" {
				// Return the entire array if using wildcard
				current = array
				continue
			} else {
				var err error
				index, err = parseArrayIndex(indexStr, len(array))
				if err != nil {
					return nil, fmt.Errorf("invalid array index at path component %d: %w", i, err)
				}
			}
			
			// Check index bounds
			if index < 0 || index >= len(array) {
				return nil, fmt.Errorf("array index %d out of bounds (array length: %d) at path component %d", 
					index, len(array), i)
			}
			
			// Update current to the array element
			current = array[index]
		} else {
			// Regular object property access
			if obj, ok := current.(map[string]interface{}); ok {
				val, exists := obj[component]
				if !exists {
					return nil, fmt.Errorf("property '%s' not found at path component %d", component, i)
				}
				current = val
			} else {
				return nil, fmt.Errorf("expected object at path component %d", i)
			}
		}
	}
	
	return current, nil
}

// parseArrayIndex parses array index expressions, supporting negative indices
func parseArrayIndex(indexStr string, arrayLength int) (int, error) {
	index, err := parseIntOrExpr(indexStr)
	if err != nil {
		return 0, err
	}
	
	// Handle negative indices (count from end)
	if index < 0 {
		index = arrayLength + index
	}
	
	return index, nil
}

// parseIntOrExpr parses an integer or expression
func parseIntOrExpr(s string) (int, error) {
	// Simple for now, just parse as integer
	// TODO: Add support for expressions like (length-1)
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	if err != nil {
		return 0, fmt.Errorf("invalid integer format: %s", s)
	}
	return result, nil
}

// init registers JSON extractor handler
func init() {
	// Register the handler with the extractor registry
	MustRegisterExtractor("extract_from_json", extractFromJSONHandler)
}
