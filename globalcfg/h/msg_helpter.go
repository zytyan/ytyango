package h

import (
	"fmt"
	"html"
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
)

func GetAllTextIncludeReply(msg *gotgbot.Message) string {
	buf := strings.Builder{}
	length := 1
	length = len(msg.Text) + len(msg.Caption)
	if reply := msg.ReplyToMessage; reply != nil {
		length += len(reply.Text) + len(msg.Caption) + 2
	}
	buf.Grow(length)
	if msg.Text != "" {
		buf.WriteString(msg.Text)
	} else if msg.Caption != "" {
		buf.WriteByte('\n')
		buf.WriteString(msg.Caption)
	}
	if reply := msg.ReplyToMessage; reply != nil {
		if reply.Text != "" {
			buf.WriteByte('\n')
			buf.WriteString(reply.Text)
		} else if reply.Caption != "" {
			buf.WriteByte('\n')
			buf.WriteString(reply.Caption)
		}
	}
	return buf.String()
}

func GetAllText(msg *gotgbot.Message) string {
	buf := strings.Builder{}
	length := 1
	length = len(msg.Text) + len(msg.Caption)
	buf.Grow(length)
	if msg.Text != "" {
		buf.WriteString(msg.Text)
	} else if msg.Caption != "" {
		buf.WriteByte('\n')
		buf.WriteString(msg.Caption)
	}
	return buf.String()
}

func MentionUserHtml(user *gotgbot.User) string {
	if user == nil {
		return ""
	}
	if user.Username != "" {
		return " @" + user.Username + " "
	}
	name := user.FirstName
	if user.LastName != "" {
		name = name + " " + user.LastName
	}
	return fmt.Sprintf(`<a href="tg://user?id=%d">%s</a>`, user.Id, html.EscapeString(name))
}
