// Package actions implements various action handlers for the VPR engine.
// This file contains utility actions such as wait and generate_data.
package actions

import (
	"fmt"
	"log"
	"log/slog"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"
	
	"vpr/pkg/context"
	"vpr/pkg/poc"
)

// waitHandler implements the wait action to pause execution for a specified duration
func waitHandler(ctx *context.ExecutionContext, action *poc.Action) (interface{}, error) {
	if action.Type != "wait" {
		return nil, fmt.Errorf("invalid action type for waitHandler: %s", action.Type)
	}
	
	// Check for timeout field
	if action.Timeout == "" {
		return nil, fmt.Errorf("wait action requires timeout field")
	}
	
	// Resolve any variables in the timeout string
	resolvedTimeout, err := ctx.Substitute(action.Timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve timeout value: %w", err)
	}
	
	// Parse the duration
	duration, err := parseDuration(resolvedTimeout)
	if err != nil {
		return nil, fmt.Errorf("invalid duration format '%s': %w", resolvedTimeout, err)
	}
	
	// Log the wait
	slog.Info("Waiting", "duration", duration.String())
	
	// Sleep for the specified duration
	time.Sleep(duration)
	
	// Return the actual duration we waited
	return map[string]interface{}{
		"duration_ms": float64(duration.Milliseconds()),
		"duration":    duration.String(),
	}, nil
}

// generateDataHandler implements the generate_data action to create random or patterned data
func generateDataHandler(ctx *context.ExecutionContext, action *poc.Action) (interface{}, error) {
	if action.Type != "generate_data" {
		return nil, fmt.Errorf("invalid action type for generateDataHandler: %s", action.Type)
	}
	
	// Require target variable to store the generated data
	if action.TargetVariable == "" {
		return nil, fmt.Errorf("generate_data action requires target_variable field")
	}
	
	// Parse action parameters
	if action.Parameters == nil {
		return nil, fmt.Errorf("generate_data action requires parameters")
	}
	params := action.Parameters
	
	// Get data type
	dataType, ok := params["type"].(string)
	if !ok {
		return nil, fmt.Errorf("generate_data requires 'type' parameter")
	}
	
	var generatedData interface{}
	var err error
	
	// Generate data based on type
	switch strings.ToLower(dataType) {
	case "string":
		generatedData, err = generateString(ctx, params)
	case "number", "integer":
		generatedData, err = generateNumber(ctx, params)
	case "boolean", "bool":
		generatedData, err = generateBoolean(ctx, params)
	case "uuid":
		generatedData, err = generateUUID(ctx, params)
	case "email":
		generatedData, err = generateEmail(ctx, params)
	case "ip":
		generatedData, err = generateIP(ctx, params)
	case "date":
		generatedData, err = generateDate(ctx, params)
	case "array":
		generatedData, err = generateArray(ctx, params)
	case "object":
		generatedData, err = generateObject(ctx, params)
	default:
		return nil, fmt.Errorf("unsupported data type: %s", dataType)
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to generate data: %w", err)
	}
	
	// Debug output for variable and value
	log.Printf("DEBUG: Generated data for variable '%s': %v (type: %T)", 
		action.TargetVariable, generatedData, generatedData)
	
	// Create a proper variable structure that matches the ContextVariable format
	varStruct := &poc.ContextVariable{
		ID: action.TargetVariable,
		Value: generatedData,
	}
	
	// Store the generated data in the variables map
	varsPath := "variables." + action.TargetVariable
	err = ctx.SetVariable(varsPath, varStruct)
	if err != nil {
		return nil, fmt.Errorf("failed to store generated data in variable: %w", err)
	}
	
	// Verify the variable was set properly
	val, err := ctx.ResolveVariable("variables." + action.TargetVariable)
	if err != nil {
		log.Printf("WARNING: Failed to verify variable was set: %v", err)
	} else {
		log.Printf("DEBUG: Variable verification - %v", val)
	}
	
	return generatedData, nil
}

// Helper functions for data generation

