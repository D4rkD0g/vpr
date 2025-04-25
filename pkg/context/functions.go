// Package context defines the ExecutionContext which holds the state during PoC execution.
// This file specifically implements the built-in functions (e.g., base64_encode, url_encode)
// available for use within variable substitution syntax (`{{ func(...) }}`) and
// provides a registration mechanism for these functions.
package context

import (
	"encoding/base64"
	"fmt"
	"html"
	"net/url"
	"strings"
	"time"
	"math/rand"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
)

// VariableFunction is the type for all built-in functions available in substitution.
type VariableFunction func(args ...interface{}) (interface{}, error)

var registeredFunctions map[string]VariableFunction

func init() {
	// Initialize random seed
	rand.Seed(time.Now().UnixNano())
	
	// Initialize function registry
	registeredFunctions = make(map[string]VariableFunction)
	
	// Register all built-in functions as defined in Section 11.3 of the specification
	// Basic encoding/decoding functions
	registeredFunctions["base64_encode"] = base64Encode
	registeredFunctions["base64_decode"] = base64Decode
	registeredFunctions["url_encode"] = urlEncode
	registeredFunctions["url_decode"] = urlDecode
	registeredFunctions["json_escape"] = jsonEscape
	registeredFunctions["html_escape"] = htmlEscape
	
	// JSON functions
	registeredFunctions["json_encode"] = jsonEncode
	registeredFunctions["json_decode"] = jsonDecode
	
	// Additional utility functions
	registeredFunctions["md5"] = md5Hash
	registeredFunctions["sha1"] = sha1Hash
	registeredFunctions["sha256"] = sha256Hash
	registeredFunctions["concat"] = concat
	registeredFunctions["length"] = length
	registeredFunctions["substring"] = substring
	registeredFunctions["replace"] = replace
	registeredFunctions["random_int"] = randomInt
	registeredFunctions["random_string"] = randomString
	registeredFunctions["timestamp"] = timestamp
	registeredFunctions["format_date"] = formatDate
}

// GetFunction retrieves a registered function by name.
func GetFunction(name string) (VariableFunction, bool) {
	f, ok := registeredFunctions[name]
	return f, ok
}

// RegisterFunction allows registering custom functions.
// This can be used by extension modules to add new functionality.
func RegisterFunction(name string, fn VariableFunction) error {
	if _, exists := registeredFunctions[name]; exists {
		return fmt.Errorf("function %s is already registered", name)
	}
	registeredFunctions[name] = fn
	return nil
}

// checkArgCount validates the number of arguments for a function.
func checkArgCount(name string, args []interface{}, expectedCount int) error {
	if len(args) != expectedCount {
		return fmt.Errorf("%s expects %d argument(s), got %d", name, expectedCount, len(args))
	}
	return nil
}

// toString attempts to convert any value to a string.
func toString(arg interface{}) (string, error) {
	switch v := arg.(type) {
	case string:
		return v, nil
	case fmt.Stringer:
		return v.String(), nil
	default:
		return fmt.Sprintf("%v", v), nil
	}
}

//
// Core Built-in Functions from DSL specification
//

// base64Encode encodes a string using base64.
func base64Encode(args ...interface{}) (interface{}, error) {
	if err := checkArgCount("base64_encode", args, 1); err != nil {
		return nil, err
	}
	
	inputStr, err := toString(args[0])
	if err != nil {
		return nil, fmt.Errorf("base64_encode requires a string argument: %w", err)
	}
	
	return base64.StdEncoding.EncodeToString([]byte(inputStr)), nil
}

// base64Decode decodes a base64 encoded string.
func base64Decode(args ...interface{}) (interface{}, error) {
	if err := checkArgCount("base64_decode", args, 1); err != nil {
		return nil, err
	}
	
	inputStr, err := toString(args[0])
	if err != nil {
		return nil, fmt.Errorf("base64_decode requires a string argument: %w", err)
	}
	
	decoded, err := base64.StdEncoding.DecodeString(inputStr)
	if err != nil {
		return nil, fmt.Errorf("invalid base64 string: %w", err)
	}
	
	return string(decoded), nil
}

// urlEncode URL-encodes a string.
func urlEncode(args ...interface{}) (interface{}, error) {
	if err := checkArgCount("url_encode", args, 1); err != nil {
		return nil, err
	}
	
	inputStr, err := toString(args[0])
	if err != nil {
		return nil, fmt.Errorf("url_encode requires a string argument: %w", err)
	}
	
	return url.QueryEscape(inputStr), nil
}

