package myhandlers

import (
	"context"
	"errors"
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/google/uuid"
	"github.com/puzpuzpuz/xsync/v3"
	"gorm.io/gorm"
	"main/globalcfg"
	"main/helpers/ytdlp"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const ytDlpUuidHeader = "X-Yt-Dlp-Uuid"

var (
	reUrl         = regexp.MustCompile(`(https?://)?(\w+\.)+(\w+)/[a-zA-Z0-9_\-?&=%.+/]+`)
	reYtDlpHosts  = regexp.MustCompile(`(youtu\.be|youtube\.com|b23\.tv|bilibili\.com|twitter\.com)`)
	reResolutions = regexp.MustCompile(`(\b|^)(144|360|480|720|1080)([pP]?)(\b|$)`)
)
var (
	ErrNoUrl = errors.New("no url")
)
var ytDlpMap = xsync.NewMapOf[string, *YtDlProgress]()
var downloading = xsync.NewMapOf[YtDlKey, string]()
var recentDownloaded = xsync.NewMapOf[YtDlDebounceKey, struct{}]()

type YtDlKey struct {
	Url        string `gorm:"index"`
	AudioOnly  bool
	Resolution int
}

type YtDlDebounceKey struct {
	YtDlKey
	ChatId int64
}

type YtDlDb struct {
	YtDlKey
	FileId string
}

type YtDlProgress struct {
	wg   *sync.WaitGroup
	Ctx  context.Context
	Path string
}

func (y *YtDlKey) TakeDb() (*YtDlDb, bool) {
	db := &YtDlDb{}
	err := globalcfg.GetDb().Where(y).Take(db).Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		log.Infof("yt-dlp db not found url:%s", y.Url)
		return nil, false
	} else if err != nil {
		log.Warnf("yt-dlp db not found and get an error: %s", err)
		return nil, false
	}
	if db.FileId != "" {
		return db, true
	}
	return nil, false
}

func (y *YtDlKey) makeConfig(ctx context.Context) (*ytdlp.Config, *YtDlProgress, string) {
	uid := uuid.NewString()
	postExec := fmt.Sprintf(`curl -H '%s: %s' -X POST --data-urlencode filepath=%%(filepath,_filename|)q http://localhost:4019/yt-dlp`,
		ytDlpUuidHeader, uid)
	prog := &YtDlProgress{
		wg:  &sync.WaitGroup{},
		Ctx: ctx,
	}
	log.Debugf("url:%s, uid:%s, post exec: %s", y.Url, uid, postExec)
	return &ytdlp.Config{
		Url:             y.Url,
		AudioOnly:       y.AudioOnly,
		Resolution:      y.Resolution,
		EmbedMetadata:   true,
		PostExec:        postExec,
		PriorityFormats: []string{"h264", "h265", "av01"},
	}, prog, uid
}

func (y *YtDlKey) Download() (string, func(), error) {
	clean := func() {}
	if uid, loaded := downloading.LoadOrStore(*y, ""); loaded {
		prog, ok := ytDlpMap.Load(uid)
		if !ok {
			return "", clean, errors.New("download failed or canceled")
		}
		log.Infof("url:%s, wait other chat downloading", y.Url)
		<-prog.Ctx.Done() // 等待视频下载完成
		prog.wg.Wait()    // 等待视频上传完成，其实只等这一个就可以了
		log.Debugf("url:%s, file:%s, download finished by other context", y.Url, prog.Path)
		if db, ok := y.TakeDb(); ok { // 重新检查数据库，因为在等待的过程中已经下载完成，并且上传完成了
			log.Infof("url:%s, file id:%s, found in cache database", y.Url, db.FileId)
			return db.FileId, clean, nil
		}
		return "", clean, errors.New("db not found")
	}
	log.Infof("url:%s, start downloading", y.Url)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	config, prog, uid := y.makeConfig(ctx)
	ytDlpMap.Store(uid, prog)
	prog.wg.Add(1)
	clean = func() {
		ytDlpMap.Delete(uid)
		downloading.Delete(*y)
		prog.wg.Done()
		prog.wg.Wait()
		log.Infof("url:%s, file:%s, clean yt-dlp", y.Url, prog.Path)
		err := config.Clean()
		if err != nil {
			log.Errorf("clean yt-dlp error %s", err)
		}
	}
	downloading.Store(*y, uid)
	err := config.RunWithCtx(ctx)
	if err != nil {
		return "", clean, err
	}
	log.Infof("url:%s, file:%s, download finished", y.Url, prog.Path)
	return prog.Path, clean, nil
}

