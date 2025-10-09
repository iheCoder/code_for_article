package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
)

// MarkdownCleaner 用于清理markdown文件
type MarkdownCleaner struct {
	// 匹配URL引用的正则表达式，如：[reallyniceday.com](https://...)
	urlRefRegex *regexp.Regexp
	// 匹配错误转义的强调符号，如：\*
	escapeStarRegex *regexp.Regexp
	// 匹配多余空格（两个及以上的空格）
	multiSpaceRegex *regexp.Regexp
	// 新增：匹配被整体转义期望退化为单星号的 \\** 模式
	escapedDoubleStarToSingle *regexp.Regexp
}

// NewMarkdownCleaner 创建一个新的markdown清理器
func NewMarkdownCleaner() *MarkdownCleaner {
	return &MarkdownCleaner{
		// 匹配形如 [domain.com](url) 的引用
		urlRefRegex: regexp.MustCompile(`\[[^\]]*\]\([^)]*\)`),
		// 匹配转义的星号 \*
		escapeStarRegex: regexp.MustCompile(`\\(\*)`),
		// 匹配多余空格（两个及以上的空格）
		multiSpaceRegex: regexp.MustCompile(` {2,}`),
		// 新增：匹配被整体转义期望退化为单星号的 \\** 模式
		escapedDoubleStarToSingle: regexp.MustCompile(`\\\*{2}`),
	}
}

