package checker

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/zx2/code-checker/pkg/api"
	"github.com/zx2/code-checker/pkg/formatter"
	"github.com/zx2/code-checker/pkg/svn"
)

// CodeChecker 实现代码检查器
type CodeChecker struct {
	rules              []api.Rule
	apiURL             string
	apiKey             string
	apiModel           string
	maxTextLength      int
	maxTokens          int
	svnLogLimit        int
	svnPriorityAuthors []string
	svnFilterAfter     *time.Time
	concurrency        int
	apiClient          api.AIClient
}

// NewCodeChecker 创建新的代码检查器
func NewCodeChecker(rules []api.Rule, apiURL, apiKey, apiModel string, maxTextLength, maxTokens, svnLogLimit, concurrency int, svnPriorityAuthors []string, svnFilterAfter *time.Time, apiClient api.AIClient) (*CodeChecker, error) {
	return &CodeChecker{
		rules:              rules,
		apiURL:             apiURL,
		apiKey:             apiKey,
		apiModel:           apiModel,
		maxTextLength:      maxTextLength,
		maxTokens:          maxTokens,
		svnLogLimit:        svnLogLimit,
		svnPriorityAuthors: svnPriorityAuthors,
		svnFilterAfter:     svnFilterAfter,
		concurrency:        concurrency,
		apiClient:          apiClient,
	}, nil
}

// splitCodeContent 将代码内容按行分片
func (c *CodeChecker) splitCodeContent(content string) []string {
	if len(content) <= c.maxTextLength {
		return []string{content}
	}

	lines := strings.Split(content, "\n")
	var chunks []string
	var currentChunk strings.Builder
	lineCount := 0

	for _, line := range lines {
		// 如果当前分片加上新行会超过最大长度，就开始新的分片
		if currentChunk.Len()+len(line)+1 > c.maxTextLength && currentChunk.Len() > 0 {
			chunks = append(chunks, currentChunk.String())
			currentChunk.Reset()
			lineCount = 0
		}

		// 添加新行到当前分片
		if currentChunk.Len() > 0 {
			currentChunk.WriteString("\n")
		}
		currentChunk.WriteString(line)
		lineCount++
	}

	// 添加最后一个分片
	if currentChunk.Len() > 0 {
		chunks = append(chunks, currentChunk.String())
	}

	return chunks
}

// 定义常量
const noIssuesFound = "经过仔细审查，未发现任何问题。"

// mergeResults 合并多个分片的检查结果
func (c *CodeChecker) mergeResults(results []string) string {
	if len(results) == 0 {
		return noIssuesFound
	}

	// 合并所有结果
	var mergedResult strings.Builder
	if len(results) > 1 {
		mergedResult.WriteString("# 文件检查结果汇总\n\n")
	}

	for i, result := range results {
		if len(results) > 1 {
			mergedResult.WriteString(fmt.Sprintf("## 第%d部分\n\n", i+1))
		}
		mergedResult.WriteString(result)
		mergedResult.WriteString("\n\n")
	}

	return mergedResult.String()
}

// matchExtension 检查文件后缀是否匹配规则
func (c *CodeChecker) matchExtension(filePath string, rule api.Rule) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	for _, ruleExt := range rule.Extensions {
		if strings.ToLower(ruleExt) == ext {
			return true
		}
	}
	return false
}

// matchKeywords 检查文件内容是否包含关键字
func (c *CodeChecker) matchKeywords(content string, rule api.Rule) bool {
	if len(rule.Keywords) == 0 {
		return true
	}
	for _, keyword := range rule.Keywords {
		if strings.Contains(content, keyword) {
			return true
		}
	}
	return false
}

// getApplicableRules 获取适用于文件的规则，needContent表示是否需要读取文件内容
func (c *CodeChecker) getApplicableRules(filePath string) (rules []api.Rule, needContent bool) {
	for _, rule := range c.rules {
		if !rule.Enabled {
			continue
		}

		// 先检查文件后缀
		if !c.matchExtension(filePath, rule) {
			continue
		}

		// 如果规则需要检查关键字，标记需要读取内容
		if len(rule.Keywords) > 0 {
			needContent = true
		}
		rules = append(rules, rule)
	}
	return rules, needContent
}

// filterRulesByContent 根据文件内容过滤规则
func (c *CodeChecker) filterRulesByContent(rules []api.Rule, content string) []api.Rule {
	var filtered []api.Rule
	for _, rule := range rules {
		if c.matchKeywords(content, rule) {
			filtered = append(filtered, rule)
		}
	}
	return filtered
}

