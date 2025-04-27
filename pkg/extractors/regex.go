// Package extractors provides the implementation for data extraction from responses.
// This file specifically implements regex-based extraction from response bodies.
package extractors

import (
	"fmt"
	"regexp"
	"log"
	
	"vpr/pkg/context"
	"vpr/pkg/poc"
)

// extractFromRegexHandler extracts data from response bodies using regex
// This is an alias for extractFromBodyRegexHandler to match the registry naming
func extractFromRegexHandler(ctx *context.ExecutionContext, action *poc.HTTPResponseAction, data interface{}) (interface{}, error) {
	return extractFromBodyRegexHandler(ctx, action, data)
}

// extractFromBodyRegexHandler extracts data from response bodies using regex
func extractFromBodyRegexHandler(ctx *context.ExecutionContext, action *poc.HTTPResponseAction, data interface{}) (interface{}, error) {
	if action.Type != "extract_from_body_regex" {
		return nil, fmt.Errorf("invalid extractor type for extractFromBodyRegexHandler: %s", action.Type)
	}
	
	// Validate required fields
	if action.Regex == "" {
		return nil, fmt.Errorf("extract_from_body_regex requires regex field")
	}
	
	if action.TargetVariable == "" {
		return nil, fmt.Errorf("extract_from_body_regex requires target_variable field")
	}
	
	// Get the source text
	var sourceText string
	
	// Determine source of text data
	if action.Source != "" {
		// Source specified, try to resolve from context
		sourceData, err := ctx.ResolveVariable(action.Source)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve source variable '%s': %w", action.Source, err)
		}
		
		// Convert source to string
		switch v := sourceData.(type) {
		case string:
			sourceText = v
		default:
			// Try to convert to string
			sourceText = fmt.Sprintf("%v", v)
		}
	} else if httpResp, ok := data.(map[string]interface{}); ok {
		// No source specified, use the data from HTTP response
		if bodyStr, ok := httpResp["body"].(string); ok {
			sourceText = bodyStr
		} else {
			return nil, fmt.Errorf("HTTP response body is not a string")
		}
	} else {
		return nil, fmt.Errorf("data is not an HTTP response")
	}
	
	// Resolve any variables in the regex pattern
	regexPattern, err := ctx.Substitute(action.Regex)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve regex pattern: %w", err)
	}
	
	// Compile the regex
	re, err := regexp.Compile(regexPattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern '%s': %w", regexPattern, err)
	}
	
	// Determine which extraction method to use
	var result interface{}
	if action.ExtractAll {
		// Extract all matches
		result, err = extractAllMatches(re, sourceText, action.Group)
	} else {
		// Extract first match only
		result, err = extractFirstMatch(re, sourceText, action.Group)
	}
	
	if err != nil {
		return nil, err
	}
	
	// Create a proper variable structure that matches the ContextVariable format
	varStruct := &poc.ContextVariable{
		ID:    action.TargetVariable,
		Value: result,
	}
	
	// Store the extracted data in the variables map
	varsPath := "variables." + action.TargetVariable
	if err := ctx.SetVariable(varsPath, varStruct); err != nil {
		return nil, fmt.Errorf("failed to set target variable: %w", err)
	}
	
	log.Printf("DEBUG: Regex extraction successful - target_variable='%s', extracted_value='%v'", 
		action.TargetVariable, result)
	
	return result, nil
}

// extractFirstMatch extracts the first match from the source text
func extractFirstMatch(re *regexp.Regexp, sourceText string, group *int) (interface{}, error) {
	// If we need to extract a specific capture group
	if group != nil {
		matches := re.FindStringSubmatch(sourceText)
		if matches == nil {
			return nil, fmt.Errorf("no matches found")
		}
		
		groupIndex := *group
		
		// Validate group index
		if groupIndex < 0 || groupIndex >= len(matches) {
			return nil, fmt.Errorf("invalid group index %d (only %d groups found)", 
				groupIndex, len(matches))
		}
		
		return matches[groupIndex], nil
	}
	
	// Extract the entire match
	match := re.FindString(sourceText)
	if match == "" {
		return nil, fmt.Errorf("no matches found")
	}
	
	return match, nil
}

// extractAllMatches extracts all matches from the source text
func extractAllMatches(re *regexp.Regexp, sourceText string, group *int) (interface{}, error) {
	// If we need to extract a specific capture group
	if group != nil {
		allMatches := re.FindAllStringSubmatch(sourceText, -1)
		if allMatches == nil || len(allMatches) == 0 {
			return nil, fmt.Errorf("no matches found")
		}
		
		groupIndex := *group
		
		// Validate group index using the first match
		if groupIndex < 0 || groupIndex >= len(allMatches[0]) {
			return nil, fmt.Errorf("invalid group index %d (only %d groups found)", 
				groupIndex, len(allMatches[0]))
		}
		
		// Extract the specified group from each match
		result := make([]string, len(allMatches))
		for i, matches := range allMatches {
			result[i] = matches[groupIndex]
		}
		
		return result, nil
	}
	
	// Extract all complete matches
	matches := re.FindAllString(sourceText, -1)
	if matches == nil || len(matches) == 0 {
		return nil, fmt.Errorf("no matches found")
	}
	
	return matches, nil
}

// init registers regex extractor handler
func init() {
	// Register the handler with the extractor registry
	MustRegisterExtractor("extract_from_body_regex", extractFromRegexHandler)
}
