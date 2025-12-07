package main

import (
	"context"
	g "main/globalcfg"
	"main/http/backend"
	"main/myhandlers"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"sync"
	"syscall"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/callbackquery"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/inlinequery"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/message"
	"go.uber.org/zap"
)

var log = g.GetLogger("main")
var compileTime = "unknown"

type GroupedDispatcher struct {
	*ext.Dispatcher
	autoInc int
	mutex   sync.Mutex
}

func (g *GroupedDispatcher) AddHandler(handler ext.Handler) {
	g.Dispatcher.AddHandlerToGroup(handler, g.autoInc)
	g.mutex.Lock()
	g.autoInc++
	g.mutex.Unlock()
}

func (g *GroupedDispatcher) AddCommand(command string, handler handlers.Response) {
	g.Dispatcher.AddHandlerToGroup(handlers.NewCommand(command, handler), g.autoInc)
	g.mutex.Lock()
	g.autoInc++
	g.mutex.Unlock()
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
	defer g.Q.FlushChatStats(context.Background())
	go func() {
		<-ctx.Done()
		_ = g.Q.FlushChatStats(context.Background())
		os.Exit(0)
	}()
	token := g.GetConfig().BotToken
	b := newBot(token)
	myhandlers.SetMainBot(b)
	myhandlers.StartChatStatScheduler()
	g.Q.StartChatStatAutoSave(ctx, time.Minute)
	backend.GoListenAndServe("127.0.0.1:4021", b)
	go myhandlers.HttpListen4019()
	dLog := log.Desugar()
	dispatcher := GroupedDispatcher{Dispatcher: ext.NewDispatcher(&ext.DispatcherOpts{
		Error: func(b *gotgbot.Bot, ctx *ext.Context, err error) ext.DispatcherAction {
			fields := zapLogFields(b, ctx)
			fields = append(fields, zap.Error(err))
			dLog.Warn("an error occurred while handling update", fields...)
			return ext.DispatcherActionContinueGroups
		},
		Panic: func(b *gotgbot.Bot, ctx *ext.Context, r interface{}) {
			fields := zapLogFields(b, ctx)
			stackBytes := debug.Stack()
			fields = append(fields,
				zap.Any("panic", r),
				zap.ByteString("stack", stackBytes),
			)
			dLog.Error("recovered from panic, a panic occurred while handling update.", fields...)
		},
		MaxRoutines: ext.DefaultMaxRoutines,
	}), autoInc: 0, mutex: sync.Mutex{}}
	updater := ext.NewUpdater(dispatcher.Dispatcher, &ext.UpdaterOpts{
		UnhandledErrFunc: func(err error) {
			log.Errorf("an error occurred while handling update: %s", err)
		},
	},
	)
	hMsg := handlers.NewMessage(message.All, myhandlers.SaveMessage)
	hMsg.AllowChannel = true
	hMsg.AllowEdited = true
	dispatcher.AddHandler(hMsg)
	dispatcher.AddHandler(handlers.NewMessage(message.All, myhandlers.StatMessage))
	dispatcher.AddHandler(handlers.NewInlineQuery(inlinequery.All, myhandlers.BiliMsgConverterInline))
	dispatcher.AddHandler(handlers.NewMessage(myhandlers.BiliMsgFilter, myhandlers.BiliMsgConverter))

	dispatcher.AddCommand("google", myhandlers.Google)
	dispatcher.AddCommand("roll", myhandlers.Roll)
	dispatcher.AddCommand("wiki", myhandlers.Wiki)
	dispatcher.AddCommand("hhsh", myhandlers.Hhsh)
	dispatcher.AddCommand("ocr", myhandlers.OcrMessage)
	dispatcher.AddCommand("score", myhandlers.CmdScore)
	dispatcher.AddCommand("prpr", myhandlers.GenPrpr)
	dispatcher.AddCommand("calc", myhandlers.SolveMath)
	dispatcher.AddCommand("downloadvideo", myhandlers.DownloadVideo)
	dispatcher.AddCommand("downloadaudio", myhandlers.DownloadAudio)
	dispatcher.AddCommand("getrank", myhandlers.GetRank)
	dispatcher.AddCommand("diag_groupstat", myhandlers.GroupStatDiagnostic)
	dispatcher.AddCommand("diag_sendstat", myhandlers.SendGroupStat)
	dispatcher.AddCommand("diag_forcenewday", myhandlers.ForceNewDay)
	dispatcher.AddCommand("diag_getcntbytime", myhandlers.GetCntByTime)
	dispatcher.AddCommand("diag_msginfo", myhandlers.GetMsgInfo)
	dispatcher.AddCommand("searchmsg", myhandlers.SearchMessage)
	dispatcher.AddCommand("cochelp", myhandlers.CoCHelp)
	dispatcher.AddCommand("list_attr", myhandlers.ListDndAttr)
	dispatcher.AddCommand("del_attr", myhandlers.DelDndAttr)
	dispatcher.AddCommand("new_battle", myhandlers.NewBattle)
	dispatcher.AddCommand("webp2png", myhandlers.WebpToPng)
	dispatcher.AddCommand("chat_config", myhandlers.ShowChatCfg)

	dispatcher.AddCommand("count_nsfw_pics", myhandlers.CountNsfwPics)
	dispatcher.AddCommand("settimezone", myhandlers.SetUserTimeZone)
	dispatcher.AddHandler(handlers.NewMessage(myhandlers.HasSinaGif, myhandlers.Gif2Mp4))
	dispatcher.AddHandler(handlers.NewCallback(myhandlers.IsBilibiliBtn, myhandlers.DownloadVideoCallback))
	dispatcher.AddHandler(handlers.NewMessage(myhandlers.DetectNsfwPhoto, myhandlers.NsfwDetect))
	dispatcher.AddHandler(handlers.NewMessage(myhandlers.NeedSolve, myhandlers.SolveMath))
	dispatcher.AddHandler(handlers.NewMessage(myhandlers.IsCalcExchangeRate, myhandlers.ExchangeRateCalc))
	dispatcher.AddHandler(handlers.NewMessage(myhandlers.IsBilibiliInlineBtn2, myhandlers.SaveBiliMsgCallbackMsgId))
	dispatcher.AddHandler(handlers.NewMessage(myhandlers.IsDndDice, myhandlers.DndDice))
	dispatcher.AddHandler(handlers.NewMessage(myhandlers.IsSetDndAttr, myhandlers.SetDndAttr))
	dispatcher.AddHandler(handlers.NewMessage(myhandlers.RequireNsfw, myhandlers.SendRandRacy))
	dispatcher.AddHandler(handlers.NewMessage(myhandlers.IsSacabam, myhandlers.GenSacabam))
	dispatcher.AddHandler(handlers.NewCallback(myhandlers.IsStopBattle, myhandlers.StopBattle))
	dispatcher.AddHandler(handlers.NewCallback(myhandlers.IsNextRound, myhandlers.NextRound))
	dispatcher.AddHandler(handlers.NewMessage(myhandlers.IsBattleCommand, myhandlers.ExecuteBattleCommand))
	dispatcher.AddHandler(handlers.NewMessage(myhandlers.IsGeminiReq, myhandlers.GeminiReply))
	dispatcher.AddHandler(handlers.NewCallback(myhandlers.IsBilibiliInlineBtn, myhandlers.DownloadInlinedBv))
	dispatcher.AddHandler(handlers.NewCallback(myhandlers.IsNsfwPicRateBtn, myhandlers.RateNsfwPicByBtn))

	dispatcher.AddHandler(handlers.NewCallback(callbackquery.Prefix(myhandlers.GroupConfigModifyPrefix), myhandlers.ModifyGroupConfigByButton))

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