// generateString generates a random string with specified options
func generateString(ctx *context.ExecutionContext, params map[string]interface{}) (interface{}, error) {
	// Check for direct value first
	if value, ok := params["value"].(string); ok {
		// Resolve any variables in the value
		resolvedValue, err := ctx.Substitute(value)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve value: %w", err)
		}
		return resolvedValue, nil
	}
	
	// Get length parameter
	length := 10 // Default length
	if lenParam, ok := params["length"].(float64); ok {
		length = int(lenParam)
	} else if lenParam, ok := params["length"].(int); ok {
		length = lenParam
	}
	
	// Get pattern parameter
	if pattern, ok := params["pattern"].(string); ok {
		// Resolve any variables in the pattern
		resolvedPattern, err := ctx.Substitute(pattern)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve pattern: %w", err)
		}
		
		// If pattern is provided, generate accordingly
		return generateFromPattern(resolvedPattern, length)
	}
	
	// Get charset parameter
	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	if charsetParam, ok := params["charset"].(string); ok {
		charset = charsetParam
	}
	
	// Generate random string
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		result[i] = charset[rand.Intn(len(charset))]
	}
	
	return string(result), nil
}

// generateNumber generates a random number within specified range
func generateNumber(ctx *context.ExecutionContext, params map[string]interface{}) (interface{}, error) {
	// Default range
	min := 0
	max := 100
	
	// Get min parameter
	if minParam, ok := params["min"].(float64); ok {
		min = int(minParam)
	} else if minParam, ok := params["min"].(int); ok {
		min = minParam
	}
	
	// Get max parameter
	if maxParam, ok := params["max"].(float64); ok {
		max = int(maxParam)
	} else if maxParam, ok := params["max"].(int); ok {
		max = maxParam
	}
	
	// Check if min is less than max
	if min >= max {
		return nil, fmt.Errorf("min must be less than max")
	}
	
	// Generate random number
	return min + rand.Intn(max-min+1), nil
}

// generateBoolean generates a random boolean
func generateBoolean(ctx *context.ExecutionContext, params map[string]interface{}) (interface{}, error) {
	return rand.Intn(2) == 1, nil
}

// generateUUID generates a random UUID
func generateUUID(ctx *context.ExecutionContext, params map[string]interface{}) (interface{}, error) {
	// Simple UUID generation (not fully RFC compliant)
	u := make([]byte, 16)
	_, err := rand.Read(u)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}
	
	// Set version (4) and variant bits
	u[6] = (u[6] & 0x0F) | 0x40 // Version 4
	u[8] = (u[8] & 0x3F) | 0x80 // Variant 1
	
	return fmt.Sprintf("%x-%x-%x-%x-%x", u[0:4], u[4:6], u[6:8], u[8:10], u[10:]), nil
}

// generateEmail generates a random email address
func generateEmail(ctx *context.ExecutionContext, params map[string]interface{}) (interface{}, error) {
	// Default domain
	domain := "example.com"
	if domainParam, ok := params["domain"].(string); ok {
		domain = domainParam
	}
	
	// Generate random username part
	usernameParams := map[string]interface{}{
		"length":  8,
		"charset": "abcdefghijklmnopqrstuvwxyz0123456789",
	}
	username, err := generateString(ctx, usernameParams)
	if err != nil {
		return nil, fmt.Errorf("failed to generate username: %w", err)
	}
	
	return fmt.Sprintf("%s@%s", username, domain), nil
}

// generateIP generates a random IP address
func generateIP(ctx *context.ExecutionContext, params map[string]interface{}) (interface{}, error) {
	// Check if we should generate IPv6
	ipv6 := false
	if v6Param, ok := params["ipv6"].(bool); ok {
		ipv6 = v6Param
	}
	
	if ipv6 {
		// Generate IPv6 address
		segments := make([]string, 8)
		for i := 0; i < 8; i++ {
			segments[i] = fmt.Sprintf("%04x", rand.Intn(65536))
		}
		return strings.Join(segments, ":"), nil
	}
	
	// Generate IPv4 address
	return fmt.Sprintf("%d.%d.%d.%d",
		rand.Intn(256),
		rand.Intn(256),
		rand.Intn(256),
		rand.Intn(256)), nil
}