func parseYtDlKey(url, allText string) (*YtDlKey, error) {
	key := &YtDlKey{
		Url: url,
	}
	resolution := reResolutions.FindString(allText)
	if resolution != "" {
		resolution = strings.TrimSuffix(strings.ToLower(resolution), "p")
		key.Resolution = parseIntDefault(resolution, 1080)
	}
	log.Debugf("get key: %#v", key)
	return key, nil
}
func buildYtDlKeyFromText(text string) (*YtDlKey, error) {
	if text == "" {
		return nil, ErrNoUrl
	}
	log.Debugf("text: %s", text)
	urls := reUrl.FindAllString(text, -1)
	for _, url := range urls {
		if reYtDlpHosts.MatchString(url) {
			log.Debugf("url: %s", url)
			return parseYtDlKey(url, text)
		}
	}
	return nil, ErrNoUrl
}
func BuildYtDlKey(ctx *ext.Context) (*YtDlKey, error) {
	text := getText(ctx)
	if replyText := getTextMsg(ctx.EffectiveMessage.ReplyToMessage); replyText != "" {
		text += "\n\n" + getTextMsg(ctx.EffectiveMessage.ReplyToMessage)
	}
	return buildYtDlKeyFromText(text)
}
func checkOrAddRecentDownloaded(key *YtDlKey, chatId int64) bool {
	debounceKey := YtDlDebounceKey{
		YtDlKey: *key,
		ChatId:  chatId,
	}
	_, loaded := recentDownloaded.LoadOrStore(debounceKey, struct{}{})
	if !loaded {
		go func() {
			time.Sleep(time.Minute * 3)
			recentDownloaded.Delete(debounceKey)
		}()
	}
	return loaded
}

func saveFileToDb(key *YtDlKey, sent *gotgbot.Message) error {
	if sent == nil {
		return nil
	}
	fileId := ""
	if sent.Video != nil {
		fileId = sent.Video.FileId
	} else if sent.Audio != nil {
		fileId = sent.Audio.FileId
	}
	if fileId == "" {
		return nil
	}
	res := globalcfg.GetDb().Where(key).Take(&YtDlDb{})
	if res.RowsAffected > 0 {
		log.Infof("url:%s, file id:%s, found in database", key.Url, fileId)
		return nil
	}
	ytdb := &YtDlDb{
		YtDlKey: *key,
		FileId:  fileId,
	}
	db := globalcfg.GetDb().Create(ytdb)
	log.Infof("create yt-dlp db: url:%s, file id:%s, rows affected:%d, error:%s", key.Url, sent.Video.FileId, db.RowsAffected, db.Error)
	return db.Error
}

func sendVideo(bot *gotgbot.Bot, ctx *ext.Context, res string, text string) (*gotgbot.Message, error) {
	// 如果不是一个路径，那么就按照file id 的类型发送
	if !strings.ContainsAny(res, "/\\") {
		return bot.SendVideo(ctx.EffectiveChat.Id, res, nil)
	}
	replyId := int64(0)
	if ctx.EffectiveMessage != nil {
		replyId = ctx.EffectiveMessage.MessageId
	}
	frame, err := ytdlp.ExtractFirstFrame(res)
	if err != nil {
		log.Warnf("extract first frame error %#v", err)
		return nil, err
	}
	defer os.Remove(frame)
	probe, err := ffmpegProbes(res)
	if err != nil {
		log.Warnf("get video probe error %#v", err)
		return nil, err
	}
	sent, err := bot.SendVideo(ctx.EffectiveChat.Id, fileSchema(res), &gotgbot.SendVideoOpts{
		Thumbnail: fileSchema(frame),
		Duration:  int64(probe.GetDuration()),
		Width:     int64(probe.GetWidth()),
		Height:    int64(probe.GetHeight()),
		Caption:   text,
		ParseMode: "HTML",

		ReplyToMessageId:  replyId,
		SupportsStreaming: true,

		RequestOpts: &gotgbot.RequestOpts{
			Timeout: time.Hour * 6,
		},
	})
	if err != nil {
		log.Warnf("send video error %s", err)
	}
	return sent, err
}
func sendAudio(bot *gotgbot.Bot, ctx *ext.Context, res string, text string) (*gotgbot.Message, error) {
	if !strings.ContainsAny(res, "/\\") {
		return bot.SendAudio(ctx.EffectiveChat.Id, res, nil)
	}
	replyId := int64(0)
	if ctx.EffectiveMessage != nil {
		replyId = ctx.EffectiveMessage.MessageId
	}
	probe, err := ffmpegProbes(res)
	if err != nil {
		log.Warnf("get audio probe error %#v", err)
		return nil, err
	}
	sent, err := bot.SendAudio(ctx.EffectiveChat.Id, fileSchema(res), &gotgbot.SendAudioOpts{
		Duration:  int64(probe.GetDuration()),
		Caption:   text,
		ParseMode: "HTML",

		ReplyToMessageId: replyId,

		RequestOpts: &gotgbot.RequestOpts{
			Timeout: time.Hour * 6,
		},
	})
	if err != nil {
		log.Warnf("send audio error %s", err)
	}
	return sent, err
}
func answerCallback(bot *gotgbot.Bot, ctx *ext.Context, text string, alert bool) (bool, error) {
	ans, err := ctx.CallbackQuery.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{
		Text:      text,
		ShowAlert: alert,
	})
	return ans, err
}

