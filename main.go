package main

import (
	"context"
	g "main/globalcfg"
	hdrs "main/handlers"
	"main/http/backend"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/callbackquery"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/inlinequery"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/message"
	"go.uber.org/zap"
)

var log = g.GetLogger("main", zap.InfoLevel)
var compileTime = "unknown"

type GroupedDispatcher struct {
	*ext.Dispatcher
	autoInc int
	logger  *zap.Logger
	mutex   sync.Mutex
}

type HookedHandler struct {
	ext.Handler
	logger      *zap.Logger
	hitCounter  atomic.Int64
	funcName    string
	checkerName string
}

func (h *HookedHandler) CheckUpdate(b *gotgbot.Bot, ctx *ext.Context) bool {
	start := time.Now()
	var check bool
	defer func() {
		dur := time.Since(start)
		h.logger.Debug("check update",
			zap.Duration("elapsed", dur),
			zap.String("name", h.Name()),
			zap.String("func_name", h.funcName),
			zap.String("check_name", h.checkerName),
			zap.Int64("update_id", ctx.UpdateId),
			zap.Bool("check", check))
	}()
	check = h.Handler.CheckUpdate(b, ctx)
	return check
}

func (h *HookedHandler) HandleUpdate(b *gotgbot.Bot, ctx *ext.Context) error {
	start := time.Now()
	newCnt := h.hitCounter.Add(1)
	defer func() {
		dur := time.Since(start)
		h.logger.Info("handle update",
			zap.Duration("elapsed", dur),
			zap.String("func_name", h.funcName),
			zap.String("check_name", h.checkerName),
			zap.String("name", h.Name()),
			zap.Int64("update_id", ctx.UpdateId),
			zap.Int64("hit_count", newCnt))
	}()
	return h.Handler.HandleUpdate(b, ctx)
}

func (g *GroupedDispatcher) inc() int {
	g.mutex.Lock()
	g.autoInc++
	g.mutex.Unlock()
	return g.autoInc
}

func funcName(f any) (res string) {
	defer func() {
		if err := recover(); err != nil {
			res = ""
		}
	}()
	pc := reflect.ValueOf(f).Pointer()
	fn := runtime.FuncForPC(pc)
	return fn.Name()
}

func (g *GroupedDispatcher) Command(command string, handler handlers.Response) {
	hdr := &HookedHandler{
		Handler:     handlers.NewCommand(command, handler),
		logger:      g.logger,
		hitCounter:  atomic.Int64{},
		funcName:    funcName(handler),
		checkerName: "cmd(" + command + ")",
	}
	g.Dispatcher.AddHandlerToGroup(hdr, g.inc())
}

func (g *GroupedDispatcher) NewMessage(msg filters.Message, handler handlers.Response) {
	hdr := &HookedHandler{
		Handler:     handlers.NewMessage(msg, handler),
		logger:      g.logger,
		hitCounter:  atomic.Int64{},
		funcName:    funcName(handler),
		checkerName: funcName(msg),
	}
	g.Dispatcher.AddHandlerToGroup(hdr, g.inc())
}

func (g *GroupedDispatcher) NewCallback(filter filters.CallbackQuery, handler handlers.Response) {
	hdr := &HookedHandler{
		Handler:     handlers.NewCallback(filter, handler),
		logger:      g.logger,
		hitCounter:  atomic.Int64{},
		funcName:    funcName(handler),
		checkerName: funcName(filter),
	}
	g.Dispatcher.AddHandlerToGroup(hdr, g.inc())
}

func (g *GroupedDispatcher) NewInlineQuery(callback filters.InlineQuery, handler handlers.Response) {
	hdr := &HookedHandler{
		Handler:     handlers.NewInlineQuery(callback, handler),
		logger:      g.logger,
		hitCounter:  atomic.Int64{},
		funcName:    funcName(handler),
		checkerName: funcName(callback),
	}
	g.Dispatcher.AddHandlerToGroup(hdr, g.inc())
}

