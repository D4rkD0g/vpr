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
	"vpr/pkg/utils"
)

// availabilityHandler implements the check_target_availability action
func availabilityHandler(ctx *execContext.ExecutionContext, action *poc.Action) (interface{}, error) {
	if action.Type != "check_target_availability" {
		return nil, fmt.Errorf("invalid action type for availabilityHandler: %s", action.Type)
	}
	
	// Prepare result structure
	result := map[string]interface{}{
		"available": false,
		"timeout":   "5s", // Default timeout
	}
	
	// Parse target information from URL, host/port, or parameters
	targetURL, hostPort, err := resolveTargetInfo(ctx, action, result)
	if err != nil {
		return result, err
	}
	
	// Parse timeout
	timeout, err := resolveTimeout(ctx, action)
	if err != nil {
		return result, err
	}
	result["timeout"] = timeout.String()
	
	// Determine check type
	checkType, err := determineCheckType(action, targetURL)
	if err != nil {
		return result, err
	}
	result["type"] = checkType
	
	// Perform availability check based on type
	available, checkErr := performAvailabilityCheck(ctx, checkType, targetURL, hostPort, timeout)
	result["available"] = available
	if checkErr != nil {
		result["error"] = checkErr.Error()
		// Store error in context for potential expected_error checks
		ctx.SetLastError(checkErr)
	}
	
	// Store result in context variable if specified
	if targetVar := resolveTargetVariable(action); targetVar != "" {
		if err := ctx.SetVariable(targetVar, result); err != nil {
			return result, fmt.Errorf("failed to set target variable: %w", err)
		}
	}
	
	// Return result or error based on availability
	if !available {
		errorMsg := fmt.Sprintf("target unavailable: %s", targetURL)
		if hostPort != "" {
			errorMsg = fmt.Sprintf("target unavailable: %s", hostPort)
		}
		
		return result, fmt.Errorf("%s", errorMsg)
	}
	
	return result, nil
}

// resolveTargetInfo extracts target information from action
func resolveTargetInfo(ctx *execContext.ExecutionContext, action *poc.Action, result map[string]interface{}) (string, string, error) {
	var targetURL, hostPort string
	
	// Try to get URL from action.URL field
	if action.URL != "" {
		resolved, err := ctx.Substitute(action.URL)
		if err != nil {
			return "", "", fmt.Errorf("failed to resolve URL: %w", err)
		}
		targetURL = resolved
		result["target"] = targetURL
	}
	
	// Check for URL in parameters
	if targetURL == "" && action.Parameters != nil {
		if urlParam, ok := action.Parameters["url"]; ok {
			if urlStr, ok := urlParam.(string); ok {
				resolved, err := ctx.Substitute(urlStr)
				if err != nil {
					return "", "", fmt.Errorf("failed to resolve URL parameter: %w", err)
				}
				targetURL = resolved
				result["target"] = targetURL
			}
		}
	}
	
	// Check for host/port in parameters (specification compliant)
	if targetURL == "" && action.Parameters != nil {
		var host, port string
		
		if hostParam, ok := action.Parameters["host"]; ok {
			if hostStr, ok := hostParam.(string); ok {
				resolved, err := ctx.Substitute(hostStr)
				if err != nil {
					return "", "", fmt.Errorf("failed to resolve host parameter: %w", err)
				}
				host = resolved
			}
		}
		
		if portParam, ok := action.Parameters["port"]; ok {
			// Handle both string and numeric port
			switch p := portParam.(type) {
			case string:
				resolved, err := ctx.Substitute(p)
				if err != nil {
					return "", "", fmt.Errorf("failed to resolve port parameter: %w", err)
				}
				port = resolved
			case int, int64, float64:
				port = fmt.Sprintf("%v", p)
			}
		}
		
		if host != "" {
			if port != "" {
				hostPort = fmt.Sprintf("%s:%s", host, port)
				result["host"] = host
				result["port"] = port
				result["target"] = hostPort
			} else {
				// Host without port - may be for HTTP check
				targetURL = host
				result["target"] = host
			}
		}
	}
	
	// Ensure we have either URL or host:port
	if targetURL == "" && hostPort == "" {
		return "", "", fmt.Errorf("check_target_availability requires 'url' or 'host'/'port' parameters")
	}
	
	return targetURL, hostPort, nil
}

// resolveTimeout extracts timeout from action
func resolveTimeout(ctx *execContext.ExecutionContext, action *poc.Action) (time.Duration, error) {
	timeout := 5 * time.Second // Default timeout: 5 seconds
	
	// Try to get timeout from action.Timeout field
	if action.Timeout != "" {
		parsedTimeout, err := time.ParseDuration(action.Timeout)
		if err != nil {
			return timeout, fmt.Errorf("invalid timeout format: %w", err)
		}
		timeout = parsedTimeout
	} else if action.Parameters != nil {
		// Try to get timeout from parameters
		if timeoutParam, ok := action.Parameters["timeout"]; ok {
			if timeoutStr, ok := timeoutParam.(string); ok {
				parsedTimeout, err := time.ParseDuration(timeoutStr)
				if err != nil {
					return timeout, fmt.Errorf("invalid timeout format in parameters: %w", err)
				}
				timeout = parsedTimeout
			}
		}
	}
	
	return timeout, nil
}