// urlDecode URL-decodes a string.
func urlDecode(args ...interface{}) (interface{}, error) {
	if err := checkArgCount("url_decode", args, 1); err != nil {
		return nil, err
	}
	
	inputStr, err := toString(args[0])
	if err != nil {
		return nil, fmt.Errorf("url_decode requires a string argument: %w", err)
	}
	
	decoded, err := url.QueryUnescape(inputStr)
	if err != nil {
		return nil, fmt.Errorf("invalid URL-encoded string: %w", err)
	}
	
	return decoded, nil
}

// jsonEscape escapes characters for embedding in JSON string values.
func jsonEscape(args ...interface{}) (interface{}, error) {
	if err := checkArgCount("json_escape", args, 1); err != nil {
		return nil, err
	}
	
	inputStr, err := toString(args[0])
	if err != nil {
		return nil, fmt.Errorf("json_escape requires a string argument: %w", err)
	}
	
	// Use the JSON encoder to properly escape the string
	encoded, err := json.Marshal(inputStr)
	if err != nil {
		return nil, fmt.Errorf("failed to JSON escape string: %w", err)
	}
	
	// Remove surrounding quotes from the JSON-encoded string
	result := string(encoded)
	if len(result) >= 2 && result[0] == '"' && result[len(result)-1] == '"' {
		result = result[1 : len(result)-1]
	}
	
	return result, nil
}

// htmlEscape escapes special characters for HTML.
func htmlEscape(args ...interface{}) (interface{}, error) {
	if err := checkArgCount("html_escape", args, 1); err != nil {
		return nil, err
	}
	
	inputStr, err := toString(args[0])
	if err != nil {
		return nil, fmt.Errorf("html_escape requires a string argument: %w", err)
	}
	
	return html.EscapeString(inputStr), nil
}

// jsonEncode encodes a value to JSON.
func jsonEncode(args ...interface{}) (interface{}, error) {
	if err := checkArgCount("json_encode", args, 1); err != nil {
		return nil, err
	}
	
	// Marshal the input value to JSON
	encoded, err := json.Marshal(args[0])
	if err != nil {
		return nil, fmt.Errorf("failed to JSON encode value: %w", err)
	}
	
	return string(encoded), nil
}

// jsonDecode decodes a JSON string to a value.
func jsonDecode(args ...interface{}) (interface{}, error) {
	if err := checkArgCount("json_decode", args, 1); err != nil {
		return nil, err
	}
	
	// Get the input JSON string
	inputStr, err := toString(args[0])
	if err != nil {
		return nil, fmt.Errorf("json_decode requires a string argument: %w", err)
	}
	
	// Unmarshal the JSON string to a value
	var result interface{}
	err = json.Unmarshal([]byte(inputStr), &result)
	if err != nil {
		return nil, fmt.Errorf("invalid JSON string: %w", err)
	}
	
	return result, nil
}

//
// Additional utility functions
//

// md5Hash generates an MD5 hash of the input string.
func md5Hash(args ...interface{}) (interface{}, error) {
	if err := checkArgCount("md5", args, 1); err != nil {
		return nil, err
	}
	
	inputStr, err := toString(args[0])
	if err != nil {
		return nil, fmt.Errorf("md5 requires a string argument: %w", err)
	}
	
	hash := md5.Sum([]byte(inputStr))
	return hex.EncodeToString(hash[:]), nil
}

// sha1Hash generates a SHA1 hash of the input string.
func sha1Hash(args ...interface{}) (interface{}, error) {
	if err := checkArgCount("sha1", args, 1); err != nil {
		return nil, err
	}
	
	inputStr, err := toString(args[0])
	if err != nil {
		return nil, fmt.Errorf("sha1 requires a string argument: %w", err)
	}
	
	hash := sha1.Sum([]byte(inputStr))
	return hex.EncodeToString(hash[:]), nil
}

// sha256Hash generates a SHA256 hash of the input string.
func sha256Hash(args ...interface{}) (interface{}, error) {
	if err := checkArgCount("sha256", args, 1); err != nil {
		return nil, err
	}
	
	inputStr, err := toString(args[0])
	if err != nil {
		return nil, fmt.Errorf("sha256 requires a string argument: %w", err)
	}
	
	hash := sha256.Sum256([]byte(inputStr))
	return hex.EncodeToString(hash[:]), nil
}

