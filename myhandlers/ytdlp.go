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
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/puzpuzpuz/xsync/v3"
	"go.uber.org/zap"
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
	buf.WriteRune('(')
	buf.WriteString(regexp.QuoteMeta(validUrl[0]))
	for i := 1; i < len(validUrl); i++ {
		buf.WriteRune('|')
		buf.WriteString(regexp.QuoteMeta(validUrl[i]))
	}
	buf.WriteRune(')')
	return regexp.MustCompile(`(?i)` + buf.String() + `/[a-zA-Z\d_%.~:/?#\[\]@!$&'()*+,;=\-]+`)
}

var reDownload = initReDownload()
var reResolution = regexp.MustCompile(`(?i)\b(144|360|480|720|1080)p?\b`)
var errNoURL = errors.New("no downloadable url found")

type dlKey struct {
	Url        string
	Resolution int64
	AudioOnly  bool
}
type DlResult struct {
	uploadFileOnce sync.Once
	cleanup        func()

	wg   sync.WaitGroup
	file string
	err  error
	q.YtDlResult
}

var downloading = xsync.NewMapOf[dlKey, *DlResult]()

func (d *dlKey) downloadToFile() *DlResult {
	const maxConcurrentDownloads = 5
	result, loaded := downloading.LoadOrCompute(*d, func() *DlResult {
		dl := &DlResult{
			YtDlResult: q.YtDlResult{
				Url:        d.Url,
				AudioOnly:  d.AudioOnly,
				Resolution: d.Resolution,
			},
		}
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
	result.cleanup = func() {
		err1 := req.Clean()
		if err1 != nil {
			log.Desugar().Warn("cleanup function error", zap.Error(err1))
		}
	}
	return result
}

func (d *dlKey) findInDb() *DlResult {
	result, err := g.Q.GetYtDlpDbCache(context.Background(), d.Url, d.AudioOnly, d.Resolution)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
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
	const maxLen = 1000
	var buf strings.Builder
	buf.Grow(2048)
	write := func(s string) bool {
		// 如果写入后超过长度，则停止写入
		if buf.Len()+len(s) > maxLen {
			return false
		}
		buf.WriteString(s)
		return true
	}
	if !write(fmt.Sprintf(`<b><a href="%s">%s</a></b>`+"\n",
		html.EscapeString(result.Url),
		html.EscapeString(result.Title),
	)) {
		return buf.String()
	}
	if !write("上传者: " + html.EscapeString(result.Uploader) + "\n") {
		return buf.String()
	}
	if !write(h.MentionUserHtml(user) + "\n") {
		return buf.String()
	}
	if result.Description != "" {
		desc := html.EscapeString(result.Description)
		prefix := "<blockquote expandable>"
		suffix := "</blockquote>"
		if !write(prefix) {
			return buf.String()
		}
		remain := maxLen - buf.Len() - len(suffix)
		if remain > 0 {
			desc = desc[:minUtf8Safe(desc, remain)]
			write(desc)
		}
		write(suffix)
	}
	return buf.String()
}

func minUtf8Safe(s string, n int) int {
	if len(s) <= n {
		return len(s)
	}
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return n
}

func buildYtDlKey(text string, audioOnly bool) (*dlKey, error) {
	url := reDownload.FindString(text)
	if url == "" {
		return nil, fmt.Errorf("没有在聊天 %s 中找到任何可下载的链接", text)
	}
	resolutionPattern := strings.TrimRight(reResolution.FindString(text), "pP")
	resolution := int64(parseIntDefault(resolutionPattern, 720))
	url = "https://" + url
	return &dlKey{
		Url:        url,
		Resolution: resolution,
		AudioOnly:  audioOnly,
	}, nil
}

func buildYtDlKeyFromContext(ctx *ext.Context, audioOnly bool) (*dlKey, error) {
	if ctx == nil || ctx.EffectiveMessage == nil {
		return nil, errNoURL
	}
	text := h.GetAllTextIncludeReply(ctx.EffectiveMessage)
	return buildYtDlKey(text, audioOnly)
}

func sendVideo(bot *gotgbot.Bot, result *DlResult, user *gotgbot.User, msgId, chatId int64) (*gotgbot.Message, error) {
	caption := buildCaption(result, user)
	if result.FileID != "" {
		return bot.SendVideo(chatId, gotgbot.InputFileByID(result.FileID), &gotgbot.SendVideoOpts{
			Caption:         caption,
			ParseMode:       gotgbot.ParseModeHTML,
			ReplyParameters: MakeReplyToMsgID(msgId),
		})
	}
	file, opt := h.PrepareTgVideo(result.file, msgId)
	opt.Caption = caption
	opt.ParseMode = gotgbot.ParseModeHTML
	return bot.SendVideo(chatId, file, opt)
}

func sendAudio(bot *gotgbot.Bot, result *DlResult, user *gotgbot.User, msgId, chatId int64) (*gotgbot.Message, error) {
	caption := buildCaption(result, user)
	if result.FileID != "" {
		return bot.SendAudio(chatId, gotgbot.InputFileByID(result.FileID), &gotgbot.SendAudioOpts{
			Caption:         caption,
			ParseMode:       gotgbot.ParseModeHTML,
			ReplyParameters: MakeReplyToMsgID(msgId),
		})
	}
	return bot.SendAudio(chatId, h.LocalFile(result.file), &gotgbot.SendAudioOpts{
		Caption:         caption,
		ParseMode:       gotgbot.ParseModeHTML,
		ReplyParameters: MakeReplyToMsgID(msgId),
	})
}

func downloadMedia(bot *gotgbot.Bot, key *dlKey, user *gotgbot.User, msgId, chatId int64) (err error) {
	msgOpt := &gotgbot.SendMessageOpts{
		ReplyParameters: MakeReplyToMsgID(msgId),
	}
	if key == nil || key.Url == "" {
		_, err = bot.SendMessage(chatId, "bot没有找到任何有效的链接", msgOpt)
		return errNoURL
	}
	if dbr := key.findInDb(); dbr != nil {
		if key.AudioOnly {
			_, err = sendAudio(bot, dbr, user, msgId, chatId)
		} else {
			_, err = sendVideo(bot, dbr, user, msgId, chatId)
		}
		if err == nil && dbr.FileID != "" {
			_ = g.Q.IncYtDlUploadCount(context.Background(), dbr.FileID)
		}
		return err
	}
	result := key.downloadToFile()
	defer downloading.Delete(*key)
	if result.err != nil {
		_, err = bot.SendMessage(chatId, "下载过程中遇到错误: "+result.err.Error(), msgOpt)
		return err
	}
	result.uploadFileOnce.Do(func() {
		defer result.cleanup()
		var sent *gotgbot.Message
		if key.AudioOnly {
			sent, err = sendAudio(bot, result, user, msgId, chatId)
			if err == nil && sent != nil && sent.Audio != nil {
				result.FileID = sent.Audio.FileId
			}
		} else {
			sent, err = sendVideo(bot, result, user, msgId, chatId)
			if err == nil && sent != nil && sent.Video != nil {
				result.FileID = sent.Video.FileId
			}
		}
		if err != nil {
			result.err = err
			log.Warnf("download send media error: %v", err)
			downloading.Delete(*key)
			return
		}
		if saveErr := result.Save(context.Background(), g.Q); saveErr != nil {
			log.Warnf("save download result to database error: %v", saveErr)
		}
	})
	if result.err != nil {
		_, _ = bot.SendMessage(chatId, "下载失败，遇到错误: "+result.err.Error(), msgOpt)
		return result.err
	}
	return nil
}

func DownloadVideo(bot *gotgbot.Bot, ctx *ext.Context) error {
	reply, done := MakeDebounceMustReply(bot, ctx, time.Second)
	key, err := buildYtDlKeyFromContext(ctx, false)
	if err != nil {
		reply("没有找到视频链接")
		return err
	}
	reply("正在下载，请稍等")
	err = downloadMedia(bot, key, ctx.EffectiveUser, ctx.EffectiveMessage.MessageId, ctx.EffectiveChat.Id)
	if err != nil {
		reply("下载失败")
		return err
	}
	done()
	return nil
}

func DownloadAudio(bot *gotgbot.Bot, ctx *ext.Context) error {
	reply, done := MakeDebounceMustReply(bot, ctx, time.Second)
	key, err := buildYtDlKeyFromContext(ctx, true)
	if err != nil {
		reply("没有找到音频链接")
		return err
	}
	key.Resolution = 0
	reply("正在下载，请稍等")
	err = downloadMedia(bot, key, ctx.EffectiveUser, ctx.EffectiveMessage.MessageId, ctx.EffectiveChat.Id)
	if err != nil {
		reply("下载失败")
		return err
	}
	done()
	return nil
}

func DownloadVideoCallback(bot *gotgbot.Bot, ctx *ext.Context) error {
	answer := MakeAnswerCallback(bot, ctx)
	key, err := buildYtDlKeyFromContext(ctx, false)
	if err != nil {
		answer("没有找到视频链接", true)
		return err
	}
	answer("视频开始下载", false)
	err = downloadMedia(bot, key, &ctx.CallbackQuery.From, ctx.EffectiveMessage.MessageId, ctx.EffectiveMessage.Chat.Id)
	if err != nil {
		answer("下载失败:"+err.Error(), true)
	}
	return err
}

func DownloadInlinedBv(bot *gotgbot.Bot, ctx *ext.Context) error {
	answer := MakeAnswerCallback(bot, ctx)
	uid, err := strconv.ParseInt(ctx.CallbackQuery.Data[len(biliInlineCallbackPrefix):], 16, 64)
	if err != nil {
		answer("没有找到视频链接", true)
		return err
	}
	inlineData, err := g.Q.GetBiliInlineData(context.Background(), uid)
	if err != nil {
		answer("没有找到视频链接，可能是数据库错误", true)
		return err
	}
	key, err := buildYtDlKey(inlineData.Text, false)
	if err != nil {
		answer("没有找到视频链接", true)
		return err
	}
	answer("视频开始下载", false)
	err = downloadMedia(bot, key, &ctx.CallbackQuery.From, inlineData.MsgID, inlineData.ChatID)
	if err != nil {
		answer("下载失败:"+err.Error(), true)
	}
	return err
}