// determineCheckType determines the type of availability check to perform
func determineCheckType(action *poc.Action, targetURL string) (string, error) {
	// Default check type
	checkType := "http"
	
	// Override with parameter if provided
	if action.Parameters != nil {
		if typeParam, ok := action.Parameters["check_type"]; ok {
			if typeStr, ok := typeParam.(string); ok {
				checkType = strings.ToLower(typeStr)
			}
		}
	}
	
	// Auto-detect from URL if possible
	if targetURL != "" {
		// If URL has http or https scheme
		if strings.HasPrefix(strings.ToLower(targetURL), "http://") {
			checkType = "http"
		} else if strings.HasPrefix(strings.ToLower(targetURL), "https://") {
			checkType = "https"
		} else if strings.Contains(targetURL, ":") {
			// Contains colon but no scheme, might be host:port
			checkType = "tcp"
		}
	}
	
	// Validate supported check types
	switch checkType {
	case "http", "https", "web", "tcp", "socket":
		return checkType, nil
	default:
		return checkType, fmt.Errorf("unsupported check type: %s", checkType)
	}
}

// resolveTargetVariable extracts the target variable name
func resolveTargetVariable(action *poc.Action) string {
	// Try action.TargetVariable field
	if action.TargetVariable != "" {
		return action.TargetVariable
	}
	
	// Try parameters
	if action.Parameters != nil {
		if targetVar, ok := action.Parameters["target_variable"]; ok {
			if targetVarStr, ok := targetVar.(string); ok {
				return targetVarStr
			}
		}
		if outputVar, ok := action.Parameters["output_variable"]; ok {
			if outputVarStr, ok := outputVar.(string); ok {
				return outputVarStr
			}
		}
	}
	
	return ""
}

// performAvailabilityCheck performs the actual availability check
func performAvailabilityCheck(ctx *execContext.ExecutionContext, checkType, targetURL, hostPort string, timeout time.Duration) (bool, error) {
	switch checkType {
	case "http", "https", "web":
		return checkHTTPAvailability(ctx, targetURL, timeout)
	case "tcp", "socket":
		// Determine hostPort from URL if not provided directly
		if hostPort == "" && targetURL != "" {
			parsedURL, err := url.Parse(ensureURLHasScheme(targetURL))
			if err != nil {
				return false, fmt.Errorf("invalid target URL: %w", err)
			}
			
			hostPort = parsedURL.Host
			// Add default port if missing
			if parsedURL.Port() == "" {
				if parsedURL.Scheme == "https" {
					hostPort = fmt.Sprintf("%s:443", parsedURL.Host)
				} else if parsedURL.Scheme == "http" {
					hostPort = fmt.Sprintf("%s:80", parsedURL.Host)
				} else {
					return false, fmt.Errorf("TCP check requires host:port format")
				}
			}
		}
		
		if hostPort == "" {
			return false, fmt.Errorf("TCP check requires host:port specification")
		}
		
		return checkTCPAvailability(hostPort, timeout)
	default:
		return false, fmt.Errorf("unsupported check type: %s", checkType)
	}
}

// ensureURLHasScheme ensures the URL has a scheme, adding http:// if missing
func ensureURLHasScheme(targetURL string) string {
	if !strings.Contains(targetURL, "://") {
		return "http://" + targetURL
	}
	return targetURL
}

// checkHTTPAvailability verifies if an HTTP/HTTPS endpoint is available
func checkHTTPAvailability(ctx *execContext.ExecutionContext, targetURL string, timeout time.Duration) (bool, error) {
	// Ensure URL has a scheme
	targetURL = ensureURLHasScheme(targetURL)
	
	// Get HTTP client from context or create with timeout
	client, err := utils.GetHTTPClient(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get HTTP client: %w", err)
	}
	
	// Create context with timeout
	reqCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	
	// Prepare request
	req, err := http.NewRequestWithContext(reqCtx, "HEAD", targetURL, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set user agent
	req.Header.Set("User-Agent", "VPR-Availability-Check/1.0")
	
	// Save original redirect policy then disable redirects for availability check
	originalRedirectPolicy := client.CheckRedirect
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	
	// Restore original redirect policy when done
	defer func() {
		client.CheckRedirect = originalRedirectPolicy
	}()
	
	// Perform the request
	resp, err := client.Do(req)
	if err != nil {
		// Check if the error is timeout or connection refused
		return false, fmt.Errorf("HTTP availability check failed: %w", err)
	}
	defer resp.Body.Close()
	
	// Any response (including error codes) means server is available
	return true, nil
}

// checkTCPAvailability verifies if a TCP socket is available
func checkTCPAvailability(hostPort string, timeout time.Duration) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", hostPort)
	if err != nil {
		return false, fmt.Errorf("TCP connection failed: %w", err)
	}
	conn.Close()
	
	return true, nil
}

// init registers the action handler
func init() {
	MustRegisterAction("check_target_availability", availabilityHandler)
}