// checkFileWithRule 检查单个文件的单个规则
func (c *CodeChecker) checkFileWithRule(filePath string, rule api.Rule) ([]formatter.Result, error) {
	// 读取文件内容
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read file failed: %v", err)
	}

	// 将代码内容分片
	chunks := c.splitCodeContent(string(content))
	var chunkResults []string

	// 对每个分片进行检查
	for i, chunk := range chunks {
		// 构建请求数据
		payload, err := c.apiClient.BuildPrompt(chunk, []api.Rule{rule}, c.apiModel, c.maxTokens)
		if err != nil {
			return nil, fmt.Errorf("build prompt failed: %v", err)
		}

		// 调用API
		responseData, err := c.apiClient.CallAPI(payload, c.apiURL, c.apiKey)
		if err != nil {
			return nil, fmt.Errorf("call API failed: %v", err)
		}

		// 解析响应
		result, err := c.apiClient.ParseResponse(responseData)
		if err != nil {
			return nil, fmt.Errorf("parse response failed: %v", err)
		}

		chunkResults = append(chunkResults, result)

		// 如果不是最后一个分片，等待一秒再继续
		if i < len(chunks)-1 {
			time.Sleep(time.Second)
		}
	}

	// 合并所有分片的结果
	mergedResult := c.mergeResults(chunkResults)

	return []formatter.Result{{
		File:         filePath,
		Result:       mergedResult,
		AppliedRules: []string{rule.Name},
	}}, nil
}

// CheckDirectory 检查目录
func (c *CodeChecker) CheckDirectory(directory, outputDir string) error {
	// 记录开始时间
	startTime := time.Now()
	fmt.Printf("开始检查，开始时间：%s (并发数: %d)\n", startTime.Format("2006-01-02 15:04:05"), c.concurrency)

	var files []string
	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("walk directory failed: %v", err)
	}

	fmt.Printf("找到 %d 个文件需要检查\n", len(files))

	enabledRules := 0
	for _, rule := range c.rules {
		if rule.Enabled {
			enabledRules++
		}
	}

	f := formatter.NewMarkdownFormatter(outputDir, c.svnLogLimit, c.svnPriorityAuthors)

	// 定义检查任务结构
	type checkTask struct {
		filePath string
		rule     api.Rule
	}

	type checkResult struct {
		task     checkTask
		result   formatter.Result
		err      error
		duration time.Duration
	}

	// 创建任务列表
	var tasks []checkTask
	total := 0
	skipped := 0

	for _, filePath := range files {
		// 先根据文件后缀获取可能适用的规则
		applicableRules, needContent := c.getApplicableRules(filePath)
		if len(applicableRules) == 0 {
			total += enabledRules
			fmt.Printf("跳过文件(后缀不匹配): %s\n", filePath)
			continue
		}

		// 如果需要检查文件内容
		if needContent {
			content, err := os.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("read file failed: %v", err)
			}
			// 根据内容进一步过滤规则
			applicableRules = c.filterRulesByContent(applicableRules, string(content))
			if len(applicableRules) == 0 {
				total += enabledRules
				fmt.Printf("跳过文件(关键字不匹配): %s\n", filePath)
				continue
			}
		}

		// 检查SVN时间过滤
		if c.svnFilterAfter != nil {
			if !svn.HasCommitsAfterSafe(filePath, *c.svnFilterAfter) {
				total += enabledRules
				fmt.Printf("跳过文件(无最近提交): %s\n", filePath)
				continue
			}
		}

		// 为每个规则创建检查任务
		for _, rule := range applicableRules {
			if c.resultExists(filePath, rule.Name, outputDir) {
				skipped++
				total++
				fmt.Printf("跳过已存在的检查结果: %s - %s\n", filePath, rule.Name)
				continue
			}
			tasks = append(tasks, checkTask{filePath: filePath, rule: rule})
			total++
		}
	}

	fmt.Printf("实际需要检查的任务数: %d\n", len(tasks))

	// 创建channel进行通信
	taskChan := make(chan checkTask, len(tasks))
	resultChan := make(chan checkResult, len(tasks))

	// 启动goroutine池
	var wg sync.WaitGroup
	for i := 0; i < c.concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for task := range taskChan {
				checkStartTime := time.Now()

				// 执行单个文件的单个规则检查
				results, err := c.checkFileWithRule(task.filePath, task.rule)
				duration := time.Since(checkStartTime)

				if err != nil {
					resultChan <- checkResult{
						task:     task,
						err:      err,
						duration: duration,
					}
				} else if len(results) > 0 {
					resultChan <- checkResult{
						task:     task,
						result:   results[0], // 单个规则只会返回一个结果
						duration: duration,
					}
				}
			}
		}(i)
	}

	// 发送所有任务到channel
	go func() {
		for _, task := range tasks {
			taskChan <- task
		}
		close(taskChan)
	}()

	// 收集结果
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// 处理结果
	completed := 0
	checkedFiles := make(map[string]bool) // 用于跟踪已检查的唯一文件
	for result := range resultChan {
		completed++
		totalDuration := time.Since(startTime)

		if result.err != nil {
			return fmt.Errorf("检查文件 %s 规则 %s 失败: %v", result.task.filePath, result.task.rule.Name, result.err)
		}

		// 添加结果到formatter
		if err := f.AddResult(result.result); err != nil {
			return fmt.Errorf("add result failed: %v", err)
		}

		// 记录已检查的文件
		checkedFiles[result.task.filePath] = true

		fmt.Printf("进度: %d/%d - 检查完成: %s - %s [单次耗时: %v, 总耗时: %v, 已检查文件: %d]\n",
			completed, len(tasks), result.task.filePath, result.task.rule.Name,
			result.duration.Round(time.Millisecond), totalDuration.Round(time.Second), len(checkedFiles))
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("close formatter failed: %v", err)
	}

	// 计算总耗时
	totalDuration := time.Since(startTime)
	endTime := time.Now()

	fmt.Printf("检查完成，报告已生成到目录: %s\n", outputDir)
	fmt.Printf("总计跳过 %d 个已存在的检查结果\n", skipped)
	fmt.Printf("总耗时: %v (开始时间: %s, 结束时间: %s)\n",
		totalDuration.Round(time.Second),
		startTime.Format("2006-01-02 15:04:05"),
		endTime.Format("2006-01-02 15:04:05"))
	return nil
}

