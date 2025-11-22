package myhandlers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"html"
	g "main/globalcfg"
	"main/globalcfg/h"
	"main/globalcfg/q"
	"main/helpers/ytdlp"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/puzpuzpuz/xsync/v3"
)

func initReDownload() *regexp.Regexp {
	validUrl := []string{
		"bilibili.com",
		"www.bilibili.com",
		"b23.tv",
		"youtu.be",
		"youtube.com",
	}
	buf := strings.Builder{}
	buf.WriteString(regexp.QuoteMeta(validUrl[0]))
	for i := 1; i < len(validUrl); i++ {
		buf.WriteRune('|')
		buf.WriteString(regexp.QuoteMeta(validUrl[i]))
	}
	return regexp.MustCompile(`(?i)` + buf.String() + `/[a-zA-Z\d_%.~:/?#\[\]@!$&'()*+,;=\-]+`)
}

var reDownload = initReDownload()
var reResolution = regexp.MustCompile(`(?i)\b(144|360|480|720|1080)p?\b`)

type dlKey struct {
	Url        string
	Resolution int64
	AudioOnly  bool
}
type DlResult struct {
	uploadFileOnce sync.Once

	wg   sync.WaitGroup
	file string
	err  error
	q.YtDlResult
}

var downloading = xsync.NewMapOf[dlKey, *DlResult]()

func (d *dlKey) downloadToFile() *DlResult {
	const maxConcurrentDownloads = 5
	result, loaded := downloading.LoadOrCompute(*d, func() *DlResult {
		dl := &DlResult{}
		if downloading.Size() >= maxConcurrentDownloads {
			dl.err = fmt.Errorf("当前正在进行的下载过多(%d >= %d)，请稍后再试", downloading.Size(), maxConcurrentDownloads)
			return dl
		}
		dl.wg.Add(1)
		return dl
	})
	if loaded {
		// 其他人无需下载，只需等待第一个下载完，然后等待第一个上传完，就可以直接用现成的file id
		// 这里仅等待第一个下载完
		result.wg.Wait() // 等待，直到下载完成。
		return result
	}
	if result.err != nil {
		return result
	}
	defer result.wg.Done()
	req := ytdlp.Req{
		Url:             d.Url,
		AudioOnly:       d.AudioOnly,
		Resolution:      d.Resolution,
		EmbedMetadata:   true,
		PriorityFormats: []string{"h264", "h265", "av01"},
	}
	resp, err := req.RunWithTimeout(20 * time.Minute)
	if err != nil {
		result.err = err
		downloading.Delete(*d)
		return result
	}
	result.file = resp.FilePath
	result.Title = resp.Info.Title
	result.Uploader = resp.Info.Uploader
	result.Description = resp.Info.Desc
	return result
}

func (d *dlKey) findInDb() *DlResult {
	result, err := g.Q.GetYtDlpDbCache(context.Background(), d.Url, d.AudioOnly, d.Resolution)
	if !errors.Is(err, sql.ErrNoRows) {
		log.Warnf("query database %s err %v", d.Url, err)
	}
	if err != nil {
		return nil
	}
	return &DlResult{
		YtDlResult: result,
	}
}

func buildCaption(result *DlResult, user *gotgbot.User) string {
	buf := strings.Builder{}
	buf.Grow(1024)
	buf.WriteString(`<b>`)
	buf.WriteString(fmt.Sprintf(`<a href="%s">%s</a>`,
		html.EscapeString(result.Url), html.EscapeString(result.Title),
	))
	buf.WriteString("</b>\n")
	buf.WriteString("上传者: ")
	buf.WriteString(html.EscapeString(result.Uploader))
	buf.WriteString("\n")
	buf.WriteString(h.MentionUserHtml(user))
	buf.WriteString("\n")
	if result.Description != "" {
		buf.WriteString(`<blockquote expandable>`)
		buf.WriteString(html.EscapeString(result.Description))
		buf.WriteString(`</blockquote>`)
	}
	/*
		TODO: 如果超过1024字节，tg就不让发Caption，所以这里将来要考虑一个截断到1024字节的东西
		设想: buf:= LimitedBuf{}, buf.Remember() buf.Write() 若写出超限，则回到调用Remember()的时候
	*/
	s := buf.String()
	return s
}

func buildYtDlKey(text string, audioOnly bool) *dlKey {
	url := reDownload.FindString(text)
	resolutionPattern := strings.TrimRight(reResolution.FindString(text), "pP")
	resolution := int64(parseIntDefault(resolutionPattern, 720))

	return &dlKey{
		Url:        url,
		Resolution: resolution,
		AudioOnly:  audioOnly,
	}
}

func downloadVideo(bot *gotgbot.Bot, key *dlKey, user *gotgbot.User, msgId, chatId int64) (err error) {
	msgOpt := &gotgbot.SendMessageOpts{
		ReplyParameters: &gotgbot.ReplyParameters{MessageId: msgId, ChatId: chatId},
	}
	if key.Url == "" {
		_, err = bot.SendMessage(chatId, "bot没有找到任何有效的视频链接", msgOpt)
		return err
	}
	if dbr := key.findInDb(); dbr != nil {
		// 如果数据库里有现成的，就用现成的。
		_, err = bot.SendVideo(chatId, gotgbot.InputFileByID(dbr.FileID), nil)
		return err
	}
	result := key.downloadToFile()
	if result.err != nil {
		_, err = bot.SendMessage(chatId, "下载视频过程中遇到错误: "+result.err.Error(), msgOpt)
		return err
	}
	uploaded := false
	result.uploadFileOnce.Do(func() {
		uploaded = true
		file, opt := h.PrepareTgVideo(result.file, msgId)
		opt.Caption = buildCaption(result, user)
		opt.ParseMode = gotgbot.ParseModeHTML
		var sent *gotgbot.Message
		sent, err = bot.SendVideo(chatId, file, opt)
		if err != nil {
			result.err = err
			log.Warnf("DownloadVideo: send video error: %v", err)
			downloading.Delete(*key)
			return
		}
		result.FileID = sent.Video.FileId
		err = result.Save(context.Background(), g.Q)
		if err != nil {
			log.Warnf("DownloadVideo: save video to database error: %v", err)
		}

	})
	if uploaded {
		return
	}
	if result.err != nil {
		_, _ = bot.SendMessage(chatId, "下载失败，遇到错误: "+result.err.Error(), msgOpt)
		return result.err
	}
	_, err = bot.SendVideo(chatId, gotgbot.InputFileByID(result.FileID), nil)
	return err
}