// concat concatenates multiple strings.
func concat(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("concat requires at least one argument")
	}
	
	var result strings.Builder
	for i, arg := range args {
		str, err := toString(arg)
		if err != nil {
			return nil, fmt.Errorf("concat argument %d is not a valid string: %w", i+1, err)
		}
		result.WriteString(str)
	}
	
	return result.String(), nil
}

// length returns the length of a string or array.
func length(args ...interface{}) (interface{}, error) {
	if err := checkArgCount("length", args, 1); err != nil {
		return nil, err
	}
	
	// Check if it's a string
	if str, ok := args[0].(string); ok {
		return len(str), nil
	}
	
	// Check if it's a slice
	value := reflect.ValueOf(args[0])
	if value.Kind() == reflect.Slice || value.Kind() == reflect.Array {
		return value.Len(), nil
	}
	
	// Try to convert to string as a fallback
	str, err := toString(args[0])
	if err != nil {
		return nil, fmt.Errorf("length requires a string or array argument")
	}
	
	return len(str), nil
}

// substring extracts a substring from start to end (optional).
func substring(args ...interface{}) (interface{}, error) {
	if len(args) < 2 || len(args) > 3 {
		return nil, fmt.Errorf("substring requires 2 or 3 arguments: string, start [, end]")
	}
	
	// Get the input string
	inputStr, err := toString(args[0])
	if err != nil {
		return nil, fmt.Errorf("substring first argument must be a string: %w", err)
	}
	
	// Parse start index
	startIndex, ok := args[1].(int)
	if !ok {
		// Try to convert from string
		startStr, err := toString(args[1])
		if err != nil {
			return nil, fmt.Errorf("substring second argument must be an integer: %w", err)
		}
		_, err = fmt.Sscanf(startStr, "%d", &startIndex)
		if err != nil {
			return nil, fmt.Errorf("substring second argument must be an integer: %w", err)
		}
	}
	
	// Validate start index
	if startIndex < 0 {
		startIndex = 0
	}
	if startIndex > len(inputStr) {
		return "", nil // Empty string if start is beyond the length
	}
	
	// If no end index is provided, return to the end
	if len(args) == 2 {
		return inputStr[startIndex:], nil
	}
	
	// Parse end index
	endIndex, ok := args[2].(int)
	if !ok {
		// Try to convert from string
		endStr, err := toString(args[2])
		if err != nil {
			return nil, fmt.Errorf("substring third argument must be an integer: %w", err)
		}
		_, err = fmt.Sscanf(endStr, "%d", &endIndex)
		if err != nil {
			return nil, fmt.Errorf("substring third argument must be an integer: %w", err)
		}
	}
	
	// Validate end index
	if endIndex < startIndex {
		return "", nil // Empty string if end is before start
	}
	if endIndex > len(inputStr) {
		endIndex = len(inputStr)
	}
	
	return inputStr[startIndex:endIndex], nil
}

// replace replaces occurrences of a substring with another.
func replace(args ...interface{}) (interface{}, error) {
	if err := checkArgCount("replace", args, 3); err != nil {
		return nil, err
	}
	
	// Get the input string
	inputStr, err := toString(args[0])
	if err != nil {
		return nil, fmt.Errorf("replace first argument must be a string: %w", err)
	}
	
	// Get the search string
	searchStr, err := toString(args[1])
	if err != nil {
		return nil, fmt.Errorf("replace second argument must be a string: %w", err)
	}
	
	// Get the replacement string
	replaceStr, err := toString(args[2])
	if err != nil {
		return nil, fmt.Errorf("replace third argument must be a string: %w", err)
	}
	
	return strings.ReplaceAll(inputStr, searchStr, replaceStr), nil
}

// randomInt generates a random integer between min and max (inclusive).
func randomInt(args ...interface{}) (interface{}, error) {
	if err := checkArgCount("random_int", args, 2); err != nil {
		return nil, err
	}
	
	// Parse min
	min, ok := args[0].(int)
	if !ok {
		// Try to convert from string
		minStr, err := toString(args[0])
		if err != nil {
			return nil, fmt.Errorf("random_int first argument must be an integer: %w", err)
		}
		_, err = fmt.Sscanf(minStr, "%d", &min)
		if err != nil {
			return nil, fmt.Errorf("random_int first argument must be an integer: %w", err)
		}
	}
	
	// Parse max
	max, ok := args[1].(int)
	if !ok {
		// Try to convert from string
		maxStr, err := toString(args[1])
		if err != nil {
			return nil, fmt.Errorf("random_int second argument must be an integer: %w", err)
		}
		_, err = fmt.Sscanf(maxStr, "%d", &max)
		if err != nil {
			return nil, fmt.Errorf("random_int second argument must be an integer: %w", err)
		}
	}
	
	// Validate range
	if max < min {
		return nil, fmt.Errorf("random_int max must be greater than or equal to min")
	}
	
	// Generate random number (inclusive of min and max)
	return min + rand.Intn(max-min+1), nil
}