func newBot(token string) *gotgbot.Bot {
	bot := &gotgbot.Bot{
		Token: token,
		User:  gotgbot.User{},
		BotClient: gotgbot.BotClient(&gotgbot.BaseBotClient{
			Client:             http.Client{},
			UseTestEnvironment: false,
			DefaultRequestOpts: &gotgbot.RequestOpts{
				APIURL:  g.GetConfig().TgApiUrl,
				Timeout: time.Second * 130},
		}),
	}
	me, err := bot.GetMe(nil)
	if err != nil {
		panic(err)
	}
	bot.User = *me
	return bot
}

func zapLogFields(b *gotgbot.Bot, ctx *ext.Context) []zap.Field {
	fields := make([]zap.Field, 0, 8)
	fields = append(fields,
		zap.Int64("update_id", ctx.UpdateId),
		zap.Int64("bot_id", b.Id),
		zap.String("bot_username", b.Username),
	)
	if ctx.EffectiveChat != nil {
		fields = append(fields, zap.Int64("effective_chat_id", ctx.EffectiveChat.Id))
	}
	if ctx.EffectiveUser != nil {
		fields = append(fields, zap.Int64("effective_user_id", ctx.EffectiveUser.Id))
	}
	if ctx.EffectiveSender != nil {
		fields = append(fields, zap.Int64("effective_sender_id", ctx.EffectiveSender.Id()))
	}
	if ctx.EffectiveMessage != nil {
		fields = append(fields, zap.Int64("effective_msg_id", ctx.EffectiveMessage.MessageId))
	}
	return fields
}