func DownloadVideoCallback(bot *gotgbot.Bot, ctx *ext.Context) error {
	key, err := BuildYtDlKey(ctx)
	if err != nil {
		_, err = answerCallback(bot, ctx, "没有找到视频链接", true)
		return err
	}
	if checkOrAddRecentDownloaded(key, ctx.EffectiveChat.Id) {
		_, err = answerCallback(bot, ctx, "请勿频繁下载", true)
		return err
	}
	WithGroupLockToday(ctx.EffectiveChat.Id, func(daily *GroupStatDaily) {
		daily.DownloadVideoCount++
	})
	if db, ok := key.TakeDb(); ok {
		log.Infof("url:%s, file id:%s, found in cache database", key.Url, db.FileId)
		_, err = answerCallback(bot, ctx, "下载完成", false)
		if err != nil {
			log.Warnf("answer callback query error %s", err)
		}
		_, err = bot.SendVideo(ctx.EffectiveChat.Id, db.FileId, &gotgbot.SendVideoOpts{ReplyToMessageId: ctx.EffectiveMessage.MessageId})
		return err
	}
	_, err = answerCallback(bot, ctx, "正在下载，请稍等", false)
	if err != nil {
		log.Warnf("answer callback query error %s", err)
	}
	path, clean, err := key.Download()
	defer clean()
	if err != nil {
		_, err = answerCallback(bot, ctx, "下载失败", true)
		return err
	}
	sent, err := sendVideo(bot, ctx, path, "")
	if err != nil {
		_, err = bot.SendMessage(ctx.EffectiveChat.Id, "发送视频失败", nil)
		return err
	}
	return saveFileToDb(key, sent)
}

func DownloadInlinedBv(bot *gotgbot.Bot, ctx *ext.Context) error {
	uid, err := strconv.ParseInt(ctx.CallbackQuery.Data[len(biliInlineCallbackPrefix):], 16, 64)
	if err != nil {
		return err
	}
	var result BiliInlineResult
	tx := globalcfg.GetDb().Model(&BiliInlineResult{}).Where("uid = ?", uid).First(&result)
	if tx.Error != nil {
		_, _ = answerCallback(bot, ctx, "没有找到视频链接，可能是数据库问题", true)
		return tx.Error
	}
	key, err := buildYtDlKeyFromText(result.Text)
	if err != nil {
		_, err = answerCallback(bot, ctx, "没有找到视频链接", true)
		return err
	}
	chatId := result.ChatId
	msgId := result.Message
	if checkOrAddRecentDownloaded(key, chatId) {
		_, err = answerCallback(bot, ctx, "请勿频繁下载", true)
		return err
	}
	WithGroupLockToday(chatId, func(daily *GroupStatDaily) {
		daily.DownloadVideoCount++
	})
	if db, ok := key.TakeDb(); ok {
		log.Infof("url:%s, file id:%s, found in cache database", key.Url, db.FileId)
		_, err = answerCallback(bot, ctx, "下载完成", false)
		if err != nil {
			log.Warnf("answer callback query error %s", err)
		}
		_, err = bot.SendVideo(chatId, db.FileId, &gotgbot.SendVideoOpts{ReplyToMessageId: msgId})
		return err
	}
	_, err = answerCallback(bot, ctx, "正在下载，请稍等", false)
	if err != nil {
		log.Warnf("answer callback query error %s", err)
	}
	path, clean, err := key.Download()
	defer clean()
	if err != nil {
		_, err = answerCallback(bot, ctx, "下载失败", true)
		return err
	}
	ctx.EffectiveChat = &gotgbot.Chat{Id: chatId} // hack一下，因为这个函数是在inline query里面调用的，所以没有chat, 但是需要chat id
	sent, err := sendVideo(bot, ctx, path, "")
	if err != nil {
		_, err = answerCallback(bot, ctx, "发送视频失败", true)
		return err
	}
	return saveFileToDb(key, sent)
}

