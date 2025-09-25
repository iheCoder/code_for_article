package main

import (
	"testing"
)

func TestMarkdownCleaner_CleanLine(t *testing.T) {
	cleaner := NewMarkdownCleaner()

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "移除简单URL引用",
			input:    "这是一个例子[reallyniceday.com](https://reallyniceday.com/prompt-engineering-four-essential-elements/)。",
			expected: "这是一个例子。",
		},
		{
			name:     "移除复杂URL引用",
			input:    "研究表明[medium.com](https://medium.com/@Micheal-Lanham/is-chain-of-thought-prompting-still-useful-a67e2fa147d8#:~:text=\"Let's think step by step,AI tools and autonomous agents)效果很好。",
			expected: "研究表明效果很好。",
		},
		{
			name:     "修复转义星号",
			input:    "这是\\*重要\\*内容。",
			expected: "这是*重要*内容。",
		},
		{
			name:     "同时处理URL引用和转义星号",
			input:    "**思维链（CoT）\\**技巧指引模型\\**一步步推理**[medium.com](https://example.com)。",
			expected: "**思维链（CoT）*技巧指引模型*一步步推理**。",
		},
		{
			name:     "保留正常的强调符号",
			input:    "这是**粗体**和*斜体*文本。",
			expected: "这是**粗体**和*斜体*文本。",
		},
		{
			name:     "多个URL引用在一行",
			input:    "参考资料[reddit.com](https://reddit.com/example1)和[shelf.io](https://shelf.io/blog/example2)。",
			expected: "参考资料和。",
		},
		{
			name:     "代码块中的URL引用也会被移除",
			input:    "代码示例：`[example.com](https://example.com)`",
			expected: "代码示例：``",
		},
		{
			name:     "空行处理",
			input:    "",
			expected: "",
		},
		{
			name:     "只有普通文本",
			input:    "这是普通的文本内容，没有需要清理的元素。",
			expected: "这是普通的文本内容，没有需要清理的元素。",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := cleaner.CleanLine(tc.input)
			if result != tc.expected {
				t.Errorf("期望: %q, 得到: %q", tc.expected, result)
			}
		})
	}
}

func TestURLRefRegex(t *testing.T) {
	cleaner := NewMarkdownCleaner()

	testCases := []struct {
		input    string
		expected []string
	}{
		{
			input:    "查看[reallyniceday.com](https://reallyniceday.com/prompt-engineering-four-essential-elements/)了解更多。",
			expected: []string{"[reallyniceday.com](https://reallyniceday.com/prompt-engineering-four-essential-elements/)"},
		},
		{
			input:    "多个引用[site1.com](https://site1.com)和[site2.org](https://site2.org)。",
			expected: []string{"[site1.com](https://site1.com)", "[site2.org](https://site2.org)"},
		},
		{
			input:    "没有URL引用的文本。",
			expected: []string{},
		},
	}

	for _, tc := range testCases {
		matches := cleaner.urlRefRegex.FindAllString(tc.input, -1)
		if len(matches) != len(tc.expected) {
			t.Errorf("期望找到%d个匹配，实际找到%d个", len(tc.expected), len(matches))
			continue
		}

		for i, match := range matches {
			if match != tc.expected[i] {
				t.Errorf("第%d个匹配期望: %q, 得到: %q", i, tc.expected[i], match)
			}
		}
	}
}

func TestEscapeStarRegex(t *testing.T) {
	cleaner := NewMarkdownCleaner()

	testCases := []struct {
		input    string
		expected []string
	}{
		{
			input:    "这是\\*转义\\*的星号。",
			expected: []string{"\\*", "\\*"},
		},
		{
			input:    "正常的*星号*不匹配。",
			expected: []string{},
		},
		{
			input:    "混合：\\*转义\\*和*正常*。",
			expected: []string{"\\*", "\\*"},
		},
	}

	for _, tc := range testCases {
		matches := cleaner.escapeStarRegex.FindAllString(tc.input, -1)
		if len(matches) != len(tc.expected) {
			t.Errorf("期望找到%d个匹配，实际找到%d个", len(tc.expected), len(matches))
			continue
		}

		for i, match := range matches {
			if match != tc.expected[i] {
				t.Errorf("第%d个匹配期望: %q, 得到: %q", i, tc.expected[i], match)
			}
		}
	}
}