// isLetterOrDigit 判断字母/数字/下划线
func isLetterOrDigit(r rune) bool {
	return (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_'
}

// isRemovalPunct 判断是否是可触发脚注数字移除的标点（包含中英文）
func isRemovalPunct(r rune) bool {
	switch r {
	case '.', ',', '，', '。', '!', '?', ';', '；', ':', '：', ')', '）', ']', '】':
		return true
	}
	return false
}

// removeFootnoteNumbers 使用 rune 级遍历移除形如 " 空格+数字+终止标点" 的脚注编号（终止标点后不能紧跟字母/数字；且不破坏版本号如 1.19, 单位如 250MB, 千位逗号 1,234, 结构 12,abc）
func removeFootnoteNumbers(line string) string {
	if len(line) == 0 {
		return line
	}
	runes := []rune(line)
	var out []rune
	for i := 0; i < len(runes); {
		// 匹配前导空白
		if runes[i] == ' ' || runes[i] == '\t' {
			spStart := i
			j := i
			for j < len(runes) && (runes[j] == ' ' || runes[j] == '\t') {
				j++
			}
			// 数字段
			d := j
			for d < len(runes) && runes[d] >= '0' && runes[d] <= '9' {
				d++
			}
			if d > j && d < len(runes) { // 有数字且未到行尾
				punct := runes[d]
				if isRemovalPunct(punct) {
					// 版本号保护：数字 + '.' + 数字
					if punct == '.' && d+1 < len(runes) && runes[d+1] >= '0' && runes[d+1] <= '9' {
						out = append(out, runes[spStart:d+1]...)
						i = d + 1
						continue
					}
					// 标点后若紧接字母/数字，视为单位/结构一部分（如 250MB / 12,abc / 1,234）
					if d+1 < len(runes) && isLetterOrDigit(runes[d+1]) {
						out = append(out, runes[spStart:d+1]...)
						i = d + 1
						continue
					}
					// 确认移除：跳过空白+数字，仅保留标点
					out = append(out, punct)
					i = d + 1
					continue
				}
				// 非移除标点，保留原始 (空白+数字)
				out = append(out, runes[spStart:d]...)
				i = d
				continue
			}
			// 没有数字，输出空白
			out = append(out, runes[spStart:j]...)
			i = j
			continue
		}
		// 其它字符直接复制
		out = append(out, runes[i])
		i++
	}
	return string(out)
}

// CleanLine 清理单行内容
func (mc *MarkdownCleaner) CleanLine(line string) string {
	// 移除URL引用
	line = mc.urlRefRegex.ReplaceAllString(line, "")

	// 先将 \\** 归一为 * （避免随后 \* -> * 产生 **）
	line = mc.escapedDoubleStarToSingle.ReplaceAllString(line, "*")

	// 修复错误的转义强调符号 \* -> *
	line = mc.escapeStarRegex.ReplaceAllString(line, "$1")

	// 精准移除句末脚注数字（空格+数字+标点）
	line = removeFootnoteNumbers(line)

	// 压缩重复空格（保持单个空格）
	line = mc.multiSpaceRegex.ReplaceAllString(line, " ")

	// 去除行尾多余空格
	line = strings.TrimRight(line, " \t")

	return line
}

// CleanFile 清理整个文件
func (mc *MarkdownCleaner) CleanFile(inputPath, outputPath string) error {
	// 打开输入文件
	inputFile, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("打开输入文件失败: %w", err)
	}
	defer inputFile.Close()

	// 创建输出文件
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("创建输出文件失败: %w", err)
	}
	defer outputFile.Close()

	// 逐行读取和处理
	scanner := bufio.NewScanner(inputFile)
	writer := bufio.NewWriter(outputFile)
	defer writer.Flush()

	lineCount := 0
	urlRefsRemoved := 0
	escapeStarsFixed := 0

	for scanner.Scan() {
		line := scanner.Text()
		lineCount++

		// 记录处理前的状态
		originalLine := line
		urlRefs := mc.urlRefRegex.FindAllString(line, -1)
		escapeStars := mc.escapeStarRegex.FindAllString(line, -1)

		// 清理这一行
		cleanedLine := mc.CleanLine(line)

		// 统计处理的项目
		urlRefsRemoved += len(urlRefs)
		escapeStarsFixed += len(escapeStars)

		// 如果行有变化，输出调试信息
		if originalLine != cleanedLine {
			fmt.Printf("第%d行处理:\n", lineCount)
			if len(urlRefs) > 0 {
				fmt.Printf("  - 移除%d个URL引用: %v\n", len(urlRefs), urlRefs)
			}
			if len(escapeStars) > 0 {
				fmt.Printf("  - 修复%d个转义星号: %v\n", len(escapeStars), escapeStars)
			}
		}

		// 写入清理后的行
		if _, err := writer.WriteString(cleanedLine + "\n"); err != nil {
			return fmt.Errorf("写入文件失败: %w", err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("读取文件失败: %w", err)
	}

	fmt.Printf("\n处理完成!\n")
	fmt.Printf("总行数: %d\n", lineCount)
	fmt.Printf("移除URL引用: %d个\n", urlRefsRemoved)
	fmt.Printf("修复转义星号: %d个\n", escapeStarsFixed)

	return nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("用法: go run rm_url_from_markdown.go <输入文件> [输出文件]")
		fmt.Println("如果不指定输出文件，将在输入文件名后添加'_cleaned'后缀")
		os.Exit(1)
	}

	inputPath := os.Args[1]
	var outputPath string

	if len(os.Args) >= 3 {
		outputPath = os.Args[2]
	} else {
		// 默认输出文件名：在原文件名后添加_cleaned后缀
		if strings.HasSuffix(inputPath, ".md") {
			outputPath = strings.TrimSuffix(inputPath, ".md") + "_cleaned.md"
		} else {
			outputPath = inputPath + "_cleaned"
		}
	}

	fmt.Printf("输入文件: %s\n", inputPath)
	fmt.Printf("输出文件: %s\n", outputPath)
	fmt.Println("开始处理...")

	cleaner := NewMarkdownCleaner()
	if err := cleaner.CleanFile(inputPath, outputPath); err != nil {
		log.Fatalf("处理文件失败: %v", err)
	}

	fmt.Printf("文件已清理完成，保存到: %s\n", outputPath)
}
