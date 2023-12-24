package main

import (
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/inlinequery"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/message"
	"main/bothttp"
	"main/globalcfg"
	"main/myhandlers"
	"net/http"
	"sync"
	"time"
)

var log = globalcfg.GetLogger("main")
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
				APIURL:  globalcfg.GetConfig().TgApiUrl,
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

func main() {
	log.Infof("compile time: %s", compileTime)
	token := globalcfg.GetConfig().BotToken
	b := newBot(token)
	myhandlers.SetMainBot(b)
	go bothttp.Run()
	go myhandlers.HttpListen4019()
	dispatcher := GroupedDispatcher{Dispatcher: ext.NewDispatcher(&ext.DispatcherOpts{
		// If an error is returned by a handler, log it and continue going.
		Error: func(b *gotgbot.Bot, ctx *ext.Context, err error) ext.DispatcherAction {
			log.Warnf("an error occurred while handling update: %s", err)
			return ext.DispatcherActionContinueGroups
		},
		MaxRoutines: ext.DefaultMaxRoutines,
	}), autoInc: 0, mutex: sync.Mutex{}}
	updater := ext.NewUpdater(&ext.UpdaterOpts{
		Dispatcher: dispatcher.Dispatcher,
	})
	hMsg := handlers.NewMessage(message.All, myhandlers.SaveMessage)
	hMsg.AllowChannel = true
	hMsg.AllowEdited = true
	dispatcher.AddHandler(hMsg)
	dispatcher.AddHandler(handlers.NewMessage(message.All, myhandlers.AddNewMsg))
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
	dispatcher.AddCommand("qrscan", myhandlers.QrScan)
	dispatcher.AddCommand("downloadvideo", myhandlers.DownloadVideo)
	dispatcher.AddCommand("downloadaudio", myhandlers.DownloadAudio)
	dispatcher.AddCommand("getrank", myhandlers.GetRank)
	dispatcher.AddCommand("diag_groupstat", myhandlers.GroupStatDiagnostic)
	dispatcher.AddCommand("diag_sendstat", myhandlers.SendGroupStat)
	dispatcher.AddCommand("diag_forcenewday", myhandlers.ForceNewDay)
	dispatcher.AddCommand("diag_getcntbytime", myhandlers.GetCntByTime)
	dispatcher.AddCommand("searchmsg", myhandlers.SearchMessage)
	dispatcher.AddHandler(handlers.NewMessage(myhandlers.HasSinaGif, myhandlers.Gif2Mp4))
	dispatcher.AddHandler(handlers.NewCallback(myhandlers.IsBilibiliBtn, myhandlers.DownloadVideoCallback))
	dispatcher.AddHandler(handlers.NewMessage(myhandlers.HasImage, myhandlers.SeseDetect))
	dispatcher.AddHandler(handlers.NewMessage(myhandlers.NeedSolve, myhandlers.SolveMath))
	dispatcher.AddHandler(handlers.NewMessage(myhandlers.IsCalcExchangeRate, myhandlers.ExchangeRateCalc))
	err := updater.StartPolling(b, &ext.PollingOpts{
		DropPendingUpdates:    globalcfg.GetConfig().DropPendingUpdates,
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
