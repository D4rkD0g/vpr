// Package checks implements check handlers for the VPR engine.
// This file implements the json_schema_validation check for validating
// JSON responses against JSON Schema definitions.
package checks

import (
	"encoding/json"
	"fmt"
	
	execContext "vpr/pkg/context"
	"vpr/pkg/poc"
)

// jsonSchemaCheck implements the json_schema_validation check
func jsonSchemaCheck(ctx *execContext.ExecutionContext, check *poc.Check) (bool, error) {
	if check.Type != "json_schema_validation" {
		return false, fmt.Errorf("invalid check type for jsonSchemaCheck: %s", check.Type)
	}
	
	// Get schema - primarily from EqualsJSON field
	var schema string
	if check.EqualsJSON != nil {
		// If EqualsJSON is provided as an object, convert to string
		schemaBytes, err := json.Marshal(check.EqualsJSON)
		if err != nil {
			return false, fmt.Errorf("failed to marshal equals_json to schema: %w", err)
		}
		schema = string(schemaBytes)
	}
	
	if schema == "" {
		return false, fmt.Errorf("json_schema_validation requires a schema (via equals_json)")
	}
	
	// Resolve any variables in the schema
	resolvedSchema, err := ctx.Substitute(schema)
	if err != nil {
		return false, fmt.Errorf("failed to resolve schema: %w", err)
	}
	
	// Get the data to validate
	// This could come from a variable or from previous action results
	var jsonData []byte
	
	// First, check if we have a specific source variable
	if check.Path != "" {
		// Path field can be used to specify a variable containing the JSON to validate
		varValue, err := ctx.ResolveVariable(check.Path)
		if err != nil {
			return false, fmt.Errorf("failed to resolve source variable '%s': %w", check.Path, err)
		}
		
		// Convert the variable value to JSON bytes
		switch v := varValue.(type) {
		case string:
			jsonData = []byte(v)
		case []byte:
			jsonData = v
		default:
			// Try to marshal anything else to JSON
			data, err := json.Marshal(v)
			if err != nil {
				return false, fmt.Errorf("failed to marshal variable value to JSON: %w", err)
			}
			jsonData = data
		}
	} else {
		// If no specific path is provided, try to use the last action result
		// This is implementation-specific and depends on how action results are stored
		lastResult, err := ctx.ResolveVariable("last_result")
		if err != nil {
			return false, fmt.Errorf("no source specified and couldn't get last_result: %w", err)
		}
		
		// Convert the last result to JSON bytes
		switch v := lastResult.(type) {
		case string:
			jsonData = []byte(v)
		case []byte:
			jsonData = v
		default:
			// Try to marshal anything else to JSON
			data, err := json.Marshal(v)
			if err != nil {
				return false, fmt.Errorf("failed to marshal last result to JSON: %w", err)
			}
			jsonData = data
		}
	}
	
	// Use native Go JSON validation
	
	// Parse the schema
	var schemaObj interface{}
	err = json.Unmarshal([]byte(resolvedSchema), &schemaObj)
	if err != nil {
		return false, fmt.Errorf("invalid schema JSON: %w", err)
	}
	
	// Parse the data
	var dataObj interface{}
	err = json.Unmarshal(jsonData, &dataObj)
	if err != nil {
		return false, fmt.Errorf("invalid input JSON: %w", err)
	}
	
	// Simple validation based on structure equality
	// This isn't a full JSON Schema validation, just a structure comparison
	valid := false
	
	if check.EqualsJSON != nil {
		// Deep equals comparison
		valid = deepEquals(dataObj, schemaObj)
	}
	
	// Store results in context if needed
	if !valid {
		// Store error in context for potential expected_error checks
		ctx.SetLastError(fmt.Errorf("JSON validation failed"))
	}
	
	return valid, nil
}

// deepEquals performs a deep equality check between two JSON objects
func deepEquals(a, b interface{}) bool {
	// Check nil values
	if a == nil || b == nil {
		return a == b
	}
	
	switch aVal := a.(type) {
	case map[string]interface{}:
		bVal, ok := b.(map[string]interface{})
		if !ok || len(aVal) != len(bVal) {
			return false
		}
		
		for k, v := range aVal {
			if !deepEquals(v, bVal[k]) {
				return false
			}
		}
		
		return true
		
	case []interface{}:
		bVal, ok := b.([]interface{})
		if !ok || len(aVal) != len(bVal) {
			return false
		}
		
		for i, v := range aVal {
			if !deepEquals(v, bVal[i]) {
				return false
			}
		}
		
		return true
		
	case string:
		bVal, ok := b.(string)
		return ok && aVal == bVal
		
	case float64:
		bVal, ok := b.(float64)
		return ok && aVal == bVal
		
	case bool:
		bVal, ok := b.(bool)
		return ok && aVal == bVal
		
	default:
		// Unsupported type
		return false
	}
}

// init registers the json_schema_validation check handler
func init() {
	MustRegisterCheck("json_schema_validation", jsonSchemaCheck)
}
