// Package poc defines the Go data structures that represent the v1.0 PoC DSL format.
// This file specifically handles loading PoC definitions from YAML/JSON files,
// parsing them into the defined Go structs, and performing basic validation.
package poc

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const ExpectedDslVersion = "1.0"

// LoadPocFromFile reads a PoC definition from a given YAML file path,
// unmarshals it into the Poc struct, and validates the DSL version.
// It handles both direct Poc objects and PocWrapper objects (with top-level 'poc:' key).
func LoadPocFromFile(filePath string) (*Poc, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read PoC file '%s': %w", filePath, err)
	}

	// Try parsing as a wrapper first (has 'poc:' top-level key)
	var pocWrapper PocWrapper
	err = yaml.Unmarshal(data, &pocWrapper)
	
	// If successful and has content, use the wrapped Poc
	if err == nil && pocWrapper.Poc.Metadata.ID != "" {
		// Basic Validation: Check DSL Version
		if pocWrapper.Poc.Metadata.DslVersion != ExpectedDslVersion {
			return nil, fmt.Errorf("invalid DSL version in '%s': expected '%s', got '%s'", 
				filePath, ExpectedDslVersion, pocWrapper.Poc.Metadata.DslVersion)
		}
		return &pocWrapper.Poc, nil
	}

	// Otherwise try direct parse (no 'poc:' wrapper)
	var poc Poc
	err = yaml.Unmarshal(data, &poc)
	if err != nil {
		// If both parse attempts failed, provide better error information
		var node yaml.Node
		if parseErr := yaml.Unmarshal(data, &node); parseErr == nil {
			// Try to provide more information about where parsing failed
			return nil, enhanceYamlError(filePath, data, err)
		}
		return nil, fmt.Errorf("failed to unmarshal YAML from '%s': %w", filePath, err)
	}

	// Basic Validation: Check DSL Version
	if poc.Metadata.DslVersion != ExpectedDslVersion {
		return nil, fmt.Errorf("invalid DSL version in '%s': expected '%s', got '%s'", 
			filePath, ExpectedDslVersion, poc.Metadata.DslVersion)
	}

	// Perform basic structural validation
	if err := ValidatePoc(&poc); err != nil {
		return nil, fmt.Errorf("validation failed for '%s': %w", filePath, err)
	}

	return &poc, nil
}

// enhanceYamlError attempts to provide more detailed error information for YAML parse errors
func enhanceYamlError(filePath string, data []byte, originalErr error) error {
	errMsg := originalErr.Error()
	
	// Extract line number if present in error message
	lineNumStart := strings.Index(errMsg, "line ")
	if lineNumStart != -1 {
		// Try to extract line number and show the problematic line
		lineNumEnd := strings.Index(errMsg[lineNumStart:], ":")
		if lineNumEnd != -1 {
			lineNumStr := errMsg[lineNumStart+5 : lineNumStart+lineNumEnd]
			_ = lineNumStr // Placeholder for future line extraction
			// Future enhancement: Show the problematic line from the file
		}
	}
	
	// Return a more descriptive error with file name
	fileName := filepath.Base(filePath)
	return fmt.Errorf("YAML parsing error in '%s': %w", fileName, originalErr)
}

