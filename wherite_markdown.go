package main

import (
	"bytes"
	"fmt"
	"strings"

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

	i := 0
	for i < len(lines) {
		line := lines[i]

		// 处理代码块（多行）
		if isCodeBlock(line) {
			lang := strings.TrimSpace(line[3:])
			i++
			var codeLines []string
			// 循环读取代码内容，直到遇到结束标记
			for i < len(lines) && !isCodeBlockEnd(lines[i]) {
				codeLines = append(codeLines, lines[i])
				i++
			}
			// 跳过结束标记
			if i < len(lines) && isCodeBlockEnd(lines[i]) {
				i++
			}
			if len(codeLines) > 0 {
				content := strings.Join(codeLines, "\n")
				if lang != "" {
					content = lang + "\n" + content
				}
				blocks = append(blocks, MarkdownBlock{
					Type:    BlockTypeCode,
					Content: content,
				})
			} else {
				blocks = append(blocks, MarkdownBlock{
					Type:    BlockTypeCode,
					Content: "",
				})
			}
			continue
		}

		// 处理引用（多行）
		if isQuote(line) {
			var quoteLines []string
			for i < len(lines) && isQuote(lines[i]) {
				// 去掉开头的 > 和可能的空格
				quoteLine := lines[i]
				if len(quoteLine) > 1 && quoteLine[1] == ' ' {
					quoteLine = quoteLine[2:]
				} else if len(quoteLine) > 1 {
					quoteLine = quoteLine[1:]
				} else {
					quoteLine = ""
				}
				quoteLines = append(quoteLines, quoteLine)
				i++
			}
			blocks = append(blocks, MarkdownBlock{
				Type:    BlockTypeQuote,
				Content: strings.Join(quoteLines, "\n"),
			})
			continue
		}

		// 处理其他单行块
		block := parseLine(line)
		if block.Type != BlockTypeEmpty {
			blocks = append(blocks, block)
		}
		i++
	}

	return blocks, nil
}

