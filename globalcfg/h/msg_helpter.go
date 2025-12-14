package h

import (
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
	lru "github.com/hashicorp/golang-lru/v2"
	"golang.org/x/sync/singleflight"
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

func isLocalFile(filename string) bool {
	p := func(prefix string) bool { return strings.HasPrefix(filename, prefix) }
	winPath := len(filename) > 2 && filename[1] == ':'
	return p("/") || p(`\\`) || winPath
}

func DownloadToDisk(bot *gotgbot.Bot, fileId string) (string, error) {
	f, err := bot.GetFile(fileId, nil)
	if err != nil {
		return "", err
	}
	path := f.FilePath
	if isLocalFile(path) {
		return path, nil
	}
	u := f.URL(bot, nil)
	resp, err := http.Get(u)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET %s, bad status: %s", u, resp.Status)
	}
	tmp, err := os.CreateTemp("", "bot_file")
	if err != nil {
		return "", err
	}
	defer tmp.Close()
	if _, err := io.Copy(tmp, resp.Body); err != nil {
		return "", err
	}
	return tmp.Name(), nil
}

func DownloadToMemory(bot *gotgbot.Bot, fileId string) ([]byte, error) {
	f, err := bot.GetFile(fileId, nil)
	if err != nil {
		return nil, err
	}
	path := f.FilePath
	if isLocalFile(path) {
		return os.ReadFile(path)
	}
	u := f.URL(bot, nil)
	resp, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s, bad status: %s", u, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

var cache *lru.Cache[string, []byte]
var single singleflight.Group

func DownloadToMemoryCached(bot *gotgbot.Bot, fileId string) (data []byte, err error) {
	var ok bool
	if data, ok = cache.Get(fileId); ok {
		return data, nil
	}
	r, err, _ := single.Do(fileId, func() (interface{}, error) {
		d, e := DownloadToMemory(bot, fileId)
		if e != nil {
			return nil, e
		}
		cache.Add(fileId, d)
		return d, e
	})
	if err != nil {
		return nil, err
	}
	data, ok = r.([]byte)
	if !ok {
		return nil, fmt.Errorf("singleflight: unexpected type %T", r)
	}
	return data, err
}

func init() {
	var err error
	cache, err = lru.New[string, []byte](128)
	if err != nil {
		panic(err)
	}
}