func DownloadVideo(bot *gotgbot.Bot, ctx *ext.Context) error {
	key, err := BuildYtDlKey(ctx)
	if err != nil {
		_, err = ctx.Message.Reply(bot, "没有找到视频链接", nil)
		return err
	}
	if checkOrAddRecentDownloaded(key, ctx.EffectiveChat.Id) {
		_, err = ctx.Message.Reply(bot, "请勿频繁下载", nil)
		return err
	}
	WithGroupLockToday(ctx.EffectiveChat.Id, func(daily *GroupStatDaily) {
		daily.DownloadVideoCount++
	})
	if db, ok := key.TakeDb(); ok {
		log.Infof("url:%s, file id:%s, found in cache database", key.Url, db.FileId)
		_, err = bot.SendVideo(ctx.EffectiveChat.Id, db.FileId, &gotgbot.SendVideoOpts{ReplyToMessageId: ctx.EffectiveMessage.MessageId})
		return err
	}
	dlMsg, err := ctx.Message.Reply(bot, "正在下载，请稍等", nil)
	if err != nil {
		log.Warnf("reply download command error:%s", err)
	}
	defer bot.DeleteMessage(ctx.EffectiveChat.Id, dlMsg.MessageId, nil)
	fileStr, clean, err := key.Download()
	defer clean()
	if err != nil {
		log.Warnf("download error %s", err)
		_, err = ctx.Message.Reply(bot, "下载失败", nil)
		return err
	}
	sent, err := sendVideo(bot, ctx, fileStr, "")
	if err != nil {
		_, err = ctx.Message.Reply(bot, "发送视频失败", nil)
		return err
	}
	return saveFileToDb(key, sent)
}

func DownloadAudio(bot *gotgbot.Bot, ctx *ext.Context) error {
	key, err := BuildYtDlKey(ctx)
	if err != nil {
		_, err = ctx.Message.Reply(bot, "没有找到音频链接", nil)
		return err
	}
	key.AudioOnly = true
	key.Resolution = 0
	if checkOrAddRecentDownloaded(key, ctx.EffectiveChat.Id) {
		_, err = ctx.Message.Reply(bot, "请勿频繁下载", nil)
		return err
	}
	WithGroupLockToday(ctx.EffectiveChat.Id, func(daily *GroupStatDaily) {
		daily.DownloadAudioCount++
	})
	if db, ok := key.TakeDb(); ok {
		log.Infof("url:%s, file id:%s, found in cache database", key.Url, db.FileId)
		_, err = bot.SendVideo(ctx.EffectiveChat.Id, db.FileId, &gotgbot.SendVideoOpts{ReplyToMessageId: ctx.EffectiveMessage.MessageId})
		return err
	}
	dlMsg, err := ctx.Message.Reply(bot, "正在下载，请稍等", nil)
	if err != nil {
		log.Warnf("reply download command error:%s", err)
	}
	defer bot.DeleteMessage(ctx.EffectiveChat.Id, dlMsg.MessageId, nil)
	path, clean, err := key.Download()
	defer clean()
	if err != nil {
		log.Warnf("download error %s", err)
		_, err = ctx.Message.Reply(bot, "下载失败", nil)
		return err
	}
	sent, err := sendAudio(bot, ctx, path, "")
	if err != nil {
		_, err = ctx.Message.Reply(bot, "发送音频失败", nil)
		return err
	}
	return saveFileToDb(key, sent)
}
