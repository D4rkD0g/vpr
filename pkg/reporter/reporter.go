// Package reporter provides functions for formatting and outputting execution results.
package reporter

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/fatih/color"
	"vpr/pkg/executor"
)

// PrintResult formats and prints the execution result to the provided writer.
func PrintResult(result *executor.ExecutionResult, w io.Writer) {
	if result == nil {
		fmt.Fprintln(w, "No result available.")
		return
	}

	// Create colored output helpers
	success := color.New(color.FgGreen).SprintFunc()
	failure := color.New(color.FgRed).SprintFunc()
	highlight := color.New(color.FgCyan).SprintFunc()
	warning := color.New(color.FgYellow).SprintFunc()
	
	// Print header
	fmt.Fprintf(w, "\n%s\n", strings.Repeat("=", 80))
	fmt.Fprintf(w, "PoC Execution Result: %s\n", result.PocID)
	fmt.Fprintf(w, "%s\n\n", strings.Repeat("-", 80))
	
	// Overall status
	statusStr := success("SUCCESS")
	if !result.Success {
		statusStr = failure("FAILURE")
	}
	fmt.Fprintf(w, "Overall Status: %s\n", statusStr)
	fmt.Fprintf(w, "Execution Time: %s\n\n", time.Duration(result.Duration*float64(time.Second)))
	
	// Phase results
	if setupResult, ok := result.PhaseResults["setup"]; ok {
		printPhaseResult(w, "Setup", setupResult, highlight, success, failure, warning)
	}
	if exploitResult, ok := result.PhaseResults["exploit"]; ok {
		printPhaseResult(w, "Exploit", exploitResult, highlight, success, failure, warning)
	}
	if assertionsResult, ok := result.PhaseResults["assertions"]; ok {
		printPhaseResult(w, "Assertions", assertionsResult, highlight, success, failure, warning)
	}
	if verificationResult, ok := result.PhaseResults["verification"]; ok {
		printPhaseResult(w, "Verification", verificationResult, highlight, success, failure, warning)
	}
	
	// Print footer
	fmt.Fprintf(w, "\n%s\n", strings.Repeat("=", 80))
}

// printPhaseResult formats and prints the result for a specific execution phase.
func printPhaseResult(
	w io.Writer, 
	phaseName string, 
	phaseResult *executor.PhaseResult,
	highlight, success, failure, warning func(a ...interface{}) string,
) {
	if phaseResult == nil {
		fmt.Fprintf(w, "%s: %s\n", highlight(phaseName), warning("Not executed"))
		return
	}
	
	// Phase header
	statusStr := success("PASSED")
	if !phaseResult.Success {
		statusStr = failure("FAILED")
	}
	fmt.Fprintf(w, "%s: %s (%d steps, %s)\n", 
		highlight(phaseName), 
		statusStr,
		len(phaseResult.StepResults),
		time.Duration(phaseResult.Duration*float64(time.Second)),
	)
	
	// Step details
	if len(phaseResult.StepResults) > 0 {
		fmt.Fprintln(w, "  Steps:")
		for i, step := range phaseResult.StepResults {
			stepStatus := success("✓")
			if !step.Success {
				stepStatus = failure("✗")
			}
			
			// Format step info
			stepInfo := fmt.Sprintf("%d. %s", i+1, step.DSL)
			if len(stepInfo) > 60 {
				stepInfo = stepInfo[:57] + "..."
			}
			
			fmt.Fprintf(w, "  %s %s\n", stepStatus, stepInfo)
			
			// Show error if any
			if step.Error != nil {
				fmt.Fprintf(w, "     %s\n", failure(step.Error.Error()))
			}
		}
	}
	
	// Show phase error if any and not in step results
	if phaseResult.Error != nil {
		fmt.Fprintf(w, "  Error: %s\n", failure(phaseResult.Error.Error()))
	}
	
	fmt.Fprintln(w)
}