// generateDate generates a random date
func generateDate(ctx *context.ExecutionContext, params map[string]interface{}) (interface{}, error) {
	// Default range: past 1 year to next 1 year
	now := time.Now()
	minTime := now.AddDate(-1, 0, 0)
	maxTime := now.AddDate(1, 0, 0)
	
	// Parse min date if provided
	if minParam, ok := params["min"].(string); ok {
		var err error
		minTime, err = parseDate(minParam)
		if err != nil {
			return nil, fmt.Errorf("invalid min date: %w", err)
		}
	}
	
	// Parse max date if provided
	if maxParam, ok := params["max"].(string); ok {
		var err error
		maxTime, err = parseDate(maxParam)
		if err != nil {
			return nil, fmt.Errorf("invalid max date: %w", err)
		}
	}
	
	// Check if min is before max
	if minTime.After(maxTime) {
		return nil, fmt.Errorf("min date must be before max date")
	}
	
	// Calculate random time between min and max
	diff := maxTime.Sub(minTime)
	randomDuration := time.Duration(rand.Int63n(int64(diff)))
	randomTime := minTime.Add(randomDuration)
	
	// Get format parameter
	format := time.RFC3339
	if formatParam, ok := params["format"].(string); ok {
		format = formatParam
	}
	
	return randomTime.Format(format), nil
}

// generateArray generates an array of random data
func generateArray(ctx *context.ExecutionContext, params map[string]interface{}) (interface{}, error) {
	// Get array size
	size := 5 // Default size
	if sizeParam, ok := params["size"].(float64); ok {
		size = int(sizeParam)
	} else if sizeParam, ok := params["size"].(int); ok {
		size = sizeParam
	}
	
	// Get item type
	itemType := "string" // Default type
	if typeParam, ok := params["item_type"].(string); ok {
		itemType = typeParam
	}
	
	// Get item parameters
	itemParams := map[string]interface{}{
		"type": itemType,
	}
	if itemParamsObj, ok := params["item_params"].(map[string]interface{}); ok {
		for k, v := range itemParamsObj {
			itemParams[k] = v
		}
	}
	
	// Generate array
	result := make([]interface{}, size)
	for i := 0; i < size; i++ {
		switch strings.ToLower(itemType) {
		case "string":
			val, err := generateString(ctx, itemParams)
			if err != nil {
				return nil, err
			}
			result[i] = val
		case "number", "integer":
			val, err := generateNumber(ctx, itemParams)
			if err != nil {
				return nil, err
			}
			result[i] = val
		case "boolean", "bool":
			val, err := generateBoolean(ctx, itemParams)
			if err != nil {
				return nil, err
			}
			result[i] = val
		default:
			return nil, fmt.Errorf("unsupported array item type: %s", itemType)
		}
	}
	
	return result, nil
}

// generateObject generates an object with random properties
func generateObject(ctx *context.ExecutionContext, params map[string]interface{}) (interface{}, error) {
	// Get properties schema
	properties, ok := params["properties"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("object generation requires 'properties' parameter")
	}
	
	result := make(map[string]interface{})
	
	// Generate each property
	for propName, propSpec := range properties {
		propConfig, ok := propSpec.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("property '%s' must be an object", propName)
		}
		
		// Get property type
		propType, ok := propConfig["type"].(string)
		if !ok {
			return nil, fmt.Errorf("property '%s' must have a 'type'", propName)
		}
		
		// Generate property value
		var propValue interface{}
		var err error
		
		switch strings.ToLower(propType) {
		case "string":
			propValue, err = generateString(ctx, propConfig)
		case "number", "integer":
			propValue, err = generateNumber(ctx, propConfig)
		case "boolean", "bool":
			propValue, err = generateBoolean(ctx, propConfig)
		case "uuid":
			propValue, err = generateUUID(ctx, propConfig)
		case "email":
			propValue, err = generateEmail(ctx, propConfig)
		case "ip":
			propValue, err = generateIP(ctx, propConfig)
		case "date":
			propValue, err = generateDate(ctx, propConfig)
		default:
			return nil, fmt.Errorf("unsupported property type: %s", propType)
		}
		
		if err != nil {
			return nil, fmt.Errorf("failed to generate property '%s': %w", propName, err)
		}
		
		result[propName] = propValue
	}
	
	return result, nil
}

