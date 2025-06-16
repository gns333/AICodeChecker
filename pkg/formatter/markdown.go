package formatter

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/zx2/code-checker/pkg/svn"
)

// Result 定义检查结果结构
type Result struct {
	File         string   `json:"file"`
	Result       string   `json:"result"`
	AppliedRules []string `json:"applied_rules"`
}

// MarkdownFormatter 实现Markdown格式的结果输出
type MarkdownFormatter struct {
	outputDir          string
	ruleDirs           map[string]string
	svnLogLimit        int
	svnPriorityAuthors []string
}

// NewMarkdownFormatter 创建新的Markdown格式化器
func NewMarkdownFormatter(outputDir string, svnLogLimit int, svnPriorityAuthors []string) *MarkdownFormatter {
	return &MarkdownFormatter{
		outputDir:          outputDir,
		ruleDirs:           make(map[string]string),
		svnLogLimit:        svnLogLimit,
		svnPriorityAuthors: svnPriorityAuthors,
	}
}

// generateSafeFileName 生成安全的文件名（不带作者前缀）
func (f *MarkdownFormatter) generateSafeFileName(fileName string) string {
	// 只替换Windows不允许的特殊字符，保留中文等其他字符
	safeFileName := regexp.MustCompile(`[<>:"/\\|?*]`).ReplaceAllString(fileName, "_")
	return safeFileName
}

// generateFileNameWithAuthor 生成带作者前缀的文件名
func (f *MarkdownFormatter) generateFileNameWithAuthor(filePath, fileName string) string {
	// 生成基础安全文件名
	safeFileName := f.generateSafeFileName(fileName)

	// 获取文件的主要作者
	author := svn.GetFileAuthorSafe(filePath, f.svnLogLimit, f.svnPriorityAuthors)
	if author != "" {
		// 清理作者名中的特殊字符
		safeAuthor := regexp.MustCompile(`[<>:"/\\|?*]`).ReplaceAllString(author, "_")
		return fmt.Sprintf("[%s]%s", safeAuthor, safeFileName)
	}

	return safeFileName
}

// AddResult 添加检查结果
func (f *MarkdownFormatter) AddResult(result Result) error {
	if len(result.AppliedRules) == 0 {
		return nil
	}

	if result.Result == "没有匹配的规则" || result.Result == "未发现任何问题。" {
		return nil
	}

	for _, ruleName := range result.AppliedRules {
		// 只替换Windows不允许的特殊字符: < > : " / \ | ? *
		// 保留中文等其他字符
		ruleDirname := regexp.MustCompile(`[<>:"/\\|?*]`).ReplaceAllString(ruleName, "_")
		ruleDir := filepath.Join(f.outputDir, ruleDirname)

		if err := os.MkdirAll(ruleDir, 0755); err != nil {
			return fmt.Errorf("create rule directory failed: %v", err)
		}

		f.ruleDirs[ruleName] = ruleDir

		// 生成文件的检查结果文件
		fileName := filepath.Base(result.File)
		// 生成带作者前缀的文件名
		finalFileName := f.generateFileNameWithAuthor(result.File, fileName)
		resultFile := filepath.Join(ruleDir, finalFileName+".md")

		// 写入检查结果
		file, err := os.Create(resultFile)
		if err != nil {
			return fmt.Errorf("create result file failed: %v", err)
		}
		defer file.Close()

		currentTime := time.Now().Format("2006-01-02 15:04:05")

		// 获取作者信息用于显示
		author := svn.GetFileAuthorSafe(result.File, f.svnLogLimit, f.svnPriorityAuthors)
		authorInfo := ""
		if author != "" {
			authorInfo = fmt.Sprintf("主要作者：%s\n", author)
		}

		content := fmt.Sprintf("# 文件检查结果：%s\n\n检查时间：%s\n%s\n%s\n\n",
			result.File, currentTime, authorInfo, result.Result)

		if _, err := file.WriteString(content); err != nil {
			return fmt.Errorf("write result file failed: %v", err)
		}
	}

	return nil
}

// Close 关闭格式化器
func (f *MarkdownFormatter) Close() error {
	// Markdown格式化器不需要特殊的关闭操作
	return nil
}
