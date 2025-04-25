// Package actions defines the interface and registry for runnable actions.
// This file implements the specific action handler for generating dynamic data
// (`type: generate_data`), such as random strings, timestamps, or using
// defined generators, and storing it in a context variable.
package actions

import (
	"log/slog"
	"strings"
	
	"vpr/pkg/context"
	"vpr/pkg/poc"
)

// GenerateDataResult represents the result of a data generation action
type GenerateDataResult struct {
	Generator     string      `json:"generator"`     // The generator type used
	TargetVariable string     `json:"target_variable"` // The variable where data was stored
	Value         interface{} `json:"value"`         // The generated value
	Parameters    map[string]interface{} `json:"parameters,omitempty"` // Parameters used for generation
}

// Supported generators
const (
	GeneratorUUID       = "uuid"
	GeneratorRandomString = "random_string"
	GeneratorRandomInt  = "random_int"
	GeneratorTimestamp  = "timestamp"
	GeneratorPattern    = "pattern"
	GeneratorSequence   = "sequence"
)

// GenerateDataHandler implements the generate_data action type as defined in DSL specification.
// It generates various types of dynamic data and stores it in a context variable.
// This is a wrapper around the core implementation in utilities.go
func GenerateDataHandler(ctx *context.ExecutionContext, action *poc.Action) (interface{}, error) {
	// Validation
	if action.Type != "generate_data" {
		return nil, errInvalidActionType(action.Type, "generate_data")
	}
	
	// Log detailed information about the data generation request
	slog.Info("Executing generate_data action", 
		"generator", action.Generator,
		"target_variable", action.TargetVariable)
	
	// Call the core implementation from utilities.go
	result, err := generateDataHandler(ctx, action)
	
	if err != nil {
		slog.Error("Data generation failed", 
			"generator", action.Generator,
			"error", err)
		return nil, err
	}
	
	// Provide more detailed logging based on the type of data generated
	dataType := "unknown"
	if action.Parameters != nil {
		if typeVal, ok := action.Parameters["type"].(string); ok {
			dataType = strings.ToLower(typeVal)
		}
	}
	
	slog.Info("Data generation successful", 
		"generator", action.Generator,
		"target_variable", action.TargetVariable,
		"data_type", dataType)
	
	return result, nil
}

func init() {
	// Register the generate_data action handler
	MustRegisterAction("generate_data", GenerateDataHandler)
}
