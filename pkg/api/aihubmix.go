package api

import (
	"fmt"
)

// AiHubMixClient 实现AiHubMix API客户端
type AiHubMixClient struct {
	BaseAIClient
}

// NewAiHubMixClient 创建新的AiHubMix客户端
func NewAiHubMixClient() AIClient {
	return &AiHubMixClient{}
}

// BuildPrompt 构建AiHubMix API请求
func (c *AiHubMixClient) BuildPrompt(codeContent string, rules []Rule, model string, maxTokens int) (map[string]interface{}, error) {
	if len(rules) == 0 {
		return nil, fmt.Errorf("no rules provided")
	}

	// 使用BaseAIClient的方法构建提示词内容
	content := c.GetPromptContent(codeContent, rules)

	// 构建请求数据（OpenAI兼容格式）
	payload := map[string]interface{}{
		"model": model,
		"messages": []map[string]interface{}{
			{
				"role":    "user",
				"content": content,
			},
		},
		"temperature": 0.7,
		"max_tokens":  maxTokens,
	}

	return payload, nil
}

// ParseResponse 解析AiHubMix API响应
func (c *AiHubMixClient) ParseResponse(responseData map[string]interface{}) (string, error) {
	// 检查是否有choices字段
	choices, ok := responseData["choices"]
	if !ok {
		return "", fmt.Errorf("no choices in response")
	}

	choicesArray, ok := choices.([]interface{})
	if !ok || len(choicesArray) == 0 {
		return "", fmt.Errorf("invalid choices format")
	}

	// 获取第一个choice
	firstChoice, ok := choicesArray[0].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid choice format")
	}

	// 获取message
	message, ok := firstChoice["message"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("no message in choice")
	}

	// 获取content
	content, ok := message["content"].(string)
	if !ok {
		return "", fmt.Errorf("no content in message")
	}

	return content, nil
}
