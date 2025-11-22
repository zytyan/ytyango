package h

import (
	"fmt"
	"html"
	"regexp"
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

var reCmd = regexp.MustCompile(`^/[a-zA-Z\d_]+(@[a-zA-Z\d_]+)?(\s+|\b|$)`)

// TrimCmd 去除聊天中的命令开头，返回去除命令后的文本
// 例: "/calc 4 + 7-5"  => "4 + 7-5"
func TrimCmd(text string) string {
	if text == "" {
		return ""
	}
	loc := reCmd.FindStringIndex(text)
	if len(loc) == 2 && loc[0] == 0 {
		text = text[loc[1]:]
	}
	return strings.TrimLeft(text, " \t")
}
