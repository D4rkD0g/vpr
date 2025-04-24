package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	
	"vpr/pkg/context"
	"vpr/pkg/executor"
	"vpr/pkg/poc"
)

// 临时凭证提供函数，用于解析凭证引用
func mockCredentialResolver(credRef string) (map[string]string, error) {
	// 创建模拟凭证
	switch credRef {
	case "victim_user_credentials":
		return map[string]string{
			"cookie":       "victim-cookie-value",
			"bearer_token": "victim-bearer-token",
		}, nil
	case "attacker_user_credentials":
		return map[string]string{
			"cookie":       "attacker-cookie-value",
			"bearer_token": "attacker-bearer-token",
		}, nil
	default:
		return nil, fmt.Errorf("unknown credential reference: %s", credRef)
	}
}

func main() {
	// 配置日志
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)
	
	// 要测试的PoC文件路径
	pocFile := filepath.Join("examples", "idor_librechat.yaml")
	
	// 加载和解析PoC文件
	pocDef, err := poc.LoadPocFromFile(pocFile)
	if err != nil {
		slog.Error("Failed to parse PoC file", "error", err)
		os.Exit(1)
	}
	
	// 打印基本信息
	fmt.Printf("Loaded PoC: %s\n", pocDef.Metadata.Title)
	fmt.Printf("Target: %s\n", pocDef.Metadata.TargetApplication.Name)
	
	// 创建执行上下文
	ctx, err := context.NewExecutionContext(&pocDef.Context)
	if err != nil {
		slog.Error("Failed to create execution context", "error", err)
		os.Exit(1)
	}
	
	// 注册模拟凭证解析函数 - 暂时注释掉，待实现凭证解析器接口
	// ctx.RegisterCredentialResolver(mockCredentialResolver)
	
	// 替换目标URL，用于本地测试
	// 注: 真实测试中应通过命令行参数指定或使用配置文件
	err = ctx.SetVariable("environment.target_host.value", "localhost:8080")
	if err != nil {
		slog.Error("Failed to set environment value", "error", err)
	}
	
	// 创建并配置执行器
	options := executor.DefaultOptions()
	options.VerboseOutput = true
	// options.DryRun = true // 暂时注释掉，需要在ExecutorOptions中添加这个选项
	
	// 执行PoC
	result, err := executor.Execute(pocDef, options)
	if err != nil {
		slog.Error("Execution failed", "error", err)
		os.Exit(1)
	}
	
	// 打印执行结果
	fmt.Println("\n--- Execution Results ---")
	fmt.Printf("Overall Success: %v\n", result.Success)
	fmt.Printf("Duration: %.2f seconds\n", result.Duration)
	if result.Error != nil {
		fmt.Printf("Error: %s\n", result.Error)
	}
	
	// 打印详细结果
	fmt.Println("\n--- Phase Results ---")
	for name, phaseResult := range result.PhaseResults {
		fmt.Printf("Phase: %s - Success: %v (%.2f s)\n", 
			name, phaseResult.Success, phaseResult.Duration)
		
		if phaseResult.Error != nil {
			fmt.Printf("  Error: %s\n", phaseResult.Error)
		}
		
		// 打印步骤结果
		for i, stepResult := range phaseResult.StepResults {
			fmt.Printf("  Step %d: %s\n", i+1, stepResult.DSL)
			fmt.Printf("    Success: %v, Duration: %.2f s\n", 
				stepResult.Success, stepResult.Duration)
			
			if stepResult.Skipped {
				fmt.Printf("    (Skipped)\n")
			}
			
			if stepResult.Error != nil {
				fmt.Printf("    Error: %s\n", stepResult.Error)
			}
		}
	}
}
