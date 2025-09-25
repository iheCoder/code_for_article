package main

import (
	"regexp"
	"strings"
	"testing"
)

// removeSpaceNumbersFromMarkdown 移除 markdown 文本中出现的 " 空格+数字" 形式（一次或多次空白后紧跟一串数字）
func removeSpaceNumbersFromMarkdown(s string) string {
	// 匹配: 一个或多个空白(含换行前的空格)后跟至少一位数字
	// 这里只移除模式整体，不做语义判断，适合去掉从 PDF/网页拷贝遗留的行内编号、脚注编号等
	re := regexp.MustCompile(`\s+\d+`)
	// 将匹配到的内容替换为单个空格，避免词粘连；随后压缩多余空格
	tmp := re.ReplaceAllString(s, " ")
	// 再把出现的多个空格压成一个（不动换行结构）
	multiSpace := regexp.MustCompile(` {2,}`)
	result := multiSpace.ReplaceAllString(tmp, " ")
	// 去掉行尾多余空格
	result = regexp.MustCompile(`[ \t]+\n`).ReplaceAllString(result, "\n")
	// 去掉文本末尾尾随空格
	result = strings.TrimRight(result, " \t")
	return result
}

func TestRemoveSpaceNumbersFromMarkdown(t *testing.T) {
	input := "# 标题 1\n\n这是 一个示例 123 文本，包含 PDF 拷贝遗留 45 的编号。\n第二段  789 里还有 10 若干 数字。结尾 7"
	want := "# 标题\n\n这是 一个示例 文本，包含 PDF 拷贝遗留 的编号。\n第二段 里还有 若干 数字。结尾"

	got := removeSpaceNumbersFromMarkdown(input)
	if got != want {
		t.Fatalf("期望:\n%q\n实际:\n%q", want, got)
	}
}
