package mdnormalizer

import (
	"regexp"
	"strings"

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

var reOpenEscape = regexp.MustCompile(`([^\s\p{P}])(\*\*?|__?|~|\|\|)([\p{Pi}\p{Pf}\p{Ps}\p{Pe}])`)
var reCloseEscape = regexp.MustCompile(`([\p{Pf}\p{Pe}])(\*\*?|__?|~|\|\|)([^\s\p{P}])`)

func preprocessMarkdown(text string) string {
	if !strings.Contains(text, "*") {
		return text
	}
	out := reOpenEscape.ReplaceAllString(text, `$1\ $2$3`)
	out = reCloseEscape.ReplaceAllString(out, `$1$2\ $3`)
	return out
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

func utf16Length(text string) int64 {
	var count int64
	for _, r := range text {
		if r > 0xFFFF {
			count += 2
			continue
		}
		count++
	}
	return count
}
