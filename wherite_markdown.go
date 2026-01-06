package main

import (
	"bytes"
	"fmt"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

var mdParser goldmark.Markdown

func init() {
	mdParser = goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Table,
			extension.Strikethrough,
			extension.TaskList,
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
		),
	)
}

// MarkdownToHTML 将Markdown文本转换为HTML
func MarkdownToHTML(markdown string) (string, error) {
	if markdown == "" {
		return "", nil
	}

	var buf bytes.Buffer
	err := mdParser.Convert([]byte(markdown), &buf)
	if err != nil {
		return "", fmt.Errorf("Markdown转换失败: %w", err)
	}

	return buf.String(), nil
}

// ParseMarkdownBlock 解析Markdown块元素
func ParseMarkdownBlock(markdown string) ([]MarkdownBlock, error) {
	var blocks []MarkdownBlock
	lines := splitLines(markdown)

	for _, line := range lines {
		block := parseLine(line)
		if block.Type != BlockTypeEmpty {
			blocks = append(blocks, block)
		}
	}

	return blocks, nil
}

// MarkdownBlock 表示一个Markdown块
type MarkdownBlock struct {
	Type    BlockType
	Content string
	Level   int
}

// BlockType 块类型
type BlockType int

const (
	BlockTypeEmpty BlockType = iota
	BlockTypeHeading
	BlockTypeParagraph
	BlockTypeCode
	BlockTypeList
	BlockTypeQuote
	BlockTypeHorizontalRule
)

// parseLine 解析单行Markdown
func parseLine(line string) MarkdownBlock {
	if line == "" {
		return MarkdownBlock{Type: BlockTypeEmpty}
	}

	if isHeading(line) {
		level, content := parseHeading(line)
		return MarkdownBlock{
			Type:    BlockTypeHeading,
			Content: content,
			Level:   level,
		}
	}

	if isCodeBlock(line) {
		return MarkdownBlock{
			Type:    BlockTypeCode,
			Content: line,
		}
	}

	if isListItem(line) {
		return MarkdownBlock{
			Type:    BlockTypeList,
			Content: line,
		}
	}

	if isQuote(line) {
		return MarkdownBlock{
			Type:    BlockTypeQuote,
			Content: line[1:],
		}
	}

	if isHorizontalRule(line) {
		return MarkdownBlock{
			Type: BlockTypeHorizontalRule,
		}
	}

	return MarkdownBlock{
		Type:    BlockTypeParagraph,
		Content: line,
	}
}

// isHeading 判断是否为标题
func isHeading(line string) bool {
	if len(line) == 0 {
		return false
	}
	return line[0] == '#'
}

// parseHeading 解析标题
func parseHeading(line string) (int, string) {
	level := 0
	for i := 0; i < len(line) && line[i] == '#'; i++ {
		level++
		if level > 6 {
			break
		}
	}

	content := ""
	if level < len(line) && line[level] == ' ' {
		content = line[level+1:]
	} else if level < len(line) {
		content = line[level:]
	}

	return level, content
}

// isCodeBlock 判断是否为代码块
func isCodeBlock(line string) bool {
	return len(line) >= 3 && (line[:3] == "```" || line[:3] == "~~~")
}

// isListItem 判断是否为列表项
func isListItem(line string) bool {
	if len(line) == 0 {
		return false
	}

	if line[0] == '-' || line[0] == '*' {
		if len(line) > 1 && line[1] == ' ' {
			return true
		}
	}

	if len(line) >= 2 && line[0] >= '0' && line[0] <= '9' && line[1] == '.' {
		if len(line) > 2 && line[2] == ' ' {
			return true
		}
	}

	return false
}

// isQuote 判断是否为引用
func isQuote(line string) bool {
	return len(line) > 0 && line[0] == '>'
}

// isHorizontalRule 判断是否为水平线
func isHorizontalRule(line string) bool {
	if len(line) < 3 {
		return false
	}

	char := line[0]
	if char != '-' && char != '*' && char != '_' {
		return false
	}

	for i := 1; i < len(line); i++ {
		if line[i] != char && line[i] != ' ' {
			return false
		}
	}

	return true
}

// splitLines 分割文本为行
func splitLines(text string) []string {
	var lines []string
	start := 0

	for i := 0; i < len(text); i++ {
		if text[i] == '\n' {
			lines = append(lines, text[start:i])
			start = i + 1
		}
	}

	if start < len(text) {
		lines = append(lines, text[start:])
	}

	return lines
}
