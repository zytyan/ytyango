package mdnormalizer

import (
	"strings"
	"unicode"

	"github.com/yuin/goldmark/ast"
)

type mathPartKind int

const (
	literalContent mathPartKind = iota
	mathContent
)

type mathPart struct {
	kind  mathPartKind
	value string
}

func preprocessMarkdown(text string) string {
	if !strings.Contains(text, "*") {
		return text
	}

	var sb strings.Builder
	lines := strings.SplitAfter(text, "\n")
	inFence := false
	var fenceChar rune
	fenceLen := 0

	for _, line := range lines {
		if inFence {
			sb.WriteString(line)
			if isFenceClose(line, fenceChar, fenceLen) {
				inFence = false
				fenceChar = 0
				fenceLen = 0
			}
			continue
		}

		if ch, count, ok := isFenceOpen(line); ok {
			inFence = true
			fenceChar = ch
			fenceLen = count
			sb.WriteString(line)
			continue
		}

		sb.WriteString(injectEscapedSpaceLine(line))
	}

	return sb.String()
}

func isFenceOpen(line string) (rune, int, bool) {
	lead := countFenceIndent(line)
	if lead > 3 {
		return 0, 0, false
	}
	runes := []rune(line[lead:])
	if len(runes) < 3 {
		return 0, 0, false
	}
	ch := runes[0]
	if ch != '`' && ch != '~' {
		return 0, 0, false
	}
	count := 0
	for _, r := range runes {
		if r != ch {
			break
		}
		count++
	}
	if count < 3 {
		return 0, 0, false
	}
	return ch, count, true
}

func isFenceClose(line string, fenceChar rune, fenceLen int) bool {
	lead := countFenceIndent(line)
	if lead > 3 {
		return false
	}
	runes := []rune(line[lead:])
	if len(runes) < fenceLen {
		return false
	}
	count := 0
	for _, r := range runes {
		if r != fenceChar {
			break
		}
		count++
	}
	return count >= fenceLen
}

func countFenceIndent(line string) int {
	count := 0
	for _, r := range line {
		if r != ' ' {
			break
		}
		count++
	}
	return count
}

func injectEscapedSpaceLine(line string) string {
	if !strings.Contains(line, "*") {
		return line
	}
	runes := []rune(line)
	var sb strings.Builder
	inCode := false
	codeTicks := 0

	for i := 0; i < len(runes); {
		r := runes[i]
		if r == '`' {
			j := i
			for j < len(runes) && runes[j] == '`' {
				j++
			}
			runLen := j - i
			sb.WriteString(string(runes[i:j]))
			if inCode {
				if runLen == codeTicks {
					inCode = false
					codeTicks = 0
				}
			} else {
				inCode = true
				codeTicks = runLen
			}
			i = j
			continue
		}

		if !inCode && r == '*' {
			j := i
			for j < len(runes) && runes[j] == '*' {
				j++
			}
			var prev rune
			var next rune
			if i > 0 {
				prev = runes[i-1]
			}
			if j < len(runes) {
				next = runes[j]
			}
			if i > 0 && j < len(runes) && shouldInsertEscapedSpace(prev, next) {
				sb.WriteString(`\ `)
			}
			sb.WriteString(string(runes[i:j]))
			i = j
			continue
		}

		sb.WriteRune(r)
		i++
	}

	return sb.String()
}

func shouldInsertEscapedSpace(prev, next rune) bool {
	return isCJK(prev) && isOpeningPunct(next)
}

func isOpeningPunct(r rune) bool {
	switch r {
	case '“', '‘', '「', '『', '《', '〈', '【', '（', '〔', '〖', '〝':
		return true
	default:
		return false
	}
}

func isCJK(r rune) bool {
	return unicode.Is(unicode.Han, r) ||
		unicode.Is(unicode.Hiragana, r) ||
		unicode.Is(unicode.Katakana, r) ||
		unicode.Is(unicode.Hangul, r)
}

func stripEscapedSpace(text string) string {
	if !strings.Contains(text, `\ `) {
		return text
	}
	return strings.ReplaceAll(text, `\ `, "")
}

func splitMath(text string) []mathPart {
	var parts []mathPart
	var current strings.Builder

	flushLiteral := func() {
		if current.Len() == 0 {
			return
		}
		parts = append(parts, mathPart{kind: literalContent, value: current.String()})
		current.Reset()
	}

	for i := 0; i < len(text); i++ {
		if text[i] != '$' {
			current.WriteByte(text[i])
			continue
		}

		delim := 1
		if i+1 < len(text) && text[i+1] == '$' {
			delim = 2
		}

		closing := strings.Index(text[i+delim:], strings.Repeat("$", delim))
		if closing < 0 {
			current.WriteByte(text[i])
			continue
		}

		flushLiteral()

		mathStart := i + delim
		mathEnd := mathStart + closing
		parts = append(parts, mathPart{kind: mathContent, value: text[mathStart:mathEnd]})
		i = mathEnd + delim - 1
	}

	flushLiteral()

	if len(parts) == 0 {
		return []mathPart{{kind: literalContent, value: text}}
	}

	return parts
}

func plainText(node ast.Node, source []byte) string {
	var sb strings.Builder

	var visit func(ast.Node)
	visit = func(n ast.Node) {
		switch v := n.(type) {
		case *ast.Text:
			sb.Write(v.Segment.Value(source))
			if v.SoftLineBreak() || v.HardLineBreak() {
				sb.WriteByte('\n')
			}
		case *ast.CodeSpan:
			sb.Write(v.Text(source))
		case *ast.AutoLink:
			sb.Write(v.URL(source))
		default:
			for child := n.FirstChild(); child != nil; child = child.NextSibling() {
				visit(child)
			}
		}
	}

	visit(node)
	return sb.String()
}

func escapeTelegramText(text string) string {
	var sb strings.Builder
	for _, r := range text {
		switch r {
		case '\\', '_', '*', '[', ']', '(', ')', '~', '`', '>', '#', '+', '-', '=', '|', '{', '}', '.', '!':
			sb.WriteRune('\\')
			sb.WriteRune(r)
		default:
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

func escapeCodeText(text string) string {
	var sb strings.Builder
	for _, r := range text {
		if r == '\\' || r == '`' {
			sb.WriteRune('\\')
		}
		sb.WriteRune(r)
	}
	return sb.String()
}

func utf16Length(text string) int64 {
	var count int64
	for _, r := range text {
		if r <= unicode.MaxRune && r > 0xFFFF {
			count += 2
			continue
		}
		count++
	}
	return count
}