// randomString generates a random string of specified length.
func randomString(args ...interface{}) (interface{}, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("random_string requires 1 or 2 arguments: length [, charset]")
	}
	
	// Parse length
	length, ok := args[0].(int)
	if !ok {
		// Try to convert from string
		lengthStr, err := toString(args[0])
		if err != nil {
			return nil, fmt.Errorf("random_string first argument must be an integer: %w", err)
		}
		_, err = fmt.Sscanf(lengthStr, "%d", &length)
		if err != nil {
			return nil, fmt.Errorf("random_string first argument must be an integer: %w", err)
		}
	}
	
	// Validate length
	if length <= 0 {
		return "", nil // Empty string for zero or negative length
	}
	
	// Default charset: alphanumeric
	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	
	// Use custom charset if provided
	if len(args) > 1 {
		customCharset, err := toString(args[1])
		if err != nil {
			return nil, fmt.Errorf("random_string second argument must be a string: %w", err)
		}
		if customCharset != "" {
			charset = customCharset
		}
	}
	
	// Generate random string
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		result[i] = charset[rand.Intn(len(charset))]
	}
	
	return string(result), nil
}

// timestamp returns the current Unix timestamp.
func timestamp(args ...interface{}) (interface{}, error) {
	if len(args) > 1 {
		return nil, fmt.Errorf("timestamp accepts at most 1 argument: [format]")
	}
	
	// Default: return Unix timestamp in seconds
	if len(args) == 0 {
		return time.Now().Unix(), nil
	}
	
	// Format as specified
	format, err := toString(args[0])
	if err != nil {
		return nil, fmt.Errorf("timestamp argument must be a string: %w", err)
	}
	
	switch strings.ToLower(format) {
	case "seconds", "s":
		return time.Now().Unix(), nil
	case "milliseconds", "ms":
		return time.Now().UnixNano() / int64(time.Millisecond), nil
	case "nanoseconds", "ns":
		return time.Now().UnixNano(), nil
	case "rfc3339":
		return time.Now().Format(time.RFC3339), nil
	case "iso8601":
		return time.Now().Format("2006-01-02T15:04:05Z07:00"), nil
	default:
		return nil, fmt.Errorf("unsupported timestamp format: %s", format)
	}
}

// formatDate formats a date according to the specified format.
func formatDate(args ...interface{}) (interface{}, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("format_date requires 1 or 2 arguments: date [, format]")
	}
	
	// Parse input date
	var date time.Time
	var err error
	
	switch v := args[0].(type) {
	case time.Time:
		date = v
	case int, int64:
		// Assume Unix timestamp in seconds
		unix, _ := v.(int64)
		date = time.Unix(unix, 0)
	default:
		// Try to parse as string
		var dateStr string
		dateStr, err = toString(args[0])
		if err != nil {
			return nil, fmt.Errorf("format_date first argument must be a date: %w", err)
		}
		
		// Try common formats
		formats := []string{
			time.RFC3339,
			"2006-01-02T15:04:05Z07:00", // ISO8601
			"2006-01-02 15:04:05",
			"2006-01-02",
		}
		
		parsed := false
		for _, format := range formats {
			parsedDate, parseErr := time.Parse(format, dateStr)
			if parseErr == nil {
				date = parsedDate
				parsed = true
				break
			}
		}
		
		if !parsed {
			return nil, fmt.Errorf("unable to parse date: %s", dateStr)
		}
	}
	
	// Default format: RFC3339
	outFormat := time.RFC3339
	
	// Use custom format if provided
	if len(args) > 1 {
		customFormat, err := toString(args[1])
		if err != nil {
			return nil, fmt.Errorf("format_date second argument must be a string: %w", err)
		}
		
		// Special format aliases
		switch customFormat {
		case "rfc3339":
			outFormat = time.RFC3339
		case "iso8601":
			outFormat = "2006-01-02T15:04:05Z07:00"
		case "date":
			outFormat = "2006-01-02"
		case "time":
			outFormat = "15:04:05"
		case "datetime":
			outFormat = "2006-01-02 15:04:05"
		default:
			outFormat = customFormat
		}
	}
	
	return date.Format(outFormat), nil
}
