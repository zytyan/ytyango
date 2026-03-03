package ent2md

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"unicode/utf16"

	"github.com/PaulSonOfLars/gotgbot/v2"
)

type entityWrapper struct {
	entity gotgbot.MessageEntity
	pos    int64
	isEnd  bool
}

func prepareEntities(raw []gotgbot.MessageEntity) []entityWrapper {
	wrapped := make([]entityWrapper, 0, len(raw)*2)
	for _, entity := range raw {
		wrapped = append(wrapped,
			entityWrapper{entity, entity.Offset, false},
			entityWrapper{entity, entity.Offset + entity.Length, true},
		)
	}
	slices.SortFunc(wrapped, func(a, b entityWrapper) int {
		return cmp.Compare(a.pos, b.pos)
	})
	return wrapped
}

type mdConv struct {
	buf          strings.Builder
	last         int64
	str          []uint16
	disableMark  bool
	inBlockQuote bool
}

func TgMsgTextToMarkdown(msg *gotgbot.Message) string {
	text := msg.GetText()
	if text == "" {
		return ""
	}
	if len(msg.GetEntities()) == 0 {
		return text
	}
	conv := &mdConv{
		buf:  strings.Builder{},
		last: 0,
		str:  utf16.Encode([]rune(text)),
	}
	entities := prepareEntities(msg.GetEntities())
	for _, entity := range entities {
		conv.addEntity(entity)
	}
	conv.addPlainText(int64(len(conv.str)))
	return conv.buf.String()
}

var mdReplacer = strings.NewReplacer(
	`-`, `\-`,
	`_`, `\_`,
	`*`, `\*`,
)

func (c *mdConv) addPlainText(end int64) {
	s := string(utf16.Decode(c.str[c.last:end]))
	if !c.disableMark {
		s = mdReplacer.Replace(s)
	}
	if c.inBlockQuote {
		s = strings.ReplaceAll(s, "\n", "\n> ")
	}
	c.buf.WriteString(s)
	c.last = end
}
func (c *mdConv) addEntity(entity entityWrapper) {
	c.addPlainText(entity.pos)
	switch entity.entity.Type {
	case "bold":
		c.buf.WriteString("**")
	case "italic":
		c.buf.WriteString("_")
	case "underline":
		c.buf.WriteString("__")
	case "strikethrough":
		c.buf.WriteString("~")
	case "spoiler":
		c.buf.WriteString("||")
	case "code":
		c.buf.WriteString("`")
	case "pre":
		if entity.isEnd {
			c.buf.WriteString("\n")
		}
		c.buf.WriteString("```")
		if !entity.isEnd {
			c.buf.WriteString(entity.entity.Language)
			c.buf.WriteString("\n")
		}
	case "blockquote", "expandable_blockquote":
		c.inBlockQuote = !entity.isEnd
	case "text_link", "url":
		if entity.isEnd {
			u := entity.entity.Url
			_, _ = fmt.Fprintf(&c.buf, "](%s)", u)
		} else {
			c.buf.WriteString("[")
		}
	case "text_mention", "mention", "hashtag", "cashtag", "bot_command", "email", "phone_number":
		// 这个不需要做
		break
	case "custom_emoji":
		// 这个也不需要解决呢
		break
	case "date_time":
		break
	default:
	}
}
