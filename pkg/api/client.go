package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Rule 定义规则结构
type Rule struct {
	Name        string   `json:"name"`        // 规则名称
	Description string   `json:"description"` // 规则描述
	Extensions  []string `json:"extensions"`  // 要检查的文件后缀列表，如 [".lua", ".cpp"]
	Keywords    []string `json:"keywords"`    // 可选的关键字列表，文件内容需要包含其中任意一个关键字
	Enabled     bool     `json:"enabled"`     // 规则是否启用
}

// AIClient 定义AI API客户端接口
type AIClient interface {
	BuildPrompt(codeContent string, rules []Rule, model string, maxTokens int) (map[string]interface{}, error)
	ParseResponse(responseData map[string]interface{}) (string, error)
	CallAPI(payload map[string]interface{}, apiURL, apiKey string) (map[string]interface{}, error)
	SetLogFile(enable bool)
}

// BaseAIClient 提供基础实现
type BaseAIClient struct {
	enableLog bool
}

// SetLogFile 设置日志开关
func (c *BaseAIClient) SetLogFile(enable bool) {
	c.enableLog = enable
}

// logAPIRequest 记录API请求日志
func (c *BaseAIClient) logAPIRequest(payload map[string]interface{}, apiURL string) error {
	if !c.enableLog {
		return nil
	}

	// 创建logs目录
	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("create log directory failed: %v", err)
	}

	// 使用固定的日志文件名
	logFile := filepath.Join(logDir, "api_requests.log")

	// 打开日志文件（追加模式）
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open log file failed: %v", err)
	}
	defer f.Close()

	// 格式化请求数据
	payloadBytes, err := json.MarshalIndent(payload, "", "    ")
	if err != nil {
		return fmt.Errorf("marshal payload failed: %v", err)
	}

	// 写入日志
	logEntry := fmt.Sprintf("\n[%s] API Request to %s:\n%s\n",
		time.Now().Format("2006-01-02 15:04:05"),
		apiURL,
		string(payloadBytes))

	if _, err := f.WriteString(logEntry); err != nil {
		return fmt.Errorf("write log failed: %v", err)
	}

	return nil
}

// GetPromptContent 返回通用的提示词内容
func (c *BaseAIClient) GetPromptContent(codeContent string, rules []Rule) string {
	content := `我是一位资深的代码审计专家，现在需要你配合我对以下代码进行严格的安全性和质量审查。请你也以代码审计专家的身份，仔细分析代码中的每一个细节，不放过任何潜在的问题。

作为代码审计专家，我们需要：
1. 深入理解代码的意图和上下文
2. 仔细检查每一行代码的潜在问题
3. 考虑所有可能的边界情况和异常情况
4. 关注代码的健壮性和可维护性
5. 提供专业、具体且可执行的改进建议

请使用以下Markdown格式返回分析结果：
1. 对于发现的每个问题：
   - 使用二级标题(##)准确描述问题
   - 使用列表(-)详细说明问题的具体表现、可能造成的影响
   - 使用引用(>)给出专业的改进建议
   - 如果需要，使用代码块()展示正确的实现方式

如果确实没有发现任何问题，请返回："经过仔细审查，未发现任何问题。"

需要重点关注的规则：
`
	for _, rule := range rules {
		content += fmt.Sprintf("- %s: %s\n", rule.Name, rule.Description)
	}

	content += fmt.Sprintf("\n待审查的代码：\n```\n%s\n```", codeContent)
	return content
}

// CallAPI 提供基础的API调用实现
func (c *BaseAIClient) CallAPI(payload map[string]interface{}, apiURL, apiKey string) (map[string]interface{}, error) {
	// 记录API请求日志
	if err := c.logAPIRequest(payload, apiURL); err != nil {
		fmt.Printf("Warning: Failed to log API request: %v\n", err)
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload failed: %v", err)
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request failed: %v", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %v", err)
	}

	return result, nil
}
