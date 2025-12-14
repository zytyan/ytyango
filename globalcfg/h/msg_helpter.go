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
	if entry, ok := diskCache.Get(fileId); ok {
		return entry.path, nil
	}
	result, err, _ := diskSingle.Do(fileId, func() (interface{}, error) {
		if entry, ok := diskCache.Get(fileId); ok {
			return entry, nil
		}
		f, err := bot.GetFile(fileId, nil)
		if err != nil {
			return nil, err
		}
		if isLocalFile(f.FilePath) {
			entry := diskCacheEntry{path: f.FilePath, removeOnEvict: false}
			diskCache.Add(fileId, entry)
			return entry, nil
		}
		target := cachedDiskPath(fileId)
		if info, err := os.Stat(target); err == nil && info.Size() > 0 {
			entry := diskCacheEntry{path: target, removeOnEvict: true}
			diskCache.Add(fileId, entry)
			return entry, nil
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
		tmp, err := os.CreateTemp(filepath.Dir(target), filepath.Base(target)+".tmp")
		if err != nil {
			return nil, err
		}
		if _, err := io.Copy(tmp, resp.Body); err != nil {
			tmp.Close()
			os.Remove(tmp.Name())
			return nil, err
		}
		if err := tmp.Close(); err != nil {
			os.Remove(tmp.Name())
			return nil, err
		}
		_ = os.Remove(target)
		if err := os.Rename(tmp.Name(), target); err != nil {
			os.Remove(tmp.Name())
			return nil, err
		}
		entry := diskCacheEntry{path: target, removeOnEvict: true}
		diskCache.Add(fileId, entry)
		return entry, nil
	})
	if err != nil {
		return "", err
	}
	entry, ok := result.(diskCacheEntry)
	if !ok {
		return "", fmt.Errorf("singleflight: unexpected type %T", result)
	}
	return entry.path, nil
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

type diskCacheEntry struct {
	path          string
	removeOnEvict bool
}

var (
	memCache   *lru.Cache[string, []byte]
	memSingle  singleflight.Group
	diskCache  *lru.Cache[string, diskCacheEntry]
	diskSingle singleflight.Group
)

func DownloadToMemoryCached(bot *gotgbot.Bot, fileId string) (data []byte, err error) {
	var ok bool
	if data, ok = memCache.Get(fileId); ok {
		return data, nil
	}
	r, err, _ := memSingle.Do(fileId, func() (interface{}, error) {
		d, e := DownloadToMemory(bot, fileId)
		if e != nil {
			return nil, e
		}
		memCache.Add(fileId, d)
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
	memCache, err = lru.New[string, []byte](128)
	if err != nil {
		panic(err)
	}
	diskCache, err = lru.NewWithEvict[string, diskCacheEntry](64, func(_ string, entry diskCacheEntry) {
		if entry.removeOnEvict {
			_ = os.Remove(entry.path)
		}
	})
	if err != nil {
		panic(err)
	}
}

func cachedDiskPath(fileId string) string {
	return filepath.Join(os.TempDir(), "bot_file_"+sanitizeFileID(fileId))
}

func sanitizeFileID(fileId string) string {
	var b strings.Builder
	b.Grow(len(fileId))
	for _, r := range fileId {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '_' || r == '-':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	if b.Len() == 0 {
		return "file"
	}
	return b.String()
}
