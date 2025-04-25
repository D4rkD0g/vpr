// Package actions defines the interface and registry for runnable actions.
// This file implements the specific action handler for making HTTP requests
// (`type: http_request`) as defined in the PoC DSL.
package actions

import (
	"fmt"
	"log/slog"
	
	"vpr/pkg/context"
	"vpr/pkg/poc"
)

// 用于创建标准化的动作类型错误
func errInvalidActionType(got, expected string) error {
	return fmt.Errorf("invalid action type: expected '%s', got '%s'", expected, got)
}

// HTTPRequestHandler implements the http_request action type as defined in DSL specification.
// It handles making HTTP requests with various options including method, URL, headers, body,
// authentication, and supports processing response actions.
func HTTPRequestHandler(ctx *context.ExecutionContext, action *poc.Action) (interface{}, error) {
	// Validation
	if action.Type != "http_request" {
		return nil, errInvalidActionType(action.Type, "http_request")
	}
	
	// Log the request attempt
	slog.Debug("Executing HTTP request", 
		"method", action.Request.Method,
		"url", action.Request.URL,
		"auth_context", action.AuthenticationContext)
	
	// The core implementation is in the httpRequestHandler function in http.go
	result, err := httpRequestHandler(ctx, action)
	
	if err != nil {
		slog.Error("HTTP request failed", 
			"error", err)
		return nil, err
	}
	
	// Log success with status code if available
	if httpResponse, ok := result.(*HTTPResponse); ok {
		slog.Info("HTTP request successful", 
			"status_code", httpResponse.StatusCode,
			"url", httpResponse.URL,
			"response_time_ms", httpResponse.ResponseTime)
	} else {
		slog.Info("HTTP request successful")
	}
	
	return result, nil
}

func init() {
	// Register the http_request action handler
	MustRegisterAction("http_request", HTTPRequestHandler)
}
