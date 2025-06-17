package svn

import (
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// CommitInfo 表示提交信息
type CommitInfo struct {
	Revision string
	Author   string
	Date     string
	Message  string
}

// AuthorStats 表示作者统计信息
type AuthorStats struct {
	Author string
	Count  int
}

// GetFileMostActiveAuthor 获取文件最近指定次数提交中提交数量最多的作者
func GetFileMostActiveAuthor(filePath string, limit int, priorityAuthors []string) (string, error) {
	// 检查SVN是否可用
	if !isSvnAvailable() {
		return "", fmt.Errorf("SVN命令不可用")
	}

	// 检查文件是否在SVN版本控制下
	if !isFileInSvn(filePath) {
		return "", fmt.Errorf("文件不在SVN版本控制下: %s", filePath)
	}

	// 获取文件的提交历史
	commits, err := getFileCommits(filePath, limit)
	if err != nil {
		return "", fmt.Errorf("获取文件提交历史失败: %v", err)
	}

	if len(commits) == 0 {
		return "", fmt.Errorf("文件无提交历史: %s", filePath)
	}

	// 统计每个作者的提交次数
	authorCount := make(map[string]int)
	for _, commit := range commits {
		authorCount[commit.Author]++
	}

	// 找出提交次数最多的作者（考虑优先级）
	mostActiveAuthor := findMostActiveAuthor(authorCount, priorityAuthors)
	return mostActiveAuthor, nil
}

// isSvnAvailable 检查SVN命令是否可用
func isSvnAvailable() bool {
	cmd := exec.Command("svn", "--version")
	err := cmd.Run()
	return err == nil
}

// isFileInSvn 检查文件是否在SVN版本控制下
func isFileInSvn(filePath string) bool {
	cmd := exec.Command("svn", "info", filePath)
	err := cmd.Run()
	return err == nil
}

// getFileCommits 获取文件的提交历史
func getFileCommits(filePath string, limit int) ([]CommitInfo, error) {
	// 执行svn log命令
	cmd := exec.Command("svn", "log", "-r", "HEAD:1", "-l", strconv.Itoa(limit), "--xml", filePath)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("执行svn log命令失败: %v", err)
	}

	// 解析XML输出
	commits, err := parseXMLLog(string(output))
	if err != nil {
		return nil, fmt.Errorf("解析SVN日志失败: %v", err)
	}

	return commits, nil
}

// parseXMLLog 解析SVN XML格式的日志输出
func parseXMLLog(xmlContent string) ([]CommitInfo, error) {
	var commits []CommitInfo

	// 使用正则表达式解析XML内容，支持多行匹配
	// 匹配logentry节点，支持标签跨行
	logentryRe := regexp.MustCompile(`(?s)<logentry[^>]*revision="(\d+)"[^>]*>(.*?)</logentry>`)
	authorRe := regexp.MustCompile(`<author>(.*?)</author>`)
	dateRe := regexp.MustCompile(`<date>(.*?)</date>`)
	msgRe := regexp.MustCompile(`(?s)<msg>(.*?)</msg>`)

	logentryMatches := logentryRe.FindAllStringSubmatch(xmlContent, -1)
	for _, match := range logentryMatches {
		if len(match) < 3 {
			continue
		}

		revision := match[1]
		content := match[2]

		var author, date, message string

		// 提取作者
		if authorMatch := authorRe.FindStringSubmatch(content); len(authorMatch) > 1 {
			author = strings.TrimSpace(authorMatch[1])
		}

		// 提取日期
		if dateMatch := dateRe.FindStringSubmatch(content); len(dateMatch) > 1 {
			date = strings.TrimSpace(dateMatch[1])
		}

		// 提取消息
		if msgMatch := msgRe.FindStringSubmatch(content); len(msgMatch) > 1 {
			message = strings.TrimSpace(msgMatch[1])
		}

		// 如果作者不为空，添加到结果中
		if author != "" {
			commits = append(commits, CommitInfo{
				Revision: revision,
				Author:   author,
				Date:     date,
				Message:  message,
			})
		}
	}

	return commits, nil
}

// findMostActiveAuthor 找出提交次数最多的作者（考虑优先级）
func findMostActiveAuthor(authorCount map[string]int, priorityAuthors []string) string {
	if len(authorCount) == 0 {
		return ""
	}

	// 转换为切片并排序
	var stats []AuthorStats
	for author, count := range authorCount {
		stats = append(stats, AuthorStats{
			Author: author,
			Count:  count,
		})
	}

	// 按提交次数降序排序
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Count > stats[j].Count
	})

	// 如果有优先级作者列表，优先从中选择
	if len(priorityAuthors) > 0 {
		// 创建优先级作者映射表
		priorityMap := make(map[string]bool)
		for _, author := range priorityAuthors {
			priorityMap[author] = true
		}

		// 在优先级作者中找提交次数最多的
		for _, stat := range stats {
			if priorityMap[stat.Author] {
				return stat.Author
			}
		}
	}

	// 如果没有优先级作者或优先级作者都没有提交记录，返回提交次数最多的作者
	return stats[0].Author
}

// GetFileAuthorSafe 安全地获取文件的主要作者，如果失败返回空字符串
func GetFileAuthorSafe(filePath string, limit int, priorityAuthors []string) string {
	author, err := GetFileMostActiveAuthor(filePath, limit, priorityAuthors)
	if err != nil {
		// 静默处理错误，返回空字符串
		return ""
	}
	return author
}

// HasCommitsAfter 检查文件是否在指定时间之后有提交
func HasCommitsAfter(filePath string, afterTime time.Time) (bool, error) {
	// 检查SVN是否可用
	if !isSvnAvailable() {
		return false, fmt.Errorf("SVN命令不可用")
	}

	// 检查文件是否在SVN版本控制下
	if !isFileInSvn(filePath) {
		return false, fmt.Errorf("文件不在SVN版本控制下: %s", filePath)
	}

	// 获取文件的提交历史（只获取最近的几条记录进行检查）
	commits, err := getFileCommits(filePath, 10)
	if err != nil {
		return false, fmt.Errorf("获取文件提交历史失败: %v", err)
	}

	if len(commits) == 0 {
		return false, nil // 没有提交历史，视为没有在指定时间后提交
	}

	// 检查是否有在指定时间之后的提交
	for _, commit := range commits {
		commitTime, err := ParseCommitDate(commit.Date)
		if err != nil {
			continue // 跳过无法解析的日期
		}

		if commitTime.After(afterTime) {
			return true, nil
		}
	}

	return false, nil
}

// ParseCommitDate 解析SVN提交日期
func ParseCommitDate(dateStr string) (time.Time, error) {
	// SVN日期格式通常是: 2024-01-01T12:00:00.000000Z
	formats := []string{
		"2006-01-02T15:04:05.000000Z",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			// 直接加8小时，不带时区信息
			return t.Add(8 * time.Hour), nil
		}
	}

	return time.Time{}, fmt.Errorf("无法解析日期格式: %s", dateStr)
}

// HasCommitsAfterSafe 安全地检查文件是否在指定时间后有提交，如果失败返回true（允许检查）
func HasCommitsAfterSafe(filePath string, afterTime time.Time) bool {
	hasCommits, err := HasCommitsAfter(filePath, afterTime)
	if err != nil {
		// 如果检查失败，默认允许检查（返回true）
		return true
	}
	return hasCommits
}
