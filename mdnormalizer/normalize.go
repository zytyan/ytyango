package mdnormalizer

import (
	"errors"
	"fmt"
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// NormalizedMessage is the output of Normalize.
type NormalizedMessage struct {
	Text     string
	Entities []gotgbot.MessageEntity
	Warnings []string
}

// Options controls normalization behavior.
type Options struct {
	Strict          bool
	CollectWarnings bool
}

// Option mutates Options.
type Option func(*Options)

// WithStrict toggles strict mode: unsupported nodes return error.
func WithStrict(strict bool) Option {
	return func(o *Options) {
		o.Strict = strict
	}
}

// WithWarnings toggles warning collection for fallback paths.
func WithWarnings(enabled bool) Option {
	return func(o *Options) {
		o.CollectWarnings = enabled
	}
}

// Normalize converts Markdown into Telegram text and message entities.
func Normalize(markdown string, opts ...Option) (*NormalizedMessage, error) {
	options := defaultOptions()
	for _, opt := range opts {
		opt(&options)
	}

	root := defaultMarkdownParser().Parser().Parse(text.NewReader([]byte(markdown)))
	b := &builder{
		source:   []byte(markdown),
		options:  options,
		offset:   0,
		warnings: make([]string, 0),
	}

	if err := b.walkBlocks(root); err != nil {
		return nil, err
	}

	return &NormalizedMessage{
		Text:     b.sb.String(),
		Entities: b.entities,
		Warnings: b.warnings,
	}, nil
}

func defaultOptions() Options {
	return Options{
		CollectWarnings: true,
	}
}

var mdParser goldmark.Markdown

func defaultMarkdownParser() goldmark.Markdown {
	if mdParser != nil {
		return mdParser
	}

	mdParser = goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Strikethrough,
			extension.Linkify,
			extension.Table,
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
	)

	return mdParser
}

type escapeMode int

const (
	escapeNone escapeMode = iota
	escapeText
	escapeCode
)

type builder struct {
	source   []byte
	sb       strings.Builder
	entities []gotgbot.MessageEntity
	warnings []string
	options  Options
	offset   int64 // UTF-16 code units
}

