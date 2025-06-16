package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/zx2/code-checker/pkg/api"
)

// Config 定义配置文件结构
type Config struct {
	// API配置
	API struct {
		Type          string `json:"type"`            // API类型：siliconflow 或 openai
		URL           string `json:"url"`             // API服务地址
		Key           string `json:"key"`             // API密钥
		Model         string `json:"model"`           // API使用的模型
		MaxTokens     int    `json:"max_tokens"`      // API返回的最大token数
		EnableLog     bool   `json:"enable_log"`      // 是否启用API请求日志
		MaxTextLength int    `json:"max_text_length"` // 单次请求最大文本长度
	} `json:"api"`

	// 检查配置
	Check struct {
		Directory   string `json:"directory"`   // 要检查的目录路径
		OutputDir   string `json:"output_dir"`  // 检查结果输出目录
		Concurrency int    `json:"concurrency"` // 并发检查任务数量
	} `json:"check"`

	// SVN配置
	SVN struct {
		LogLimit        int      `json:"log_limit"`        // SVN日志获取的最大记录数
		PriorityAuthors []string `json:"priority_authors"` // 优先级作者列表，优先选择这些作者中提交次数最多的
		FilterAfter     string   `json:"filter_after"`     // 过滤时间，格式：2024-01-01T00:00:00Z，只检查该时间之后有提交的文件
	} `json:"svn"`

	// 规则配置
	Rules []api.Rule `json:"rules"` // 检查规则列表
}

// LoadConfig 从文件加载配置
func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %v", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %v", err)
	}

	// 验证必要的配置项
	if err := config.validate(); err != nil {
		return nil, err
	}

	return &config, nil
}

// validate 验证配置是否完整
func (c *Config) validate() error {
	if c.API.URL == "" {
		return fmt.Errorf("缺少API URL配置")
	}
	if c.API.Key == "" {
		return fmt.Errorf("缺少API Key配置")
	}
	if c.Check.Directory == "" {
		return fmt.Errorf("缺少检查目录配置")
	}
	if c.API.Type == "" {
		c.API.Type = "siliconflow" // 默认使用siliconflow
	}
	if c.API.Model == "" {
		switch c.API.Type {
		case "aihubmix":
			c.API.Model = "gpt-3.5-turbo" // AiHubMix默认模型
		case "volcengine":
			c.API.Model = "doubao-1.5-pro-32k" // 火山引擎默认模型
		default:
			c.API.Model = "Pro/deepseek-ai/DeepSeek-R1" // 其他默认模型
		}
	}
	if c.API.MaxTokens <= 0 {
		switch c.API.Type {
		case "aihubmix":
			c.API.MaxTokens = 60000 // AiHubMix默认max_tokens
		case "volcengine":
			c.API.MaxTokens = 8192 // 火山引擎默认max_tokens
		default:
			c.API.MaxTokens = 8192 // 其他API默认max_tokens
		}
	}
	if c.Check.OutputDir == "" {
		c.Check.OutputDir = "check_results" // 默认输出目录
	}
	if c.API.MaxTextLength <= 0 {
		c.API.MaxTextLength = 4000 // 默认最大文本长度为4000字符
	}
	if c.SVN.LogLimit <= 0 {
		c.SVN.LogLimit = 30 // 默认获取最近30条SVN日志
	}
	if c.Check.Concurrency <= 0 {
		c.Check.Concurrency = 3 // 默认并发数为3
	}

	return nil
}