// ValidatePoc performs comprehensive validation of a Poc struct against the DSL schema rules
func ValidatePoc(poc *Poc) error {
	if poc == nil {
		return fmt.Errorf("nil Poc cannot be validated")
	}
	
	// Basic structural validation
	if poc.Metadata.ID == "" {
		return fmt.Errorf("Poc metadata.id is required")
	}
	
	if poc.Metadata.Title == "" {
		return fmt.Errorf("Poc metadata.title is required") 
	}
	
	if poc.Metadata.DslVersion == "" {
		return fmt.Errorf("Poc metadata.dsl_version is required")
	}
	
	// Context validation
	if poc.Context.Users != nil && len(poc.Context.Users) > 0 {
		for i, user := range poc.Context.Users {
			if user.ID == "" {
				return fmt.Errorf("context.users[%d].id is required", i)
			}
		}
	}
	
	if poc.Context.Resources != nil && len(poc.Context.Resources) > 0 {
		for i, resource := range poc.Context.Resources {
			if resource.ID == "" {
				return fmt.Errorf("context.resources[%d].id is required", i)
			}
		}
	}
	
	if poc.Context.Environment != nil && len(poc.Context.Environment) > 0 {
		for i, env := range poc.Context.Environment {
			if env.ID == "" {
				return fmt.Errorf("context.environment[%d].id is required", i)
			}
		}
	}
	
	if poc.Context.Files != nil && len(poc.Context.Files) > 0 {
		for i, file := range poc.Context.Files {
			if file.ID == "" {
				return fmt.Errorf("context.files[%d].id is required", i)
			}
			if file.LocalPath == "" {
				return fmt.Errorf("context.files[%d].local_path is required", i)
			}
		}
	}
	
	if poc.Context.Variables != nil && len(poc.Context.Variables) > 0 {
		for i, variable := range poc.Context.Variables {
			if variable.ID == "" {
				return fmt.Errorf("context.variables[%d].id is required", i)
			}
		}
	}
	
	// Exploit scenario validation
	if len(poc.Exploit.Steps) == 0 {
		return fmt.Errorf("Poc exploit_scenario.steps must contain at least one step")
	}
	
	// Assertions validation
	if len(poc.Assertions) == 0 {
		return fmt.Errorf("Poc assertions must contain at least one step")
	}
	
	// Validate all steps
	allSteps := [][]Step{
		poc.Setup,
		poc.Exploit.Steps,
		poc.Exploit.Setup,
		poc.Exploit.Teardown,
		poc.Assertions,
		poc.Verification,
	}
	
	for _, steps := range allSteps {
		if steps == nil {
			continue
		}
		
		for i, step := range steps {
			if step.DSL == "" {
				return fmt.Errorf("step[%d].dsl is required", i)
			}
			
			// Validate that at least one of Action or Check is present (but not both)
			if step.Action == nil && step.Check == nil && step.Loop == nil {
				return fmt.Errorf("step[%d] must have either an action, check, or loop", i)
			}
			
			// If step has an Action, validate it
			if step.Action != nil {
				if step.Action.Type == "" {
					return fmt.Errorf("step[%d].action.type is required", i)
				}
				
				// HTTP Request validation
				if step.Action.Type == "http_request" && step.Action.Request != nil {
					if step.Action.Request.Method == "" {
						return fmt.Errorf("step[%d].action.request.method is required", i)
					}
					if step.Action.Request.URL == "" {
						return fmt.Errorf("step[%d].action.request.url is required", i)
					}
				}
				
				// For response actions, validate target variables
				if step.Action.ResponseActions != nil && len(step.Action.ResponseActions) > 0 {
					for j, ra := range step.Action.ResponseActions {
						if ra.Type == "" {
							return fmt.Errorf("step[%d].action.response_actions[%d].type is required", i, j)
						}
						if ra.TargetVariable == "" {
							return fmt.Errorf("step[%d].action.response_actions[%d].target_variable is required", i, j)
						}
					}
				}
			}
			
			// If step has a Check, validate it
			if step.Check != nil {
				if step.Check.Type == "" {
					return fmt.Errorf("step[%d].check.type is required", i)
				}
				
				// Additional validation based on check type
				switch step.Check.Type {
				case "http_response_status":
					if step.Check.ExpectedStatus == nil {
						return fmt.Errorf("step[%d].check.expected_status is required for http_response_status check", i)
					}
				case "http_response_body":
					// At least one of contains, equals, regex, json_path should be present
					if step.Check.Contains == nil && step.Check.Equals == nil && 
					   step.Check.EqualsJSON == nil && step.Check.Regex == "" && step.Check.JSONPath == nil {
						return fmt.Errorf("step[%d].check requires at least one assertion (contains, equals, equals_json, regex, json_path)", i)
					}
				case "json_path":
					if step.Check.JSONPath == nil {
						return fmt.Errorf("step[%d].check.json_path is required for json_path check", i)
					}
					if step.Check.JSONPath.Path == "" {
						return fmt.Errorf("step[%d].check.json_path.path is required", i)
					}
				}
			}
			
			// If step has a Loop, validate it
			if step.Loop != nil {
				if step.Loop.Over == "" {
					return fmt.Errorf("step[%d].loop.over is required", i)
				}
				if step.Loop.VariableName == "" {
					return fmt.Errorf("step[%d].loop.variable_name is required", i)
				}
				if step.Loop.Steps == nil || len(step.Loop.Steps) == 0 {
					return fmt.Errorf("step[%d].loop.steps must contain at least one step", i)
				}
				
				// Recursively validate steps within the loop
				for j, loopStep := range step.Loop.Steps {
					if loopStep.DSL == "" {
						return fmt.Errorf("step[%d].loop.steps[%d].dsl is required", i, j)
					}
				}
			}
		}
	}
	
	return nil
}

// SavePocToFile serializes a Poc struct to YAML and saves it to a file
func SavePocToFile(poc *Poc, filePath string, asWrapper bool) error {
	var data []byte
	var err error
	
	if asWrapper {
		wrapper := PocWrapper{
			Poc: *poc,
		}
		data, err = yaml.Marshal(wrapper)
	} else {
		data, err = yaml.Marshal(poc)
	}
	
	if err != nil {
		return fmt.Errorf("failed to marshal Poc to YAML: %w", err)
	}
	
	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write to file '%s': %w", filePath, err)
	}
	
	return nil
}
