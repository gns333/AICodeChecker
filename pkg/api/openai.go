package api

// OpenAIClient 实现OpenAI API客户端
type OpenAIClient struct {
	BaseAIClient
}

// BuildPrompt 构建OpenAI API的请求数据
func (c *OpenAIClient) BuildPrompt(codeContent string, rules []Rule, model string, maxTokens int) (map[string]interface{}, error) {
	content := c.GetPromptContent(codeContent, rules)
	return map[string]interface{}{
		"model": model,
		"messages": []map[string]interface{}{
			{
				"role":    "system",
				"content": "你是一个专业的代码审计专家，擅长发现代码中的潜在问题和安全隐患。",
			},
			{
				"role":    "user",
				"content": content,
			},
		},
		"temperature":       0.2,
		"max_tokens":        maxTokens,
		"top_p":             0.95,
		"frequency_penalty": 0,
		"presence_penalty":  0,
	}, nil
}

// ParseResponse 解析OpenAI API的响应数据
func (c *OpenAIClient) ParseResponse(responseData map[string]interface{}) (string, error) {
	choices, ok := responseData["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return "API返回结果格式错误", nil
	}

	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		return "API返回结果格式错误", nil
	}

	message, ok := choice["message"].(map[string]interface{})
	if !ok {
		return "API返回结果格式错误", nil
	}

	content, ok := message["content"].(string)
	if !ok {
		return "API返回结果格式错误", nil
	}

	return content, nil
}