// MarkdownBlock 表示一个Markdown块
type MarkdownBlock struct {
	Type    BlockType
	Content string
	Level   int
	Inlines []InlineElement
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

// InlineElement 表示行内元素
type InlineElement struct {
	Type InlineType
	Text string
	URL  string // 对于链接类型，存储 URL
}

// InlineType 行内元素类型
type InlineType int

const (
	InlineTypeText InlineType = iota
	InlineTypeBold
	InlineTypeItalic
	InlineTypeStrike
	InlineTypeCode
	InlineTypeLink
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

// isCodeBlock 判断是否为代码块标记行
// 对于开始标记（```lang），允许有语言标识符
// 对于结束标记，必须只有 ``` 或 ~~~（前后都不能有内容）
func isCodeBlock(line string) bool {
	trimmed := strings.TrimSpace(line)
	if len(trimmed) < 3 {
		return false
	}
	if trimmed[:3] != "```" && trimmed[:3] != "~~~" {
		return false
	}
	rest := trimmed[3:]
	trimmedRest := strings.TrimSpace(rest)
	// 如果标记后没有非空格内容，是结束标记
	if len(trimmedRest) == 0 {
		return true
	}
	// 如果标记后有内容且全是字母/数字，可能是语言标识符，也认为是有效的开始标记
	// 但如果包含其他字符（如反引号），则不是有效的标记
	for _, c := range trimmedRest {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '+' || c == '#') {
			return false
		}
	}
	return true
}

// isCodeBlockEnd 判断是否为代码块结束标记（必须是纯 ``` 或 ~~~，不能有语言标识符）
func isCodeBlockEnd(line string) bool {
	trimmed := strings.TrimSpace(line)
	return (len(trimmed) == 3 && (trimmed == "```" || trimmed == "~~~")) ||
		(len(trimmed) > 3 && (trimmed[:3] == "```" || trimmed[:3] == "~~~") && len(strings.TrimSpace(trimmed[3:])) == 0)
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

// ParseInlines 解析行内元素
func ParseInlines(text string) []InlineElement {
	var inlines []InlineElement
	i := 0

	for i < len(text) {
		// 链接 [text](url) 或 ![alt](url)
		if text[i] == '[' || text[i] == '!' {
			newPos, linkElem := parseInlineLink(text, i)
			if linkElem != nil {
				inlines = append(inlines, *linkElem)
				i = newPos
				continue
			}
			// 不是有效链接，当作普通字符，增加 i 避免死循环
			i++
			continue
		}

		// 代码 `...`
		if text[i] == '`' {
			end := findNextByte(text, i+1, '`')
			if end > i {
				inlines = append(inlines, InlineElement{
					Type: InlineTypeCode,
					Text: text[i+1 : end],
				})
				i = end + 1
				continue
			}
			// 没找到结束标记，作为普通文本处理
			i++
			continue
		}

		// 删除线 ~~...~~
		if i+1 < len(text) && text[i] == '~' && text[i+1] == '~' {
			end := findNextStr(text, i+2, "~~")
			if end > i {
				inlines = append(inlines, InlineElement{
					Type: InlineTypeStrike,
					Text: text[i+2 : end],
				})
				i = end + 2
				continue
			}
			// 没找到结束标记，作为普通文本处理
			i += 2
			continue
		}

		// 加粗斜体组合 ***...*** 或 ___...___
		if i+2 < len(text) && ((text[i] == '*' && text[i+1] == '*' && text[i+2] == '*') || (text[i] == '_' && text[i+1] == '_' && text[i+2] == '_')) {
			marker := text[i : i+3]
			end := findNextStr(text, i+3, marker)
			if end > i {
				// 解析内部样式（支持嵌套的加粗和斜体）
				innerText := text[i+3 : end]
				// 内部再解析一次，处理内部的 ** 或 __
				innerInlines := ParseInlines(innerText)
				if len(innerInlines) > 0 {
					// 第一个元素设置为加粗斜体
					innerInlines[0].Type = InlineTypeBold | 0x10 // 使用位标志表示组合
				} else if len(innerText) > 0 {
					innerInlines = []InlineElement{{Type: InlineTypeBold | 0x10, Text: innerText}}
				}
				for _, inner := range innerInlines {
					inlines = append(inlines, inner)
				}
				i = end + 3
				continue
			}
			// 没找到结束标记，作为普通文本处理
			i += 3
			continue
		}

		// 加粗 **...** 或 __...__
		if i+1 < len(text) && ((text[i] == '*' && text[i+1] == '*') || (text[i] == '_' && text[i+1] == '_')) {
			marker := text[i : i+2]
			end := findNextStr(text, i+2, marker)
			if end > i {
				// 解析内部样式（支持嵌套的斜体）
				innerText := text[i+2 : end]
				innerInlines := ParseInlines(innerText)
				if len(innerInlines) > 0 {
					// 第一个元素设置为加粗
					innerInlines[0].Type = InlineTypeBold
				} else if len(innerText) > 0 {
					innerInlines = []InlineElement{{Type: InlineTypeBold, Text: innerText}}
				}
				for _, inner := range innerInlines {
					inlines = append(inlines, inner)
				}
				i = end + 2
				continue
			}
			// 没找到结束标记，作为普通文本处理
			i += 2
			continue
		}

		// 斜体 *...* 或 _..._
		if text[i] == '*' || text[i] == '_' {
			marker := text[i]
			end := findNextByte(text, i+1, marker)
			if end > i {
				// 检查是否是行尾的标点符号
				for end > i+1 && (text[end-1] == '.' || text[end-1] == ',' || text[end-1] == '!' || text[end-1] == '?' || text[end-1] == ':' || text[end-1] == ';') {
					end--
				}
				if end > i {
					inlines = append(inlines, InlineElement{
						Type: InlineTypeItalic,
						Text: text[i+1 : end],
					})
					i = end
					continue
				}
			}
			// 没找到结束标记，作为普通文本处理
			i++
			continue
		}

		// 普通文本
		start := i
		for i < len(text) && text[i] != '*' && text[i] != '_' && text[i] != '`' && text[i] != '~' && text[i] != '[' && text[i] != '!' {
			i++
		}
		if start < i {
			inlines = append(inlines, InlineElement{
				Type: InlineTypeText,
				Text: text[start:i],
			})
		}
	}

	return inlines
}

// parseInlineLink 解析行内链接 [text](url) 或 ![alt](url)
func parseInlineLink(text string, start int) (int, *InlineElement) {
	if start >= len(text) || (text[start] != '[' && text[start] != '!') {
		return start, nil
	}

	isImage := (text[start] == '!')
	searchStart := start + 1

	// 如果是图片，从 `[` 开始找
	if isImage {
		if searchStart >= len(text) || text[searchStart] != '[' {
			return start, nil
		}
		searchStart = start + 2 // 跳过 ![
	}

	// 找到闭合的 ]
	closeBracket := -1
	for i := searchStart; i < len(text); i++ {
		if text[i] == ']' {
			closeBracket = i
			break
		}
	}
	if closeBracket == -1 {
		return start, nil
	}

	// 找到 ( 后面的 )
	if closeBracket+1 >= len(text) || text[closeBracket+1] != '(' {
		return start, nil
	}

	// 查找匹配的 )，考虑 URL 中可能包含括号
	closeParen := -1
	parenDepth := 1 // 已经开始遍历 URL，跳过了 (
	for i := closeBracket + 2; i < len(text); i++ {
		if text[i] == '(' {
			parenDepth++
		} else if text[i] == ')' {
			parenDepth--
			if parenDepth == 0 {
				closeParen = i
				break
			}
		}
	}
	if closeParen == -1 {
		return start, nil
	}

	var linkText, linkURL string
	if isImage {
		// 图片格式 ![alt](url)
		linkText = text[searchStart : closeBracket]
		linkURL = text[closeBracket+2 : closeParen]
	} else {
		// 普通链接格式 [text](url)
		linkText = text[start+1 : closeBracket]
		linkURL = text[closeBracket+2 : closeParen]
	}

	return closeParen + 1, &InlineElement{
		Type: InlineTypeLink,
		Text: linkText,
		URL:  linkURL,
	}
}

// findNextChar 查找下一个指定字符的位置
func findNextChar(text string, start int, char byte) int {
	for i := start; i < len(text); i++ {
		if text[i] == char {
			return i
		}
	}
	return -1
}

// findNextStr 查找下一个指定字符串的位置
func findNextStr(text string, start int, str string) int {
	for i := start; i <= len(text)-len(str); i++ {
		if text[i:i+len(str)] == str {
			return i
		}
	}
	return -1
}

// findNextByte 查找下一个指定字节的位置
func findNextByte(text string, start int, byte byte) int {
	for i := start; i < len(text); i++ {
		if text[i] == byte {
			return i
		}
	}
	return -1
}
