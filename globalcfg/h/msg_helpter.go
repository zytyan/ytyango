package h

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"main/helpers/lrusf"

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

// downloadToWriter streams a telegram file into the provided writer.
func downloadToWriter(bot *gotgbot.Bot, fileId string, w io.Writer) error {
	f, err := bot.GetFile(fileId, nil)
	if err != nil {
		return err
	}
	return downloadWithFile(bot, f, w)
}

func downloadWithFile(bot *gotgbot.Bot, f *gotgbot.File, w io.Writer) error {
	if isLocalFile(f.FilePath) {
		fp, err := os.Open(f.FilePath)
		if err != nil {
			return err
		}
		defer fp.Close()
		_, err = io.Copy(w, fp)
		return err
	}
	u := f.URL(bot, nil)
	resp, err := http.Get(u)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s, bad status: %s", u, resp.Status)
	}
	_, err = io.Copy(w, resp.Body)
	return err
}

func DownloadToDisk(bot *gotgbot.Bot, fileId string) (string, error) {
	entry, err := diskCache.Get(fileId, func() (diskCacheEntry, error) {
		target := cachedDiskPath(fileId)
		if info, err := os.Stat(target); err == nil && info.Size() > 0 {
			return diskCacheEntry{path: target, removeOnEvict: true}, nil
		}

		f, err := bot.GetFile(fileId, nil)
		if err != nil {
			var zero diskCacheEntry
			return zero, err
		}
		if isLocalFile(f.FilePath) {
			return diskCacheEntry{path: f.FilePath, removeOnEvict: false}, nil
		}

		if data, ok := memCache.TryGet(fileId); ok {
			if err := writeBytesToPath(target, data); err == nil {
				return diskCacheEntry{path: target, removeOnEvict: true}, nil
			}
		}

		data, err := DownloadToMemoryCached(bot, fileId)
		if err == nil {
			if err := writeBytesToPath(target, data); err != nil {
				var zero diskCacheEntry
				return zero, err
			}
			return diskCacheEntry{path: target, removeOnEvict: true}, nil
		}

		if err := writeToPath(target, func(w io.Writer) error {
			return downloadWithFile(bot, f, w)
		}); err != nil {
			var zero diskCacheEntry
			return zero, err
		}

		return diskCacheEntry{path: target, removeOnEvict: true}, nil
	})
	if err != nil {
		return "", err
	}
	return entry.path, nil
}

func DownloadToMemory(bot *gotgbot.Bot, fileId string) ([]byte, error) {
	var buf bytes.Buffer
	if err := downloadToWriter(bot, fileId, &buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type diskCacheEntry struct {
	path          string
	removeOnEvict bool
}

var (
	memCache  *lrusf.Cache[string, []byte]
	diskCache *lrusf.Cache[string, diskCacheEntry]
)

func DownloadToMemoryCached(bot *gotgbot.Bot, fileId string) (data []byte, err error) {
	return memCache.Get(fileId, func() ([]byte, error) {
		return DownloadToMemory(bot, fileId)
	})
}

func init() {
	memCache = lrusf.NewCache[string, []byte](128, idToKey, nil)
	diskCache = lrusf.NewCache[string, diskCacheEntry](2048, idToKey, func(_ string, entry diskCacheEntry) {
		if entry.removeOnEvict {
			_ = os.Remove(entry.path)
		}
	})
}

func writeBytesToPath(target string, data []byte) error {
	return writeToPath(target, func(w io.Writer) error {
		_, err := w.Write(data)
		return err
	})
}

func writeToPath(target string, fill func(io.Writer) error) error {
	tmp, err := os.CreateTemp(filepath.Dir(target), filepath.Base(target)+".tmp")
	if err != nil {
		return err
	}
	if err := fill(tmp); err != nil {
		tmp.Close()
		_ = os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	_ = os.Remove(target)
	if err := os.Rename(tmp.Name(), target); err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	return nil
}

func cachedDiskPath(fileId string) string {
	return filepath.Join(os.TempDir(), "bot_file_"+sanitizeFileID(fileId))
}

func idToKey(k string) string { return k }

func sanitizeFileID(fileId string) string {
	var b strings.Builder
	b.Grow(len(fileId))
	if len(fileId) > 200 {
		hash := sha256.Sum224([]byte(fileId))
		return "sha256_" + hex.EncodeToString(hash[:])
	}
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

func WithChatAction(bot *gotgbot.Bot, action string, chatId, topicId int64, useTopic bool) func() {
	ctx, cancel := context.WithCancel(context.Background())
	ticker := time.NewTicker(time.Second * 3)
	go func() {
		opt := &gotgbot.SendChatActionOpts{}
		if useTopic {
			opt.MessageThreadId = topicId
		}
		_, _ = bot.SendChatAction(chatId, action, opt)
		for {
			select {
			case <-ticker.C:
				_, _ = bot.SendChatAction(chatId, action, opt)
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
	return cancel
}
