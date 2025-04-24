// Package actions implements various action handlers for the VPR engine.
// This file specifically contains the implementation of the HTTP request action,
// which is one of the core actions defined in the DSL specification.
package actions

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
	
	"vpr/pkg/context"
	"vpr/pkg/poc"
	"vpr/pkg/utils"
)

// HTTPResponse represents the result of an HTTP request
type HTTPResponse struct {
	StatusCode    int                 `json:"status_code"`
	Headers       map[string][]string `json:"headers"`
	Body          string              `json:"body"`
	ResponseTime  float64             `json:"response_time_ms"`
	ContentLength int64               `json:"content_length"`
	URL           string              `json:"url"`
}

// httpRequestHandler executes an HTTP request
func httpRequestHandler(ctx *context.ExecutionContext, action *poc.Action) (interface{}, error) {
	if action.Type != "http_request" {
		return nil, fmt.Errorf("invalid action type for httpRequestHandler: %s", action.Type)
	}
	
	// 处理认证上下文
	if action.AuthenticationContext != "" {
		// 如果指定了认证上下文，则在此请求中使用该上下文
		if err := ctx.SetAuthenticationContext(action.AuthenticationContext); err != nil {
			return nil, fmt.Errorf("failed to set authentication context: %w", err)
		}
	}
	
	// Validate required parameters
	if action.Request == nil {
		return nil, fmt.Errorf("http_request action requires a 'request' section")
	}
	
	if action.Request.URL == "" {
		return nil, fmt.Errorf("http_request action requires a URL")
	}
	
	// Get HTTP client
	client, err := utils.GetHTTPClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get HTTP client: %w", err)
	}
	
	// Build request
	req, err := buildHTTPRequest(ctx, action)
	if err != nil {
		return nil, fmt.Errorf("failed to build HTTP request: %w", err)
	}
	
	// Log the request
	logHTTPRequest(req, action)
	
	// Execute the request
	startTime := time.Now()
	resp, err := client.Do(req)
	elapsedMs := float64(time.Since(startTime).Microseconds()) / 1000.0
	
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()
	
	// Process the response
	return processHTTPResponse(resp, elapsedMs)
}

// buildHTTPRequest constructs an HTTP request from action parameters
func buildHTTPRequest(ctx *context.ExecutionContext, action *poc.Action) (*http.Request, error) {
	// Resolve the URL - substitute variables if needed
	resolvedURL, err := ctx.Substitute(action.Request.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve URL: %w", err)
	}
	
	// Determine the method (default to GET)
	method := "GET"
	if action.Request.Method != "" {
		method = strings.ToUpper(action.Request.Method)
	}
	
	// Prepare the request body if applicable
	var body io.Reader
	if action.Request.Body != nil {
		// Convert body to string depending on type
		var bodyStr string
		
		switch v := action.Request.Body.(type) {
		case string:
			bodyStr = v
		case map[string]interface{}, []interface{}:
			// Handle complex body types based on body_type here
			// For now a simple string representation
			bodyStr = fmt.Sprintf("%v", v)
		default:
			bodyStr = fmt.Sprintf("%v", v)
		}
		
		// Resolve any variables in body
		resolvedBody, err := ctx.Substitute(bodyStr)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve request body: %w", err)
		}
		body = bytes.NewBufferString(resolvedBody)
	}
	
	// Create the request
	req, err := http.NewRequest(method, resolvedURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	
	// Add headers if present
	if action.Request.Headers != nil {
		for key, value := range action.Request.Headers {
			// Resolve any variables in header values
			resolvedValue, err := ctx.Substitute(value)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve header '%s': %w", key, err)
			}
			req.Header.Set(key, resolvedValue)
		}
	}
	
	// Apply authentication if an authentication context is specified
	if action.AuthenticationContext != "" {
		if err := ctx.ApplyAuthentication(req, action.AuthenticationContext); err != nil {
			return nil, fmt.Errorf("failed to apply authentication for context '%s': %w", 
				action.AuthenticationContext, err)
		}
	} else {
		// Apply the current active authentication context (if any)
		if err := ctx.ApplyAuthentication(req, ""); err != nil {
			return nil, fmt.Errorf("failed to apply active authentication context: %w", err)
		}
	}
	
	// Set default headers if not already set
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "VPR-PoCRunner/1.0")
	}
	
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "*/*")
	}
	
	// Handle special body types
	if action.Request.BodyType != "" {
		switch strings.ToLower(action.Request.BodyType) {
		case "json":
			if req.Header.Get("Content-Type") == "" {
				req.Header.Set("Content-Type", "application/json")
			}
		case "form":
			if req.Header.Get("Content-Type") == "" {
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			}
		case "multipart":
			// The multipart form data handling would go here
			// This would need to set the appropriate content type with boundary
			// and prepare the multipart writer
		}
	}
	
	return req, nil
}

// processHTTPResponse converts an HTTP response to our internal representation
func processHTTPResponse(resp *http.Response, elapsedMs float64) (*HTTPResponse, error) {
	// Read the body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	
	// Create the response object
	httpResp := &HTTPResponse{
		StatusCode:    resp.StatusCode,
		Headers:       resp.Header,
		Body:          string(bodyBytes),
		ResponseTime:  elapsedMs,
		ContentLength: resp.ContentLength,
		URL:           resp.Request.URL.String(),
	}
	
	// Log the response
	logHTTPResponse(httpResp)
	
	return httpResp, nil
}

// logHTTPRequest logs an HTTP request at debug level
func logHTTPRequest(req *http.Request, action *poc.Action) {
	slog.Debug("HTTP Request",
		"method", req.Method,
		"url", req.URL.String(),
		"headers", fmt.Sprintf("%v", req.Header),
		"has_body", action.Request.Body != nil,
	)
}

// logHTTPResponse logs an HTTP response at debug level
func logHTTPResponse(resp *HTTPResponse) {
	// Truncate body for logging
	bodyPreview := resp.Body
	if len(bodyPreview) > 200 {
		bodyPreview = bodyPreview[:200] + "..."
	}
	
	slog.Debug("HTTP Response",
		"status", resp.StatusCode,
		"content_length", resp.ContentLength,
		"response_time_ms", resp.ResponseTime,
		"body_preview", bodyPreview,
	)
}

// init registers the HTTP request handler
func init() {
	// Register the handler with the action registry
	MustRegisterAction("http_request", httpRequestHandler)
}