// generateFromPattern generates a string based on a pattern with placeholders
func generateFromPattern(pattern string, maxLength int) (string, error) {
	// Handle special placeholders
	result := pattern
	
	// Replace numeric ranges like [0-9] with a random digit in that range
	rangeRegex := regexp.MustCompile(`\[(\d+)-(\d+)\]`)
	for {
		match := rangeRegex.FindStringSubmatch(result)
		if match == nil {
			break
		}
		
		min, err := strconv.Atoi(match[1])
		if err != nil {
			return "", fmt.Errorf("invalid range pattern: %s", match[0])
		}
		
		max, err := strconv.Atoi(match[2])
		if err != nil {
			return "", fmt.Errorf("invalid range pattern: %s", match[0])
		}
		
		if min > max {
			return "", fmt.Errorf("invalid range, min > max: %s", match[0])
		}
		
		randomVal := min + rand.Intn(max-min+1)
		result = strings.Replace(result, match[0], strconv.Itoa(randomVal), 1)
	}
	
	// Replace character classes
	charClasses := map[string]string{
		"{alpha}":      "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ",
		"{lower}":      "abcdefghijklmnopqrstuvwxyz",
		"{upper}":      "ABCDEFGHIJKLMNOPQRSTUVWXYZ",
		"{digit}":      "0123456789",
		"{alnum}":      "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789",
		"{hex}":        "0123456789abcdef",
		"{symbol}":     "!@#$%^&*()_+-=[]{}|;:,.<>?",
		"{whitespace}": " \t\n\r",
	}
	
	for placeholder, charset := range charClasses {
		for strings.Contains(result, placeholder) {
			randomChar := string(charset[rand.Intn(len(charset))])
			result = strings.Replace(result, placeholder, randomChar, 1)
		}
	}
	
	// Handle repeating patterns like {alpha*5} for 5 random alpha chars
	repeatRegex := regexp.MustCompile(`\{(\w+)\*(\d+)\}`)
	for {
		match := repeatRegex.FindStringSubmatch(result)
		if match == nil {
			break
		}
		
		className := "{" + match[1] + "}"
		count, err := strconv.Atoi(match[2])
		if err != nil {
			return "", fmt.Errorf("invalid repeat count: %s", match[0])
		}
		
		charset, exists := charClasses[className]
		if !exists {
			return "", fmt.Errorf("unknown character class: %s", className)
		}
		
		var replacement strings.Builder
		for i := 0; i < count; i++ {
			randomChar := charset[rand.Intn(len(charset))]
			replacement.WriteByte(randomChar)
		}
		
		result = strings.Replace(result, match[0], replacement.String(), 1)
	}
	
	// Truncate if necessary
	if maxLength > 0 && len(result) > maxLength {
		result = result[:maxLength]
	}
	
	return result, nil
}

// parseDuration parses a duration string with more formats than time.ParseDuration
// Supports: ms, s, m, h, d for milliseconds, seconds, minutes, hours, days
func parseDuration(durationStr string) (time.Duration, error) {
	// Try standard Go duration parsing first
	d, err := time.ParseDuration(durationStr)
	if err == nil {
		return d, nil
	}
	
	// Check for days format (e.g., "2d")
	if strings.HasSuffix(durationStr, "d") {
		daysPart := strings.TrimSuffix(durationStr, "d")
		days, err := strconv.ParseFloat(daysPart, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid days value: %s", daysPart)
		}
		
		// Convert days to hours (24 hours per day)
		return time.Duration(days * 24 * float64(time.Hour)), nil
	}
	
	return 0, fmt.Errorf("invalid duration format: %s", durationStr)
}

// parseDate parses a date string in common formats
func parseDate(dateStr string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z07:00", // ISO8601
		"2006-01-02 15:04:05",
		"2006-01-02",
		"01/02/2006",
		"02/01/2006",
	}
	
	for _, format := range formats {
		t, err := time.Parse(format, dateStr)
		if err == nil {
			return t, nil
		}
	}
	
	return time.Time{}, fmt.Errorf("failed to parse date: %s", dateStr)
}

// init registers utility action handlers
func init() {
	// wait和generate_data处理程序都已在专门的文件中注册，这里不再重复注册
	
	// Seed the random number generator
	rand.Seed(time.Now().UnixNano())
}
