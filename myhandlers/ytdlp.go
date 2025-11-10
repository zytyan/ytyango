package myhandlers

import (
	"errors"
	"fmt"
	"html"
	"main/globalcfg"
	"main/groupstatv2"
	"main/helpers/ytdlp"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/puzpuzpuz/xsync/v3"
	"golang.org/x/time/rate"
	"gorm.io/gorm"
)

var (
	reUrl         = regexp.MustCompile(`(?i)(https?://)?(\w+\.)+(\w+)/[a-zA-Z0-9_\-.~/?#\[\]@!$&'()*+,;=%]+`)
	reYtDlpHosts  = regexp.MustCompile(`(?i)(youtu\.be|youtube\.com|b23\.tv|bilibili\.com|twitter\.com)`)
	reResolutions = regexp.MustCompile(`(?i)(\b|^)(144|360|480|720|1080)([pP]?)(\b|$)`)
)
var (
	ErrNoUrl = errors.New("no url")
)

var downloading = xsync.NewMapOf[YtDlKey, *sync.WaitGroup]()

type rateLimiter struct {
	total            *rate.Limiter
	totalLowPriority *rate.Limiter
	byChatId         *xsync.MapOf[int64, *rate.Limiter]
	byDlKey          *xsync.MapOf[YtDlDebounceKey, struct{}]
}

var gRateLimiter = &rateLimiter{
	total:            rate.NewLimiter(rate.Every(time.Minute), 60),
	totalLowPriority: rate.NewLimiter(rate.Every(time.Minute), 10),
	byChatId:         xsync.NewMapOf[int64, *rate.Limiter](),
	byDlKey:          xsync.NewMapOf[YtDlDebounceKey, struct{}](),
}

func (r *rateLimiter) Check(chatId int64, key YtDlKey) (string, bool) {
	r1 := r.total.Reserve()
	if !r1.OK() {
		return "下载过于频繁，请稍后再试", true
	}
	if chatId != -1001471592463 && chatId != globalcfg.GetConfig().God {
		r2 := r.totalLowPriority.Reserve()
		if !r2.OK() {
			r1.Cancel()
			return "下载过于频繁，请稍后再试", true
		}
		if limiter, _ := r.byChatId.LoadOrStore(chatId, rate.NewLimiter(rate.Every(time.Minute), 6)); !limiter.Allow() {
			r1.Cancel()
			r2.Cancel()
			return "该聊天操作过于频繁，请稍后再试", true
		}
	}
	debounceKey := YtDlDebounceKey{
		YtDlKey: key,
		ChatId:  chatId,
	}
	if _, loaded := r.byDlKey.LoadOrStore(debounceKey, struct{}{}); loaded {
		time.AfterFunc(3*time.Minute, func() { r.byDlKey.Delete(debounceKey) })
		return "请勿频繁下载同一文件", true
	}
	return "", false
}

type YtDlKey struct {
	Url        string `gorm:"index"`
	AudioOnly  bool
	Resolution int
}

type YtDlDebounceKey struct {
	YtDlKey
	ChatId int64
}

type YtDlResult struct {
	YtDlKey
	File        string `gorm:"column:file_id"`
	Title       string `gorm:"default:''"`
	Description string `gorm:"default:''"`
	Uploader    string `gorm:"default:''"`
	UploadCount int64  `gorm:"default:1"`
	mu          sync.Mutex
}

func makeYtDlResult(key YtDlKey, resp *ytdlp.Resp) *YtDlResult {
	if resp == nil {
		return nil
	}
	return &YtDlResult{
		YtDlKey: key,
		File:    resp.FilePath,
		Title:   resp.Title(),
		Description: reUrl.ReplaceAllStringFunc(resp.Description(), func(s string) string {
			return " " + s + " "
		}),
		Uploader:    resp.Uploader(),
		UploadCount: 1,
		mu:          sync.Mutex{},
	}
}

func (r *YtDlResult) replaceFile(newFile string) {
	r.mu.Lock()
	if r.IsFileId() || r.File == newFile {
		r.mu.Unlock()
		return
	}
	r.File = newFile
	r.mu.Unlock()
	log.Infof("replace file %s to %s", r.File, newFile)
	err := globalcfg.GetDb().Create(r).Error

	if err != nil {
		log.Warnf("replace file error %s", err)
	}
}

