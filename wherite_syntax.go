package main

import (
	"strings"
)

type SyntaxToken struct {
	Type     TokenType
	StartPos int
	EndPos   int
	Text     string
}

type TokenType int

const (
	TokenTypeText TokenType = iota
	TokenTypeHeading
	TokenTypeBold
	TokenTypeItalic
	TokenTypeCodeInline
	TokenTypeCodeBlock
	TokenTypeList
	TokenTypeLink
	TokenTypeQuote
)

func HighlightMarkdown(text string) []SyntaxToken {
	var tokens []SyntaxToken

	lines := strings.Split(text, "\n")
	currentPos := 0

	for _, line := range lines {
		lineTokens := highlightLine(line, currentPos)
		tokens = append(tokens, lineTokens...)
		currentPos += len(line) + 1
	}

	return tokens
}

func highlightLine(line string, startPos int) []SyntaxToken {
	var tokens []SyntaxToken

	if isHeadingLine(line) {
		content := strings.TrimLeft(line, "# ")
		tokens = append(tokens, SyntaxToken{
			Type:     TokenTypeHeading,
			StartPos: startPos,
			EndPos:   startPos + len(line),
			Text:     content,
		})
		return tokens
	}

	if isCodeBlockLine(line) {
		tokens = append(tokens, SyntaxToken{
			Type:     TokenTypeCodeBlock,
			StartPos: startPos,
			EndPos:   startPos + len(line),
			Text:     line,
		})
		return tokens
	}

	if isListItemLine(line) {
		tokens = append(tokens, SyntaxToken{
			Type:     TokenTypeList,
			StartPos: startPos,
			EndPos:   startPos + len(line),
			Text:     line,
		})
		return tokens
	}

	if isQuoteLine(line) {
		content := strings.TrimLeft(line, "> ")
		tokens = append(tokens, SyntaxToken{
			Type:     TokenTypeQuote,
			StartPos: startPos,
			EndPos:   startPos + len(line),
			Text:     content,
		})
		return tokens
	}

	inlineTokens := highlightInlineSyntax(line, startPos)
	tokens = append(tokens, inlineTokens...)
	return tokens
}

func highlightInlineSyntax(line string, startPos int) []SyntaxToken {
	var tokens []SyntaxToken
	pos := 0

	for pos < len(line) {
		if pos+2 < len(line) && line[pos:pos+2] == "**" {
			endPos := findClosing(line, pos+2, "**")
			if endPos != -1 {
				if pos > startPos {
					tokens = append(tokens, SyntaxToken{
						Type:     TokenTypeText,
						StartPos: startPos + pos,
						EndPos:   startPos + pos,
						Text:     line[:pos],
					})
				}
				tokens = append(tokens, SyntaxToken{
					Type:     TokenTypeBold,
					StartPos: startPos + pos,
					EndPos:   startPos + endPos,
					Text:     line[pos+2 : endPos],
				})
				line = line[endPos+2:]
				pos = 0
				continue
			}
		}

		if pos+1 < len(line) && line[pos:pos+1] == "*" {
			endPos := findClosing(line, pos+1, "*")
			if endPos != -1 && (pos == 0 || line[pos-1] != '*') {
				if pos > startPos {
					tokens = append(tokens, SyntaxToken{
						Type:     TokenTypeText,
						StartPos: startPos + pos,
						EndPos:   startPos + pos,
						Text:     line[:pos],
					})
				}
				tokens = append(tokens, SyntaxToken{
					Type:     TokenTypeItalic,
					StartPos: startPos + pos,
					EndPos:   startPos + endPos,
					Text:     line[pos+1 : endPos],
				})
				line = line[endPos+1:]
				pos = 0
				continue
			}
		}

		if pos+1 < len(line) && line[pos:pos+1] == "`" {
			endPos := findClosing(line, pos+1, "`")
			if endPos != -1 {
				if pos > startPos {
					tokens = append(tokens, SyntaxToken{
						Type:     TokenTypeText,
						StartPos: startPos + pos,
						EndPos:   startPos + pos,
						Text:     line[:pos],
					})
				}
				tokens = append(tokens, SyntaxToken{
					Type:     TokenTypeCodeInline,
					StartPos: startPos + pos,
					EndPos:   startPos + endPos,
					Text:     line[pos+1 : endPos],
				})
				line = line[endPos+1:]
				pos = 0
				continue
			}
		}

		linkMatch := findLink(line, pos)
		if linkMatch != nil {
			if pos > startPos {
				tokens = append(tokens, SyntaxToken{
					Type:     TokenTypeText,
					StartPos: startPos + pos,
					EndPos:   startPos + pos,
					Text:     line[:pos],
				})
			}
			tokens = append(tokens, SyntaxToken{
				Type:     TokenTypeLink,
				StartPos: startPos + pos,
				EndPos:   startPos + linkMatch.end,
				Text:     linkMatch.text,
			})
			line = line[linkMatch.end:]
			pos = 0
			continue
		}

		pos++
	}

	if len(line) > 0 {
		tokens = append(tokens, SyntaxToken{
			Type:     TokenTypeText,
			StartPos: startPos,
			EndPos:   startPos + len(line),
			Text:     line,
		})
	}

	return tokens
}

func findClosing(text string, start int, marker string) int {
	for i := start; i < len(text); i++ {
		if i+len(marker) <= len(text) && text[i:i+len(marker)] == marker {
			if i > start {
				return i
			}
		}
	}
	return -1
}

type LinkMatch struct {
	text string
	url  string
	end  int
}

func findLink(text string, start int) *LinkMatch {
	if start+1 >= len(text) || text[start] != '[' {
		return nil
	}

	endBracket := strings.Index(text[start:], "]")
	if endBracket == -1 {
		return nil
	}

	linkText := text[start+1 : start+endBracket]
	afterBracket := start + endBracket + 1
	if afterBracket >= len(text) || text[afterBracket] != '(' {
		return nil
	}

	endParen := strings.Index(text[afterBracket:], ")")
	if endParen == -1 {
		return nil
	}

	url := text[afterBracket+1 : afterBracket+endParen]
	return &LinkMatch{
		text: linkText,
		url:  url,
		end:  afterBracket + endParen + 1,
	}
}

func isHeadingLine(line string) bool {
	return len(line) > 0 && line[0] == '#'
}

func countHeadingLevel(line string) int {
	level := 0
	for i := 0; i < len(line) && line[i] == '#'; i++ {
		level++
		if level > 6 {
			break
		}
	}
	return level
}

func isCodeBlockLine(line string) bool {
	return len(line) >= 3 && (line[:3] == "```" || line[:3] == "~~~")
}

func isListItemLine(line string) bool {
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

func isQuoteLine(line string) bool {
	return len(line) > 0 && line[0] == '>'
}

func GetTokenColor(tokenType TokenType) string {
	switch tokenType {
	case TokenTypeHeading:
		return "heading"
	case TokenTypeBold:
		return "bold"
	case TokenTypeItalic:
		return "italic"
	case TokenTypeCodeInline:
		return "code_inline"
	case TokenTypeCodeBlock:
		return "code_block"
	case TokenTypeList:
		return "list"
	case TokenTypeLink:
		return "link"
	case TokenTypeQuote:
		return "quote"
	default:
		return "text"
	}
}
