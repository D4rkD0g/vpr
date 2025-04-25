// Package checks implements check handlers for the VPR engine.
// This file implements the json_schema_validation check for validating
// JSON responses against JSON Schema definitions.
package checks

import (
	"encoding/json"
	"fmt"
	"strings"
	"regexp"
	
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
		lastResult, err := ctx.ResolveVariable("last_result")
		if err != nil {
			// Try to get from last response if available
			if lastResponse := ctx.GetLastResponse(); lastResponse != nil && len(lastResponse) > 0 {
				jsonData = lastResponse
			} else {
				return false, fmt.Errorf("no source specified and couldn't get last_result or last_response: %w", err)
			}
		} else {
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
	}
	
	// Parse the JSON data and schema
	var dataObj interface{}
	if err := json.Unmarshal(jsonData, &dataObj); err != nil {
		return false, fmt.Errorf("invalid input JSON: %w", err)
	}
	
	var schemaObj interface{}
	if err := json.Unmarshal([]byte(resolvedSchema), &schemaObj); err != nil {
		return false, fmt.Errorf("invalid schema JSON: %w", err)
	}
	
	// Store validation details in context for later reference
	validationDetails := map[string]interface{}{
		"valid": false,
		"schema": resolvedSchema,
		"data": string(jsonData),
	}
	
	// Validate using enhanced deep equals with schema validation rules
	valid, validationErrors := validateJSONSchema(dataObj, schemaObj)
	validationDetails["valid"] = valid
	
	// If invalid, format detailed error message
	if !valid {
		validationDetails["errors"] = validationErrors
		
		// Format a detailed error message
		errorMessage := fmt.Sprintf("JSON Schema validation failed: %s", strings.Join(validationErrors, "; "))
		
		// Store error in context for potential expected_error checks
		ctx.SetLastError(fmt.Errorf("%s", errorMessage))
		
		// Allow storing validation result in a target variable if specified
		outputVar := getOutputVariableName(check)
		if outputVar != "" {
			_ = ctx.SetVariable(outputVar, validationDetails)
		}
		
		// Return validation failure
		return false, fmt.Errorf("%s", errorMessage)
	}
	
	// For successful validation, we can also store the result
	outputVar := getOutputVariableName(check)
	if outputVar != "" {
		_ = ctx.SetVariable(outputVar, validationDetails)
	}
	
	// Validation passed
	return true, nil
}

// getOutputVariableName extracts a variable name for storing results
// from path or from implementation-specific conventions
func getOutputVariableName(check *poc.Check) string {
	// First try to use path with "_result" suffix if present
	if check.Path != "" {
		return check.Path + "_result"
	}
	
	// Otherwise use a default name
	return "json_schema_validation_result"
}

// validateJSONSchema performs JSON Schema validation with basic support
// for type validation, required properties, and pattern matching
func validateJSONSchema(data, schema interface{}) (bool, []string) {
	var errors []string
	
	// Handle different schema types
	switch schemaObj := schema.(type) {
	case map[string]interface{}:
		// This is a JSON Schema object
		return validateObject(data, schemaObj)
		
	default:
		// For simple schemas, just do deep comparison
		if !deepEquals(data, schema) {
			errors = append(errors, fmt.Sprintf("data does not match schema pattern"))
			return false, errors
		}
	}
	
	return true, nil
}