func (b *builder) walkBlocks(node ast.Node) error {
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		switch n := child.(type) {
		case *ast.Paragraph:
			b.ensureBlockSeparation()
			if err := b.walkInline(n); err != nil {
				return err
			}
			b.appendNewline()
		case *ast.Heading:
			b.ensureBlockSeparation()
			if err := b.walkInline(n); err != nil {
				return err
			}
			b.appendNewline()
		case *ast.Blockquote:
			b.ensureBlockSeparation()
			start := b.offset
			if err := b.walkBlocks(n); err != nil {
				return err
			}
			b.addEntity("blockquote", start, b.offset-start, "")
			b.appendNewline()
		case *ast.FencedCodeBlock:
			b.ensureBlockSeparation()
			b.handleCodeBlock(n)
			b.appendNewline()
		case *ast.CodeBlock:
			b.ensureBlockSeparation()
			b.handleIndentedCodeBlock(n)
			b.appendNewline()
		case *ast.List:
			b.ensureBlockSeparation()
			if err := b.handleList(n); err != nil {
				return err
			}
			b.appendNewline()
		case *extast.Table:
			b.ensureBlockSeparation()
			if err := b.handleTable(n); err != nil {
				return err
			}
			b.appendNewline()
		case *ast.HTMLBlock:
			b.ensureBlockSeparation()
			if err := b.handleFallbackBlock(n); err != nil {
				return err
			}
			b.appendNewline()
		case *ast.ThematicBreak:
			b.ensureBlockSeparation()
		default:
			if child.HasChildren() {
				if err := b.walkBlocks(child); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (b *builder) walkInline(node ast.Node) error {
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		switch n := child.(type) {
		case *ast.Text:
			if err := b.handleText(n); err != nil {
				return err
			}
		case *ast.Emphasis:
			entityType := "italic"
			if n.Level == 2 {
				entityType = "bold"
			}
			start := b.offset
			if err := b.walkInline(n); err != nil {
				return err
			}
			b.addEntity(entityType, start, b.offset-start, "")
		case *extast.Strikethrough:
			start := b.offset
			if err := b.walkInline(n); err != nil {
				return err
			}
			b.addEntity("strikethrough", start, b.offset-start, "")
		case *ast.CodeSpan:
			start := b.offset
			b.appendText(string(n.Text(b.source)), escapeCode)
			b.addEntity("code", start, b.offset-start, "")
		case *ast.Link:
			if err := b.handleLink(n); err != nil {
				return err
			}
		case *ast.AutoLink:
			if err := b.handleAutoLink(n); err != nil {
				return err
			}
		case *ast.Image:
			if err := b.handleImage(n); err != nil {
				return err
			}
		default:
			if child.HasChildren() {
				if err := b.walkInline(child); err != nil {
					return err
				}
			} else if literal := child.Text(b.source); len(literal) > 0 {
				b.appendText(string(literal), escapeText)
			}
		}
	}

	return nil
}

func (b *builder) handleText(node *ast.Text) error {
	segment := node.Segment.Value(b.source)
	parts := splitMath(string(segment))
	for _, part := range parts {
		if part.kind == mathContent {
			if err := b.fallback("math converted to inline code"); err != nil {
				return err
			}
			start := b.offset
			b.appendText(part.value, escapeCode)
			b.addEntity("code", start, b.offset-start, "")
			continue
		}
		b.appendText(part.value, escapeText)
	}

	if node.HardLineBreak() {
		b.appendText("\n", escapeNone)
	}
	if node.SoftLineBreak() {
		b.appendText("\n", escapeNone)
	}

	return nil
}

func (b *builder) handleLink(node *ast.Link) error {
	label := strings.TrimSpace(plainText(node, b.source))
	if label == "" {
		label = string(node.Destination)
	}

	start := b.offset
	b.appendText(label, escapeText)
	b.addEntity("text_link", start, b.offset-start, string(node.Destination))
	return nil
}

func (b *builder) handleAutoLink(node *ast.AutoLink) error {
	start := b.offset
	text := string(node.URL(b.source))
	b.appendText(text, escapeText)
	b.addEntity("text_link", start, b.offset-start, text)
	return nil
}

func (b *builder) handleImage(node *ast.Image) error {
	label := strings.TrimSpace(plainText(node, b.source))
	if label == "" {
		label = "image"
	}

	start := b.offset
	b.appendText(label, escapeText)
	b.addEntity("text_link", start, b.offset-start, string(node.Destination))
	return b.fallback("image converted to link")
}

func (b *builder) handleCodeBlock(node *ast.FencedCodeBlock) {
	start := b.offset
	b.appendText(string(node.Text(b.source)), escapeCode)
	lang := string(node.Language(b.source))
	b.addEntity("pre", start, b.offset-start, lang)
}

func (b *builder) handleIndentedCodeBlock(node *ast.CodeBlock) {
	start := b.offset
	b.appendText(string(node.Text(b.source)), escapeCode)
	b.addEntity("pre", start, b.offset-start, "")
}

func (b *builder) handleList(list *ast.List) error {
	var lines []string
	index := int(list.Start)
	for item := list.FirstChild(); item != nil; item = item.NextSibling() {
		prefix := "•"
		if list.IsOrdered() {
			prefix = fmt.Sprintf("%d.", index)
			index++
		}

		content := strings.TrimSpace(plainText(item, b.source))
		line := strings.TrimSpace(fmt.Sprintf("%s %s", prefix, content))
		lines = append(lines, line)
	}

	start := b.offset
	b.appendText(strings.Join(lines, "\n"), escapeCode)
	b.addEntity("pre", start, b.offset-start, "")
	return b.fallback("list converted to code block")
}

func (b *builder) handleTable(table *extast.Table) error {
	var lines []string
	for row := table.FirstChild(); row != nil; row = row.NextSibling() {
		tr, ok := row.(*extast.TableRow)
		if !ok {
			continue
		}

		var cells []string
		for cell := tr.FirstChild(); cell != nil; cell = cell.NextSibling() {
			tc, ok := cell.(*extast.TableCell)
			if !ok {
				continue
			}
			cells = append(cells, strings.TrimSpace(plainText(tc, b.source)))
		}

		lines = append(lines, strings.Join(cells, " | "))
	}

	start := b.offset
	b.appendText(strings.Join(lines, "\n"), escapeCode)
	b.addEntity("pre", start, b.offset-start, "")
	return b.fallback("table converted to code block")
}

func (b *builder) handleFallbackBlock(node ast.Node) error {
	start := b.offset
	content := strings.TrimSpace(plainText(node, b.source))
	if content == "" {
		if provider, ok := node.(interface{ Lines() *text.Segments }); ok {
			content = strings.TrimSpace(string(provider.Lines().Value(b.source)))
		}
	}
	if content == "" {
		content = node.Kind().String()
	}
	b.appendText(content, escapeCode)
	b.addEntity("pre", start, b.offset-start, "")
	return b.fallback(fmt.Sprintf("unsupported block %T converted to code block", node))
}

func (b *builder) ensureBlockSeparation() {
	if b.sb.Len() == 0 {
		return
	}
	if strings.HasSuffix(b.sb.String(), "\n") {
		return
	}
	b.appendNewline()
}

func (b *builder) appendNewline() {
	b.appendText("\n", escapeNone)
}

func (b *builder) appendText(text string, mode escapeMode) {
	if text == "" {
		return
	}

	switch mode {
	case escapeText:
		text = escapeTelegramText(text)
	case escapeCode:
		text = escapeCodeText(text)
	}

	b.sb.WriteString(text)
	b.offset += utf16Length(text)
}

func (b *builder) addEntity(entityType string, offset int64, length int64, languageOrURL string) {
	if length == 0 {
		return
	}

	ent := gotgbot.MessageEntity{
		Type:   entityType,
		Offset: offset,
		Length: length,
	}

	switch entityType {
	case "pre":
		ent.Language = languageOrURL
	case "text_link":
		ent.Url = languageOrURL
	}

	b.entities = append(b.entities, ent)
}

func (b *builder) fallback(msg string) error {
	if b.options.Strict {
		return errors.New(msg)
	}
	b.warn(msg)
	return nil
}

func (b *builder) warn(msg string) {
	if !b.options.CollectWarnings {
		return
	}
	b.warnings = append(b.warnings, msg)
}
