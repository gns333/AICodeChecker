package api

// SiliconflowClient 实现硅基流动 API 客户端
type SiliconflowClient struct {
	BaseAIClient
}

// BuildPrompt 构建硅基流动 API 的请求数据
func (c *SiliconflowClient) BuildPrompt(codeContent string, rules []Rule, model string, maxTokens int) (map[string]interface{}, error) {
	content := c.GetPromptContent(codeContent, rules)
	return map[string]interface{}{
		"model": model,
		"messages": []map[string]interface{}{
			{
				"role":    "user",
				"content": content,
			},
		},
		"stream":            false,
		"max_tokens":        maxTokens,
		"stop":              []string{"null"},
		"temperature":       0.2,
		"top_p":             0.7,
		"top_k":             50,
		"frequency_penalty": 0.5,
		"n":                 1,
		"response_format": map[string]string{
			"type": "text",
		},
	}, nil
}

// ParseResponse 解析硅基流动 API 的响应数据
func (c *SiliconflowClient) ParseResponse(responseData map[string]interface{}) (string, error) {
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