// validateObject validates a JSON object against a schema object
func validateObject(data interface{}, schema map[string]interface{}) (bool, []string) {
	var errors []string
	
	// Check type first
	if typeConstraint, ok := schema["type"]; ok {
		if !validateType(data, typeConstraint) {
			errors = append(errors, fmt.Sprintf("invalid type: expected %v", typeConstraint))
			return false, errors
		}
	}
	
	// Handle different validation based on data type
	switch dataObj := data.(type) {
	case map[string]interface{}:
		// Validate required properties
		if required, ok := schema["required"].([]interface{}); ok {
			for _, req := range required {
				if propName, ok := req.(string); ok {
					if _, exists := dataObj[propName]; !exists {
						errors = append(errors, fmt.Sprintf("missing required property: %s", propName))
					}
				}
			}
		}
		
		// Validate properties
		if properties, ok := schema["properties"].(map[string]interface{}); ok {
			for propName, propSchema := range properties {
				if propValue, exists := dataObj[propName]; exists {
					if propSchemaObj, ok := propSchema.(map[string]interface{}); ok {
						valid, propErrors := validateObject(propValue, propSchemaObj)
						if !valid {
							for _, err := range propErrors {
								errors = append(errors, fmt.Sprintf("property '%s': %s", propName, err))
							}
						}
					}
				}
			}
		}
		
	case []interface{}:
		// Validate array items
		if items, ok := schema["items"].(map[string]interface{}); ok {
			for i, item := range dataObj {
				valid, itemErrors := validateObject(item, items)
				if !valid {
					for _, err := range itemErrors {
						errors = append(errors, fmt.Sprintf("item[%d]: %s", i, err))
					}
				}
			}
		}
		
		// Validate min/max items
		if minItems, ok := schema["minItems"].(float64); ok {
			if float64(len(dataObj)) < minItems {
				errors = append(errors, fmt.Sprintf("array length %d is less than minItems %v", len(dataObj), minItems))
			}
		}
		
		if maxItems, ok := schema["maxItems"].(float64); ok {
			if float64(len(dataObj)) > maxItems {
				errors = append(errors, fmt.Sprintf("array length %d is greater than maxItems %v", len(dataObj), maxItems))
			}
		}
		
	case string:
		// Validate string patterns
		if pattern, ok := schema["pattern"].(string); ok {
			matched, err := regexp.MatchString(pattern, dataObj)
			if err != nil || !matched {
				errors = append(errors, fmt.Sprintf("string does not match pattern: %s", pattern))
			}
		}
		
		// Validate min/max length
		if minLength, ok := schema["minLength"].(float64); ok {
			if float64(len(dataObj)) < minLength {
				errors = append(errors, fmt.Sprintf("string length %d is less than minLength %v", len(dataObj), minLength))
			}
		}
		
		if maxLength, ok := schema["maxLength"].(float64); ok {
			if float64(len(dataObj)) > maxLength {
				errors = append(errors, fmt.Sprintf("string length %d is greater than maxLength %v", len(dataObj), maxLength))
			}
		}
		
	case float64:
		// Validate numeric constraints
		if minimum, ok := schema["minimum"].(float64); ok {
			if dataObj < minimum {
				errors = append(errors, fmt.Sprintf("value %v is less than minimum %v", dataObj, minimum))
			}
		}
		
		if maximum, ok := schema["maximum"].(float64); ok {
			if dataObj > maximum {
				errors = append(errors, fmt.Sprintf("value %v is greater than maximum %v", dataObj, maximum))
			}
		}
	}
	
	if len(errors) > 0 {
		return false, errors
	}
	
	return true, nil
}

// validateType checks if a value matches the expected JSON Schema type
func validateType(value interface{}, typeConstraint interface{}) bool {
	typeStr, ok := typeConstraint.(string)
	if !ok {
		return false
	}
	
	switch typeStr {
	case "object":
		_, ok := value.(map[string]interface{})
		return ok
	case "array":
		_, ok := value.([]interface{})
		return ok
	case "string":
		_, ok := value.(string)
		return ok
	case "number":
		_, ok := value.(float64)
		return ok
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "null":
		return value == nil
	}
	
	return false
}

// deepEquals performs a deep equality check between two JSON objects
// This is used as a fallback for simple structure comparison when 
// the schema library is not available
func deepEquals(a, b interface{}) bool {
	// Handle nil cases
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	
	// Compare based on type
	switch valA := a.(type) {
	case map[string]interface{}:
		// If a is a map, b must also be a map with the same entries
		mapB, ok := b.(map[string]interface{})
		if !ok {
			return false
		}
		
		// Check if maps have the same number of keys
		if len(valA) != len(mapB) {
			return false
		}
		
		// Check each key/value pair
		for k, v := range valA {
			valueB, ok := mapB[k]
			if !ok {
				return false // Key doesn't exist in b
			}
			
			if !deepEquals(v, valueB) {
				return false // Values are not equal
			}
		}
		return true
		
	case []interface{}:
		// If a is an array, b must also be an array with the same elements
		arrB, ok := b.([]interface{})
		if !ok {
			return false
		}
		
		// Check if arrays have the same length
		if len(valA) != len(arrB) {
			return false
		}
		
		// Check each element
		for i, v := range valA {
			if !deepEquals(v, arrB[i]) {
				return false
			}
		}
		return true
		
	default:
		// For primitive types, use direct equality
		return a == b
	}
}

// init registers the json_schema_validation check handler
func init() {
	MustRegisterCheck("json_schema_validation", jsonSchemaCheck)
}