func main() {
	log.Infof("compile time: %s", compileTime)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	defer func() {
		if err := g.Q.FlushChatStats(context.Background()); err != nil {
			log.Errorf("flush chat stats: %s", err)
		}
	}()
	go func() {
		<-ctx.Done()
		if err := g.Q.FlushChatStats(context.Background()); err != nil {
			log.Errorf("flush chat stats: %s", err)
		}
		os.Exit(0)
	}()
	token := g.GetConfig().BotToken
	b := newBot(token)
	hdrs.SetMainBot(b)
	hdrs.StartChatStatScheduler()
	g.Q.StartChatStatAutoSave(ctx, time.Minute)
	backend.GoListenAndServe("127.0.0.1:4021", b)
	go hdrs.HttpListen4019()
	dLog := log.Desugar()
	dp := GroupedDispatcher{Dispatcher: ext.NewDispatcher(&ext.DispatcherOpts{
		Error: func(b *gotgbot.Bot, ctx *ext.Context, err error) ext.DispatcherAction {
			fields := zapLogFields(b, ctx)
			fields = append(fields, zap.Error(err))
			dLog.Warn("an error occurred while handling update", fields...)
			return ext.DispatcherActionContinueGroups
		},
		Panic: func(b *gotgbot.Bot, ctx *ext.Context, r interface{}) {
			fields := zapLogFields(b, ctx)
			fields = append(fields,
				zap.Any("panic", r),
				zap.Stack("stack"),
			)
			dLog.Error("recovered from panic, a panic occurred while handling update.", fields...)
		},
		MaxRoutines: ext.DefaultMaxRoutines,
	}),
		autoInc: 0,
		logger:  g.GetLogger("handler-midware", zap.WarnLevel).Desugar(),
		mutex:   sync.Mutex{}}
	updater := ext.NewUpdater(dp.Dispatcher, &ext.UpdaterOpts{
		UnhandledErrFunc: func(err error) {
			log.Errorf("an error occurred while handling update: %s", err)
		},
	},
	)
	hMsg := handlers.NewMessage(message.All, hdrs.SaveMessage)
	hMsg.AllowChannel = true
	hMsg.AllowEdited = true
	wrappedHMsg := &HookedHandler{
		Handler:     hMsg,
		logger:      dp.logger,
		hitCounter:  atomic.Int64{},
		funcName:    funcName(hdrs.SaveMessage),
		checkerName: "any message",
	}
	dp.AddHandler(wrappedHMsg)

	dp.NewMessage(message.All, hdrs.StatMessage)
	dp.NewInlineQuery(inlinequery.All, hdrs.BiliMsgConverterInline)

	dp.Command("google", hdrs.Google)
	dp.Command("roll", hdrs.Roll)
	dp.Command("wiki", hdrs.Wiki)
	dp.Command("hhsh", hdrs.Hhsh)
	dp.Command("ocr", hdrs.OcrMessage)
	dp.Command("score", hdrs.CmdScore)
	dp.Command("prpr", hdrs.GenPrpr)
	dp.Command("calc", hdrs.SolveMath)
	dp.Command("downloadvideo", hdrs.DownloadVideo)
	dp.Command("downloadaudio", hdrs.DownloadAudio)
	dp.Command("getrank", hdrs.GetRank)
	dp.Command("diag_groupstat", hdrs.GroupStatDiagnostic)
	dp.Command("diag_sendstat", hdrs.SendGroupStat)
	dp.Command("diag_forcenewday", hdrs.ForceNewDay)
	dp.Command("diag_getcntbytime", hdrs.GetCntByTime)
	dp.Command("diag_msginfo", hdrs.GetMsgInfo)
	dp.Command("searchmsg", hdrs.SearchMessage)
	dp.Command("cochelp", hdrs.CoCHelp)
	dp.Command("list_attr", hdrs.ListDndAttr)
	dp.Command("del_attr", hdrs.DelDndAttr)
	dp.Command("new_battle", hdrs.NewBattle)
	dp.Command("webp2png", hdrs.WebpToPng)
	dp.Command("chat_config", hdrs.ShowChatCfg)

	dp.NewMessage(hdrs.BiliMsgFilter, hdrs.BiliMsgConverter)
	dp.Command("count_nsfw_pics", hdrs.CountNsfwPics)
	dp.Command("settimezone", hdrs.SetUserTimeZone)
	dp.NewMessage(hdrs.HasSinaGif, hdrs.Gif2Mp4)
	dp.NewCallback(hdrs.IsBilibiliBtn, hdrs.DownloadVideoCallback)
	dp.NewMessage(hdrs.DetectNsfwPhoto, hdrs.NsfwDetect)
	dp.NewMessage(hdrs.NeedSolve, hdrs.SolveMath)
	dp.NewMessage(hdrs.IsCalcExchangeRate, hdrs.ExchangeRateCalc)
	dp.NewMessage(hdrs.IsBilibiliInlineBtn2, hdrs.SaveBiliMsgCallbackMsgId)
	dp.NewMessage(hdrs.IsDndDice, hdrs.DndDice)
	dp.NewMessage(hdrs.IsSetDndAttr, hdrs.SetDndAttr)
	dp.NewMessage(hdrs.RequireNsfw, hdrs.SendRandRacy)
	dp.NewMessage(hdrs.IsSacabam, hdrs.GenSacabam)
	dp.NewCallback(hdrs.IsStopBattle, hdrs.StopBattle)
	dp.NewCallback(hdrs.IsNextRound, hdrs.NextRound)
	dp.NewMessage(hdrs.IsBattleCommand, hdrs.ExecuteBattleCommand)
	dp.NewMessage(hdrs.IsGeminiReq, hdrs.GeminiReply)
	dp.NewCallback(hdrs.IsBilibiliInlineBtn, hdrs.DownloadInlinedBv)
	dp.NewCallback(hdrs.IsNsfwPicRateBtn, hdrs.RateNsfwPicByBtn)

	dp.NewCallback(callbackquery.Prefix(hdrs.GroupConfigModifyPrefix), hdrs.ModifyGroupConfigByButton)

	err := updater.StartPolling(b, &ext.PollingOpts{
		DropPendingUpdates:    g.GetConfig().DropPendingUpdates,
		EnableWebhookDeletion: false,
		GetUpdatesOpts: &gotgbot.GetUpdatesOpts{
			Timeout: 120,
		},
	})
	if err != nil {
		panic("failed to start polling: " + err.Error())
	}
	log.Infof("%s has been started...", b.User.Username)
	updater.Idle()
}
