// Package actions implements action handlers for the VPR engine.
// This file implements the check_target_availability action for verifying
// target system availability before proceeding with PoC execution.
package actions

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
	
	execContext "vpr/pkg/context"
	"vpr/pkg/poc"
)

// availabilityHandler implements the check_target_availability action
func availabilityHandler(ctx *execContext.ExecutionContext, action *poc.Action) (interface{}, error) {
	if action.Type != "check_target_availability" {
		return nil, fmt.Errorf("invalid action type for availabilityHandler: %s", action.Type)
	}
	
	// Resolve target URL
	targetURL, err := ctx.Substitute(action.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve target URL: %w", err)
	}
	
	// If URL not provided, check parameters
	if targetURL == "" {
		// Check if URL is in parameters
		if action.Parameters != nil {
			if urlParam, ok := action.Parameters["url"]; ok {
				if urlStr, ok := urlParam.(string); ok {
					targetURL, err = ctx.Substitute(urlStr)
					if err != nil {
						return nil, fmt.Errorf("failed to resolve target URL from parameters: %w", err)
					}
				}
			}
		}
		
		// Still no URL, error
		if targetURL == "" {
			return nil, fmt.Errorf("check_target_availability requires 'url' field")
		}
	}
	
	// Parse timeout
	timeout := 5 * time.Second // Default timeout: 5 seconds
	if action.Timeout != "" {
		parsedTimeout, err := time.ParseDuration(action.Timeout)
		if err != nil {
			return nil, fmt.Errorf("invalid timeout format: %w", err)
		}
		timeout = parsedTimeout
	} else if action.Parameters != nil {
		if timeoutParam, ok := action.Parameters["timeout"]; ok {
			if timeoutStr, ok := timeoutParam.(string); ok {
				parsedTimeout, err := time.ParseDuration(timeoutStr)
				if err != nil {
					return nil, fmt.Errorf("invalid timeout format in parameters: %w", err)
				}
				timeout = parsedTimeout
			}
		}
	}
	
	// Determine check type based on URL or protocol
	checkType := "http"
	if action.Parameters != nil {
		if typeParam, ok := action.Parameters["check_type"]; ok {
			if typeStr, ok := typeParam.(string); ok {
				checkType = strings.ToLower(typeStr)
			}
		}
	}
	
	// If URL doesn't have a scheme, try to add one based on check type
	if !strings.Contains(targetURL, "://") {
		switch checkType {
		case "http", "web":
			targetURL = "http://" + targetURL
		case "https":
			targetURL = "https://" + targetURL
		case "tcp", "socket":
			// No scheme needed
		}
	}
	
	// Parse URL to determine scheme and host:port
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("invalid target URL: %w", err)
	}
	
	// Create result map
	result := map[string]interface{}{
		"available": false,
		"target":    targetURL,
		"type":      checkType,
		"timeout":   timeout.String(),
	}
	
	// Perform availability check based on type
	switch checkType {
	case "http", "https", "web":
		available, httpErr := checkHTTPAvailability(parsedURL.String(), timeout)
		result["available"] = available
		if httpErr != nil {
			result["error"] = httpErr.Error()
		}
		
	case "tcp", "socket":
		hostPort := parsedURL.Host
		if parsedURL.Port() == "" {
			// If no port is specified, try to guess it
			if parsedURL.Scheme == "https" {
				hostPort = fmt.Sprintf("%s:443", parsedURL.Host)
			} else if parsedURL.Scheme == "http" {
				hostPort = fmt.Sprintf("%s:80", parsedURL.Host)
			} else {
				// No port and can't guess, error
				return nil, fmt.Errorf("TCP check requires host:port format")
			}
		}
		
		available, tcpErr := checkTCPAvailability(hostPort, timeout)
		result["available"] = available
		if tcpErr != nil {
			result["error"] = tcpErr.Error()
		}
		
	default:
		return nil, fmt.Errorf("unsupported check type: %s", checkType)
	}
	
	// If target variable specified, store result
	if action.TargetVariable != "" {
		err = ctx.SetVariable(action.TargetVariable, result)
		if err != nil {
			return nil, fmt.Errorf("failed to set target variable: %w", err)
		}
	}
	
	// If not available and no retries, return error
	if !result["available"].(bool) {
		// Allow retries if specified
		if action.Retries > 0 {
			// Retries will be handled by executor
			return result, fmt.Errorf("target unavailable, retry scheduled")
		}
		return result, fmt.Errorf("target unavailable: %s", targetURL)
	}
	
	return result, nil
}

// checkHTTPAvailability verifies if an HTTP/HTTPS endpoint is available
func checkHTTPAvailability(targetURL string, timeout time.Duration) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	
	req, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}
	
	// Create client with timeout
	client := &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Don't follow redirects for availability check
			return http.ErrUseLastResponse
		},
	}
	
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	
	// Any response (including errors) means the server is reachable
	return true, nil
}

// checkTCPAvailability verifies if a TCP socket is available
func checkTCPAvailability(hostPort string, timeout time.Duration) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", hostPort)
	if err != nil {
		return false, err
	}
	conn.Close()
	
	return true, nil
}

// init registers the action handler
func init() {
	MustRegisterAction("check_target_availability", availabilityHandler)
}
