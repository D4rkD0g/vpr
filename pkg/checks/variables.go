// Package checks provides custom condition checking for PoC assertions
package checks

import (
	"fmt"
	"log"
	"reflect"
	"regexp"
	"strings"
	
	"vpr/pkg/context"
	"vpr/pkg/poc"
)

// variableEqualsCheck checks if a variable equals a specific value
func variableEqualsCheck(ctx *context.ExecutionContext, check *poc.Check) (bool, error) {
	// Validate required fields
	if check.Path == "" {
		return false, fmt.Errorf("variable_equals check requires path field")
	}
	
	if check.Equals == nil {
		return false, fmt.Errorf("variable_equals check requires equals field")
	}
	
	// Extract expected value as interface{}
	expectedValue := check.Equals
	
	// Convert to string if it's a string type
	expectedValueStr, isString := expectedValue.(string)
	
	// If expected is a string, resolve any variables
	if isString {
		resolvedStr, err := ctx.Substitute(expectedValueStr)
		if err != nil {
			return false, fmt.Errorf("failed to resolve expected value: %w", err)
		}
		expectedValue = resolvedStr
		expectedValueStr = resolvedStr
	}
	
	// Resolve the variable from context
	variableValue, err := ctx.ResolveVariable(check.Path)
	if err != nil {
		return false, fmt.Errorf("failed to resolve variable '%s': %w", check.Path, err)
	}
	
	// Add debug logging
	log.Printf("DEBUG: Variable Check - Path: '%s', Expected: '%v', Actual: '%v', Types: exp=%T act=%T", 
		check.Path, expectedValue, variableValue, expectedValue, variableValue)
	
	// Handle different types of expected values
	if isString {
		// Convert variable value to string if it's not already
		varStr := fmt.Sprintf("%v", variableValue)
		return varStr == expectedValueStr, nil
	} else {
		// For other types, use reflect.DeepEqual
		return reflect.DeepEqual(variableValue, expectedValue), nil
	}
}

