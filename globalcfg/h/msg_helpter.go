package h

import (
	"fmt"
	"html"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
)

func GetAllTextIncludeReply(msg *gotgbot.Message) string {
	buf := strings.Builder{}
	length := len(msg.Text) + len(msg.Caption)
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
	length := len(msg.Text) + len(msg.Caption)
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

func LocalFile(filename string) gotgbot.InputFileOrString {
	if !filepath.IsAbs(filename) {
		var err error
		filename, err = filepath.Abs(filename)
		if err != nil {
			panic(err)
		}
	}
	return gotgbot.InputFileByURL("file://" + url.PathEscape(filename))
}