func (r *YtDlResult) IsFileId() bool {
	return !strings.ContainsAny(r.File, `/\`)
}

func (r *YtDlResult) formatCaption(user *gotgbot.User) string {
	title := html.EscapeString(r.Title)
	uploader := html.EscapeString(r.Uploader)
	description := html.EscapeString(r.Description)

	if title == "" {
		title = "未知标题"
	}
	title = cutString(title, 200)
	if uploader == "" {
		uploader = "未知上传者"
	}
	uploader = cutString(uploader, 200)
	if description == "" {
		description = "无描述"
	}
	downloader := html.EscapeString(getUserName(user))
	description = cutString(description, 1024-len([]rune(title))-len([]rune(uploader))-len([]rune(downloader))-20)
	downloader = fmt.Sprintf(`<a href="tg://user?id=%d">%s</a>`, user.Id, downloader)
	return fmt.Sprintf("<b>%s</b>\nUploader: %s\n%s\n<blockquote expandable>%s</blockquote>", title, uploader, downloader, description)
}

func (y *YtDlKey) TakeDb() (*YtDlResult, bool) {
	db := &YtDlResult{}
	err := globalcfg.GetDb().Where(y).Take(db).Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		log.Infof("yt-dlp db not found url:%s", y.Url)
		return nil, false
	} else if err != nil {
		log.Warnf("yt-dlp db not found and get an error: %s", err)
		return nil, false
	}
	if db.File != "" {
		db.UploadCount++
		globalcfg.GetDb().Where(*y).Save(db)
		return db, true
	}
	return nil, false
}

func (y *YtDlKey) makeConfig() *ytdlp.Req {
	req := &ytdlp.Req{
		Url:             y.Url,
		AudioOnly:       y.AudioOnly,
		Resolution:      y.Resolution,
		EmbedMetadata:   true,
		PriorityFormats: []string{"h264", "h265", "av01"},
		WriteInfoJson:   true,
	}
	return req
}

func (y *YtDlKey) Download() (*YtDlResult, func(), error) {
	clean := func() {}
	if db, ok := y.TakeDb(); ok { // 重新检查数据库，因为在等待的过程中已经下载完成，并且上传完成了
		log.Infof("url:%s, file id:%s, found in database", y.Url, db.File)
		return db, clean, nil
	}
	newWg := &sync.WaitGroup{}
	if wg, loaded := downloading.LoadOrStore(*y, newWg); loaded {
		log.Infof("url:%s, wait other chat downloading", y.Url)
		wg.Wait()
		if db, ok := y.TakeDb(); ok { // 重新检查数据库，因为在等待的过程中已经下载完成，并且上传完成了
			log.Infof("url:%s, file id:%s, found in cache database", y.Url, db.File)
			return db, clean, nil
		}
		return nil, clean, errors.New("db not found")
	}
	newWg.Add(1)
	log.Infof("url:%s, start downloading", y.Url)
	req := y.makeConfig()
	resp, err := req.RunWithTimeout(30 * time.Minute)
	result := makeYtDlResult(*y, resp)
	clean = func() {
		newWg.Done()
		downloading.Delete(*y)
		log.Infof("url:%s, file:%s, clean yt-dlp", y.Url, resp.FilePath)
		err := req.Clean()
		if err != nil {
			log.Errorf("clean yt-dlp error %s", err)
		}
	}

	if err != nil {
		log.Warnf("download url: %s failed, err = %s", y.Url, err)
		return nil, clean, err
	}
	log.Infof("url:%s, file:%s, download completed", y.Url, resp.FilePath)
	return result, clean, nil
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
		text += "\n" + getTextMsg(ctx.EffectiveMessage.ReplyToMessage)
	}
	return buildYtDlKeyFromText(text)
}

func sendVideo(bot *gotgbot.Bot, ctx *ext.Context, res *YtDlResult) (*gotgbot.Message, error) {
	replyId := int64(0)
	if ctx.EffectiveMessage != nil {
		replyId = ctx.EffectiveMessage.MessageId
	}
	text := res.formatCaption(ctx.EffectiveUser)
	// 如果不是一个路径，那么就按照file id 的类型发送
	if res.IsFileId() {
		return bot.SendVideo(ctx.EffectiveChat.Id, gotgbot.InputFileByID(res.File), &gotgbot.SendVideoOpts{
			Caption:         text,
			ReplyParameters: MakeReplyToMsgID(replyId),
			ParseMode:       "HTML",
		})
	}
	frame, err := ytdlp.ExtractFirstFrame(res.File)
	if err != nil {
		log.Warnf("extract first frame error %#v", err)
		return nil, err
	}
	probe, err := ffmpegProbes(res.File)
	if err != nil {
		log.Warnf("get video probe error %#v", err)
		return nil, err
	}
	sent, err := bot.SendVideo(ctx.EffectiveChat.Id, fileSchema(res.File), &gotgbot.SendVideoOpts{
		Thumbnail: fileSchema(frame).(gotgbot.InputFile),
		Duration:  int64(probe.GetDuration()),
		Width:     int64(probe.GetWidth()),
		Height:    int64(probe.GetHeight()),
		Caption:   text,
		ParseMode: "HTML",

		ReplyParameters:   MakeReplyToMsgID(replyId),
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

func sendAudio(bot *gotgbot.Bot, ctx *ext.Context, res *YtDlResult) (*gotgbot.Message, error) {
	replyId := int64(0)
	if ctx.EffectiveMessage != nil {
		replyId = ctx.EffectiveMessage.MessageId
	}
	text := res.formatCaption(ctx.EffectiveUser)
	if res.IsFileId() {
		return bot.SendAudio(
			ctx.EffectiveChat.Id, gotgbot.InputFileByID(res.File),
			&gotgbot.SendAudioOpts{ReplyParameters: MakeReplyToMsgID(replyId), Caption: text, ParseMode: "HTML"},
		)
	}

	probe, err := ffmpegProbes(res.File)
	if err != nil {
		log.Warnf("get audio probe error %#v", err)
		return nil, err
	}
	sent, err := bot.SendAudio(ctx.EffectiveChat.Id, fileSchema(res.File), &gotgbot.SendAudioOpts{
		Duration:  int64(probe.GetDuration()),
		Caption:   text,
		ParseMode: "HTML",

		ReplyParameters: MakeReplyToMsgID(replyId),

		RequestOpts: &gotgbot.RequestOpts{
			Timeout: time.Hour * 6,
		},
	})
	if err != nil {
		log.Warnf("send audio error %s", err)
	}
	return sent, err
}

func DownloadVideoCallback(bot *gotgbot.Bot, ctx *ext.Context) error {
	answer := MakeAnswerCallback(bot, ctx)
	key, err := BuildYtDlKey(ctx)
	if err != nil {
		answer("没有找到视频链接", true)
		return err
	}
	if msg, limited := gRateLimiter.Check(ctx.EffectiveChat.Id, *key); limited {
		answer(msg, true)
		return err
	}
	groupstatv2.GetGroupToday(ctx.Message.MessageId).DownloadVideoCount.Inc()
	answer("视频开始下载", false)
	msgOpt := &gotgbot.SendMessageOpts{ParseMode: "HTML"}
	atUser := fmt.Sprintf(`<a href="tg://user?id=%d">%s</a> `,
		ctx.EffectiveUser.Id, html.EscapeString(getUserName(ctx.EffectiveUser)))
	result, clean, err := key.Download()
	defer clean()
	if err != nil {
		_, err = bot.SendMessage(ctx.EffectiveChat.Id, atUser+"下载失败", msgOpt)
		return err
	}
	sent, err := sendVideo(bot, ctx, result)
	if err != nil {
		_, err = bot.SendMessage(ctx.EffectiveChat.Id, atUser+"发送视频失败", msgOpt)
		return err
	}
	result.replaceFile(sent.Video.FileId)
	return nil
}

func DownloadInlinedBv(bot *gotgbot.Bot, ctx *ext.Context) error {
	answer := MakeAnswerCallback(bot, ctx)
	log.Infof("download inlined bv %s", ctx.CallbackQuery.Data)
	uid, err := strconv.ParseInt(ctx.CallbackQuery.Data[len(biliInlineCallbackPrefix):], 16, 64)
	if err != nil {
		answer("没有找到视频链接", true)
		return err
	}
	var inlineResult BiliInlineResult
	if err = globalcfg.GetDb().Model(&BiliInlineResult{}).Where("uid = ?", uid).First(&inlineResult).Error; err != nil {
		answer("没有找到视频链接，可能是数据库错误", true)
		return err
	}
	key, err := buildYtDlKeyFromText(inlineResult.Text)
	if err != nil {
		answer("没有找到视频链接", true)
		return err
	}
	chatId := inlineResult.ChatId
	if msg, limited := gRateLimiter.Check(chatId, *key); limited {
		answer(msg, true)
		return err
	}
	log.Infof("chat id %d, download inlined bv %s", chatId, ctx.CallbackQuery.Data)
	groupstatv2.GetGroupToday(chatId).DownloadVideoCount.Inc()
	answer("视频开始下载", false)
	msgOpt := &gotgbot.SendMessageOpts{ParseMode: "HTML"}
	atUser := fmt.Sprintf(`<a href="tg://user?id=%d">%s</a> `,
		ctx.CallbackQuery.From.Id, html.EscapeString(getUserName(&ctx.CallbackQuery.From)))

	result, clean, err := key.Download()
	log.Infof("download process exit. bv %s", ctx.CallbackQuery.Data)
	defer clean()
	if err != nil {
		_, err = bot.SendMessage(chatId, atUser+"下载失败", msgOpt)
		return err
	}
	log.Infof("download succeed")
	ctx.EffectiveChat = &gotgbot.Chat{Id: chatId} // hack一下，因为这个函数是在inline query里面调用的，所以没有chat, 但是需要chat id
	ctx.EffectiveUser = &ctx.CallbackQuery.From   // 同上
	sent, err := sendVideo(bot, ctx, result)
	if err != nil {
		_, err = bot.SendMessage(chatId, atUser+"发送视频失败", msgOpt)
		return err
	}
	result.replaceFile(sent.Video.FileId)
	return nil
}

func DownloadVideo(bot *gotgbot.Bot, ctx *ext.Context) error {
	reply, del := MakeDebounceMustReply(bot, ctx, time.Second*1)
	key, err := BuildYtDlKey(ctx)
	if err != nil {
		reply("没有找到视频链接")
		return err
	}
	if msg, limited := gRateLimiter.Check(ctx.EffectiveChat.Id, *key); limited {
		reply(msg)
		return err
	}
	groupstatv2.GetGroupToday(ctx.EffectiveChat.Id).DownloadVideoCount.Inc()
	reply("正在下载，请稍等")
	result, clean, err := key.Download()
	defer clean()
	if err != nil {
		log.Warnf("download error %s", err)
		reply("下载失败")
		return err
	}
	sent, err := sendVideo(bot, ctx, result)
	if err != nil {
		reply("发送视频失败")
		return err
	}
	del()
	result.replaceFile(sent.Video.FileId)
	return nil
}

func DownloadAudio(bot *gotgbot.Bot, ctx *ext.Context) error {
	reply, del := MakeDebounceMustReply(bot, ctx, time.Second*1)
	key, err := BuildYtDlKey(ctx)
	if err != nil {
		reply("没有找到音频链接")
		return err
	}
	key.AudioOnly = true
	key.Resolution = 0
	if msg, limited := gRateLimiter.Check(ctx.EffectiveChat.Id, *key); limited {
		reply(msg)
		return err
	}
	groupstatv2.GetGroupToday(ctx.EffectiveChat.Id).DownloadAudioCount.Inc()
	reply("正在下载，请稍等")
	result, clean, err := key.Download()
	defer clean()
	if err != nil {
		reply("下载失败")
		return err
	}
	sent, err := sendAudio(bot, ctx, result)
	if err != nil {
		reply("发送音频失败")
		return err
	}
	del()
	result.replaceFile(sent.Audio.FileId)
	return nil
}
