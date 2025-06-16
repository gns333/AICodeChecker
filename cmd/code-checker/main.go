package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/zx2/code-checker/pkg/api"
	"github.com/zx2/code-checker/pkg/checker"
	"github.com/zx2/code-checker/pkg/config"
)

func main() {
	var configFile = flag.String("config", "config.json", "配置文件路径")
	flag.Parse()

	// 加载配置文件
	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		fmt.Printf("加载配置文件失败: %v\n", err)
		os.Exit(1)
	}

	// 创建API客户端
	var apiClient api.AIClient
	switch cfg.API.Type {
	case "siliconflow":
		apiClient = &api.SiliconflowClient{}
	case "openai":
		apiClient = &api.OpenAIClient{}
	case "aihubmix":
		apiClient = &api.AiHubMixClient{}
	case "volcengine":
		apiClient = &api.VolcEngineClient{}
	default:
		fmt.Printf("不支持的API类型: %s\n", cfg.API.Type)
		os.Exit(1)
	}

	// 设置日志开关
	apiClient.SetLogFile(cfg.API.EnableLog)

	// 处理时间过滤参数
	var svnFilterAfter *time.Time
	if cfg.SVN.FilterAfter != "" {
		parsedTime, err := time.Parse(time.RFC3339, cfg.SVN.FilterAfter)
		if err != nil {
			fmt.Printf("解析SVN过滤时间失败: %v\n", err)
			os.Exit(1)
		}
		svnFilterAfter = &parsedTime
		fmt.Printf("启用SVN时间过滤，只检查 %s 之后有提交的文件\n", parsedTime.Format("2006-01-02 15:04:05"))
	}

	// 创建代码检查器
	checker, err := checker.NewCodeChecker(
		cfg.Rules,
		cfg.API.URL,
		cfg.API.Key,
		cfg.API.Model,
		cfg.API.MaxTextLength,
		cfg.API.MaxTokens,
		cfg.SVN.LogLimit,
		cfg.Check.Concurrency,
		cfg.SVN.PriorityAuthors,
		svnFilterAfter,
		apiClient,
	)
	if err != nil {
		fmt.Printf("创建代码检查器失败: %v\n", err)
		os.Exit(1)
	}

	// 执行目录检查
	if err := checker.CheckDirectory(cfg.Check.Directory, cfg.Check.OutputDir); err != nil {
		fmt.Printf("执行检查失败: %v\n", err)
		os.Exit(1)
	}
}