// variableContainsCheck checks if a variable contains an expected value
func variableContainsCheck(ctx *context.ExecutionContext, check *poc.Check) (bool, error) {
	if check.Type != "variable_contains" {
		return false, fmt.Errorf("invalid check type for variableContainsCheck: %s", check.Type)
	}
	
	// Check requires path and contains fields
	if check.Path == "" {
		return false, fmt.Errorf("variable_contains check requires path field")
	}
	
	if check.Contains == nil {
		return false, fmt.Errorf("variable_contains check requires contains field")
	}
	
	// Resolve the variable from context
	variableValue, err := ctx.ResolveVariable(check.Path)
	if err != nil {
		return false, fmt.Errorf("failed to resolve variable at path %s: %w", check.Path, err)
	}
	
	// Handle different types for contains check
	switch expected := check.Contains.(type) {
	case string:
		// Resolve any variables in the expected string
		resolvedExpected, err := ctx.Substitute(expected)
		if err != nil {
			return false, fmt.Errorf("failed to resolve contains value: %w", err)
		}
		
		// Different semantics based on variable type
		switch v := variableValue.(type) {
		case string:
			// For strings, check substring
			return strings.Contains(v, resolvedExpected), nil
			
		case []interface{}:
			// For arrays, check if item exists in array
			for _, item := range v {
				// Convert item to string for comparison
				itemStr := fmt.Sprintf("%v", item)
				if itemStr == resolvedExpected {
					return true, nil
				}
			}
			return false, nil
			
		case map[string]interface{}:
			// For maps, check if key or value exists
			for k, val := range v {
				if k == resolvedExpected {
					return true, nil
				}
				
				// Also check values
				valStr := fmt.Sprintf("%v", val)
				if valStr == resolvedExpected {
					return true, nil
				}
			}
			return false, nil
			
		default:
			// For other types, convert to string and check
			varStr := fmt.Sprintf("%v", v)
			return strings.Contains(varStr, resolvedExpected), nil
		}
		
	case []interface{}:
		// For array of expected values, check if variable contains all of them
		switch v := variableValue.(type) {
		case string:
			// For strings, check if it contains all expected strings
			for _, item := range expected {
				itemStr, ok := item.(string)
				if !ok {
					return false, fmt.Errorf("contains array must only contain strings for string variable")
				}
				
				resolvedItemStr, err := ctx.Substitute(itemStr)
				if err != nil {
					return false, fmt.Errorf("failed to resolve contains value: %w", err)
				}
				
				if !strings.Contains(v, resolvedItemStr) {
					return false, nil
				}
			}
			return true, nil
			
		case []interface{}:
			// For arrays, check if all expected items exist in the array
			for _, expectedItem := range expected {
				found := false
				for _, varItem := range v {
					if fmt.Sprintf("%v", expectedItem) == fmt.Sprintf("%v", varItem) {
						found = true
						break
					}
				}
				if !found {
					return false, nil
				}
			}
			return true, nil
			
		case map[string]interface{}:
			// For maps, check if all expected items exist as keys or values
			for _, expectedItem := range expected {
				expectedStr := fmt.Sprintf("%v", expectedItem)
				found := false
				
				for k, val := range v {
					if k == expectedStr || fmt.Sprintf("%v", val) == expectedStr {
						found = true
						break
					}
				}
				
				if !found {
					return false, nil
				}
			}
			return true, nil
			
		default:
			// For other types, convert to string and check
			varStr := fmt.Sprintf("%v", v)
			for _, item := range expected {
				itemStr, ok := item.(string)
				if !ok {
					return false, fmt.Errorf("contains array must only contain strings for string comparison")
				}
				
				resolvedItemStr, err := ctx.Substitute(itemStr)
				if err != nil {
					return false, fmt.Errorf("failed to resolve contains value: %w", err)
				}
				
				if !strings.Contains(varStr, resolvedItemStr) {
					return false, nil
				}
			}
			return true, nil
		}
		
	default:
		// For other types, convert both to strings and check
		expectedStr := fmt.Sprintf("%v", expected)
		
		switch v := variableValue.(type) {
		case string:
			return strings.Contains(v, expectedStr), nil
		default:
			varStr := fmt.Sprintf("%v", v)
			return strings.Contains(varStr, expectedStr), nil
		}
	}
}

// variableRegexCheck checks if a variable matches a regex pattern
func variableRegexCheck(ctx *context.ExecutionContext, check *poc.Check) (bool, error) {
	if check.Type != "variable_regex" {
		return false, fmt.Errorf("invalid check type for variableRegexCheck: %s", check.Type)
	}
	
	// Check requires path and regex fields
	if check.Path == "" {
		return false, fmt.Errorf("variable_regex check requires path field")
	}
	
	if check.Regex == "" {
		return false, fmt.Errorf("variable_regex check requires regex field")
	}
	
	// Resolve the variable from context
	variableValue, err := ctx.ResolveVariable(check.Path)
	if err != nil {
		return false, fmt.Errorf("failed to resolve variable at path %s: %w", check.Path, err)
	}
	
	// Resolve any variables in the regex pattern
	resolvedRegex, err := ctx.Substitute(check.Regex)
	if err != nil {
		return false, fmt.Errorf("failed to resolve regex pattern: %w", err)
	}
	
	// Compile the regex
	re, err := regexp.Compile(resolvedRegex)
	if err != nil {
		return false, fmt.Errorf("invalid regex pattern %s: %w", resolvedRegex, err)
	}
	
	// Convert variable to string if needed
	var varStr string
	switch v := variableValue.(type) {
	case string:
		varStr = v
	default:
		varStr = fmt.Sprintf("%v", v)
	}
	
	// Check if variable matches regex
	return re.MatchString(varStr), nil
}

// init registers all variable-related checks
func init() {
	// Register the variable check handlers
	MustRegisterCheck("variable_equals", variableEqualsCheck)
	MustRegisterCheck("variable_contains", variableContainsCheck)
	MustRegisterCheck("variable_regex", variableRegexCheck)
}
