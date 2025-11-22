package h

import (
	"regexp"
	"strings"
)

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

