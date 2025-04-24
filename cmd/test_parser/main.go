// Package main provides a simple test for the PoC parser functionality.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"

	"vpr/pkg/poc"
)

func main() {
	// Check command line args
	if len(os.Args) < 2 {
		fmt.Println("Usage: test_parser <path-to-poc-file>")
		os.Exit(1)
	}

	filePath := os.Args[1]
	fmt.Printf("Parsing file: %s\n\n", filePath)

	// Load and parse the PoC file
	pocInstance, err := poc.LoadPocFromFile(filePath)
	if err != nil {
		color.Red("Error parsing PoC file: %v\n", err)
		os.Exit(1)
	}

	color.Green("✅ Successfully parsed PoC file!\n")
	color.Cyan("Title: %s\n", pocInstance.Metadata.Title)
	color.Cyan("ID: %s\n", pocInstance.Metadata.ID)
	color.Cyan("DSL Version: %s\n", pocInstance.Metadata.DslVersion)

	// Print detailed summary of what was loaded
	fmt.Println("\n📋 PoC Structure Summary:")
	
	// Context details
	fmt.Printf("- Context:\n")
	if len(pocInstance.Context.Users) > 0 {
		fmt.Printf("  - Users (%d):\n", len(pocInstance.Context.Users))
		for i, user := range pocInstance.Context.Users {
			fmt.Printf("    [%d] ID: %s, Description: %s\n", i, user.ID, 
				truncateString(user.Description, 50))
		}
	}
	
	if len(pocInstance.Context.Resources) > 0 {
		fmt.Printf("  - Resources (%d):\n", len(pocInstance.Context.Resources))
		for i, resource := range pocInstance.Context.Resources {
			fmt.Printf("    [%d] ID: %s, Type: %s\n", i, resource.ID, resource.Type)
		}
	}
	
	if len(pocInstance.Context.Environment) > 0 {
		fmt.Printf("  - Environment (%d):\n", len(pocInstance.Context.Environment))
		for i, env := range pocInstance.Context.Environment {
			fmt.Printf("    [%d] ID: %s, Value: %v\n", i, env.ID, env.Value)
		}
	}
	
	if len(pocInstance.Context.Variables) > 0 {
		fmt.Printf("  - Variables (%d):\n", len(pocInstance.Context.Variables))
		for i, variable := range pocInstance.Context.Variables {
			fmt.Printf("    [%d] ID: %s\n", i, variable.ID)
		}
	}
	
	if len(pocInstance.Context.Files) > 0 {
		fmt.Printf("  - Files (%d):\n", len(pocInstance.Context.Files))
		for i, file := range pocInstance.Context.Files {
			fmt.Printf("    [%d] ID: %s, LocalPath: %s\n", i, file.ID, file.LocalPath)
		}
	}
	
	// Setup steps
	if len(pocInstance.Setup) > 0 {
		fmt.Printf("- Setup (%d steps):\n", len(pocInstance.Setup))
		printSteps(pocInstance.Setup, "  ")
	} else {
		fmt.Printf("- Setup: None defined\n")
	}
	
	// Exploit scenario
	fmt.Printf("- Exploit Scenario: \"%s\"\n", pocInstance.Exploit.Name)
	
	if len(pocInstance.Exploit.Setup) > 0 {
		fmt.Printf("  - Scenario Setup (%d steps):\n", len(pocInstance.Exploit.Setup))
		printSteps(pocInstance.Exploit.Setup, "    ")
	}
	
	fmt.Printf("  - Scenario Steps (%d):\n", len(pocInstance.Exploit.Steps))
	printSteps(pocInstance.Exploit.Steps, "    ")
	
	if len(pocInstance.Exploit.Teardown) > 0 {
		fmt.Printf("  - Scenario Teardown (%d steps):\n", len(pocInstance.Exploit.Teardown))
		printSteps(pocInstance.Exploit.Teardown, "    ")
	}
	
	// Assertions
	fmt.Printf("- Assertions (%d steps):\n", len(pocInstance.Assertions))
	printSteps(pocInstance.Assertions, "  ")
	
	// Verification
	if len(pocInstance.Verification) > 0 {
		fmt.Printf("- Verification (%d steps):\n", len(pocInstance.Verification))
		printSteps(pocInstance.Verification, "  ")
	} else {
		fmt.Printf("- Verification: None defined\n")
	}

	// Save as YAML to verify our understanding
	tmpDir := os.TempDir()
	outputPath := filepath.Join(tmpDir, "parsed_poc.yaml")

	// Save with wrapper (poc: at the top level)
	err = poc.SavePocToFile(pocInstance, outputPath, true)
	if err != nil {
		color.Red("Error saving parsed PoC: %v\n", err)
	} else {
		color.Green("\n✅ Saved parsed PoC to: %s\n", outputPath)
		fmt.Println("You can compare this with the original to verify the parsing is correct.")
	}
	
	// Validate the PoC
	color.Yellow("\n🔍 Running additional validation checks...")
	err = poc.ValidatePoc(pocInstance)
	if err != nil {
		color.Red("❌ Validation failed: %v\n", err)
	} else {
		color.Green("✅ All validation checks passed!\n")
	}
}

// printSteps prints summary information about a list of steps
func printSteps(steps []poc.Step, indent string) {
	for i, step := range steps {
		stepDescription := fmt.Sprintf("[%d] %s", i+1, truncateString(step.DSL, 60))
		fmt.Printf("%s%s\n", indent, stepDescription)
		
		// Print action details if present
		if step.Action != nil {
			fmt.Printf("%s  Action: %s\n", indent, step.Action.Type)
			if step.Action.Type == "http_request" && step.Action.Request != nil {
				fmt.Printf("%s    HTTP %s %s\n", indent, 
					step.Action.Request.Method, truncateString(step.Action.Request.URL, 50))
			}
		}
		
		// Print check details if present
		if step.Check != nil {
			fmt.Printf("%s  Check: %s\n", indent, step.Check.Type)
		}
		
		// Print loop details if present
		if step.Loop != nil {
			fmt.Printf("%s  Loop over '%s' as '%s' with %d nested steps\n", 
				indent, step.Loop.Over, step.Loop.VariableName, len(step.Loop.Steps))
		}
	}
}

// truncateString shortens a string if it's too long
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