// resultExists 检查结果文件是否已存在（检查带作者前缀的文件）
func (c *CodeChecker) resultExists(filePath, ruleName, outputDir string) bool {
	// 只替换Windows不允许的特殊字符: < > : " / \ | ? *
	// 保留中文等其他字符
	ruleDirname := regexp.MustCompile(`[<>:"/\\|?*]`).ReplaceAllString(ruleName, "_")
	ruleDir := filepath.Join(outputDir, ruleDirname)

	fileName := filepath.Base(filePath)
	// 生成基础安全文件名（不带作者前缀）
	safeFileName := regexp.MustCompile(`[<>:"/\\|?*]`).ReplaceAllString(fileName, "_")

	// 检查是否存在任何以这个基础文件名结尾的文件
	// 这样可以匹配带作者前缀的文件，如 [author]filename.lua.md
	pattern := filepath.Join(ruleDir, "*"+safeFileName+".md")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return false
	}

	// 如果找到匹配的文件，说明结果已存在
	return len(matches) > 0
}

// CheckFile 检查单个文件
func (c *CodeChecker) CheckFile(filePath string) ([]formatter.Result, error) {
	// 先根据文件后缀获取可能适用的规则
	applicableRules, needContent := c.getApplicableRules(filePath)
	if len(applicableRules) == 0 {
		return []formatter.Result{{
			File:         filePath,
			Result:       "没有匹配的规则",
			AppliedRules: []string{},
		}}, nil
	}

	// 读取文件内容
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read file failed: %v", err)
	}

	// 如果需要检查文件内容，进一步过滤规则
	if needContent {
		applicableRules = c.filterRulesByContent(applicableRules, string(content))
		if len(applicableRules) == 0 {
			return []formatter.Result{{
				File:         filePath,
				Result:       "没有匹配的规则",
				AppliedRules: []string{},
			}}, nil
		}
	}

	var results []formatter.Result
	for _, rule := range applicableRules {
		// 将代码内容分片
		chunks := c.splitCodeContent(string(content))
		var chunkResults []string

		// 对每个分片进行检查
		for i, chunk := range chunks {
			// 构建请求数据
			payload, err := c.apiClient.BuildPrompt(chunk, []api.Rule{rule}, c.apiModel, c.maxTokens)
			if err != nil {
				return nil, fmt.Errorf("build prompt failed: %v", err)
			}

			// 调用API
			responseData, err := c.apiClient.CallAPI(payload, c.apiURL, c.apiKey)
			if err != nil {
				return nil, fmt.Errorf("call API failed: %v", err)
			}

			// 解析响应
			result, err := c.apiClient.ParseResponse(responseData)
			if err != nil {
				return nil, fmt.Errorf("parse response failed: %v", err)
			}

			chunkResults = append(chunkResults, result)

			// 如果不是最后一个分片，等待一秒再继续
			if i < len(chunks)-1 {
				time.Sleep(time.Second)
			}
		}

		// 合并所有分片的结果
		mergedResult := c.mergeResults(chunkResults)

		results = append(results, formatter.Result{
			File:         filePath,
			Result:       mergedResult,
			AppliedRules: []string{rule.Name},
		})
	}

	return results, nil
}
