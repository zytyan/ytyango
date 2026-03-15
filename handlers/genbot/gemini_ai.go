package genbot

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	g "main/globalcfg"
	"main/globalcfg/h"
	"main/helpers/mdnormalizer"
	"math/rand/v2"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"google.golang.org/genai"
)

var mainBot *gotgbot.Bot
var log *slog.Logger
var client = g.NewPtrLinkedCfg(
	func(old, new *g.Config) bool {
		return old.GeminiKey != new.GeminiKey
	},
	func(new *g.Config) *genai.Client {
		ctx := context.Background()
		c, err := genai.NewClient(ctx, &genai.ClientConfig{
			APIKey:  new.GeminiKey,
			Backend: genai.BackendGeminiAPI,
		})
		if err != nil {
			panic(err)
		}
		return c
	},
)

func getGenAiClient() *genai.Client {
	return client.Get()
}

const (
	geminiSessionContentLimit = 150
	geminiInterval            = time.Second * 30
	geminiMemoriesLimit       = 60
)

var geminiModel = "gemini-3-flash-preview"

type geminiTopic struct {
	chatId  int64
	topicId int64
}

func newTopic(msg *gotgbot.Message) geminiTopic {
	res := geminiTopic{
		chatId: msg.Chat.Id,
	}
	if msg.IsTopicMessage {
		res.topicId = msg.MessageThreadId
	}
	return res
}

func IsGeminiReq(msg *gotgbot.Message) bool {
	text := msg.GetText()
	if strings.HasPrefix(text, "/") {
		return false
	}
	if strings.Contains(text, "@"+mainBot.Username) {
		return true
	}
	if msg.ReplyToMessage != nil {
		return msg.ReplyToMessage.GetSender().Id() == mainBot.Id
	}
	return false
}

var reLabelHeader = regexp.MustCompile(`(?s)^-start-label-\n.*-end-label-\n`)

//go:embed gemini_sysprompt.txt
var gDefaultSysPrompt string
var geminiSysPromptReplacer = NewReplacer(gDefaultSysPrompt)
var sysPromptReplacerCache = make(map[geminiTopic]*Replacer)
var gMu sync.Mutex

func getSysPrompt(msg *gotgbot.Message) *Replacer {
	gMu.Lock()
	defer gMu.Unlock()
	topic := newTopic(msg)
	if r, ok := sysPromptReplacerCache[topic]; ok {
		return r
	}
	tmpl, err := g.Q.GetGeminiSystemPrompt(context.Background(), topic.chatId, topic.topicId)
	if err == nil {
		r := NewReplacer(tmpl)
		sysPromptReplacerCache[topic] = &r
		return &r
	}
	sysPromptReplacerCache[topic] = &geminiSysPromptReplacer
	return &geminiSysPromptReplacer
}

func GeminiReply(bot *gotgbot.Bot, ctx *ext.Context) error {
	if !slices.Contains(g.GetConfig().AIChats, ctx.EffectiveChat.Id) {
		return nil
	}
	msg := ctx.EffectiveMessage
	topic := newTopic(msg)
	genCtx, cancel := context.WithTimeout(context.Background(), time.Minute*15)
	defer cancel()
	text := msg.GetText()
	createNewSession := false
	ignoreSessionTimeout := false
	if strings.Contains(text, "@new") {
		createNewSession = true
	} else if strings.Contains(text, "@last") {
		ignoreSessionTimeout = true
	}
	session := GeminiGetSession(genCtx, msg, createNewSession, ignoreSessionTimeout)
	if session == nil {
		return nil
	}
	if len(session.Memories) == 0 {
		memories, err := g.Q.ListGeminiMemory(genCtx, topic.chatId, topic.topicId, 30)
		if err != nil {
			return err
		}
		session.Memories = memories
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	setReaction(bot, msg, "👀")

	sysPromptCtx := ReplaceCtx{
		Bot: bot,
		Msg: ctx.EffectiveMessage,
		Now: time.Now(),
	}
	for _, mem := range session.Memories {
		sysPromptCtx.Memories = append(sysPromptCtx.Memories, mem.Content)
	}
	sysPrompt := getSysPrompt(msg).Replace(&sysPromptCtx)
	config := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(sysPrompt, genai.RoleModel),
		Tools: []*genai.Tool{
			{GoogleSearch: &genai.GoogleSearch{}},
		},
	}
	if err := session.AddTgMessage(bot, ctx.EffectiveMessage.ReplyToMessage); err != nil {
		return err
	}
	if err := session.AddTgMessage(bot, ctx.EffectiveMessage); err != nil {
		return err
	}
	if session.AllowCodeExecution {
		config.Tools[0].CodeExecution = &genai.ToolCodeExecution{}
	}
	defer session.DiscardTmpUpdates()

	actionCancel := h.WithChatAction(bot, "typing", msg.Chat.Id, msg.MessageThreadId, msg.IsTopicMessage)
	defer actionCancel()
	res, err := generate(genCtx, session, config)
	actionCancel()
	if err != nil {
		setReaction(bot, msg, "😭")
		_, _ = ctx.EffectiveMessage.Reply(bot, fmt.Sprintf("error:%s", err), nil)
		return err
	}
	_ = g.Q.IncrementSessionTokenCounters(
		genCtx,
		int64(res.UsageMetadata.PromptTokenCount),
		int64(res.UsageMetadata.CandidatesTokenCount+res.UsageMetadata.ThoughtsTokenCount),
		session.ID,
	)
	aiText := res.Text()
	if aiText == "" {
		aiText = "模型没有返回任何信息"
		if res.PromptFeedback != nil {
			aiText += "，原因: " + string(res.PromptFeedback.BlockReason) + res.PromptFeedback.BlockReasonMessage
		}
		setReaction(bot, msg, "🤯")
		session.DiscardTmpUpdates()
	}
	aiText = reLabelHeader.ReplaceAllString(aiText, "")
	normTxt, err := mdnormalizer.Normalize(aiText)
	var respMsg *gotgbot.Message
	if err != nil {
		respMsg, err = ctx.EffectiveMessage.Reply(bot, aiText, nil)
		log.Warn("parse markdown failed", "err", err)
	} else {
		respMsg, err = ctx.EffectiveMessage.Reply(bot, normTxt.Text, &gotgbot.SendMessageOpts{Entities: normTxt.Entities})
	}
	if err != nil {
		j, _ := res.MarshalJSON()
		log.Warn("gemini response", "resp", string(j), "err", err)
		return err
	}
	err = session.AddTgMessage(bot, respMsg)
	if err != nil {
		return err
	}
	return session.PersistTmpUpdates(genCtx)
}

func generate(ctx context.Context, session *GeminiSession, config *genai.GenerateContentConfig) (res *genai.GenerateContentResponse, err error) {
	client := getGenAiClient()
	base := 3.0
	jitter := 0.1
	multiplier := 1.5
	maxDelay := 180.0
	jit := func() float64 {
		return 1.0 + (rand.Float64()*2-1)*jitter
	}
	// rand/v2 可安全使用全局rand
	current := base
	sleepCtx := func(seconds float64) {
		d := time.Duration(seconds * float64(time.Second))
		t := time.NewTimer(d)
		defer t.Stop()
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			return
		}
	}
	for i := range 5 {
		wait := func() {
			if i == 4 {
				return
			}
			sleepCtx(current * jit())
			current = current * multiplier
			if current > maxDelay {
				current = maxDelay
			}
		}
		if ctx.Err() != nil {
			err = ctx.Err()
			break
		}
		res, err = client.Models.GenerateContent(ctx, geminiModel, session.ToGenaiContents(), config)
		if err != nil {
			wait()
			continue
		}
		if res.PromptFeedback != nil {
			return
		}
		if res.Text() == "" {
			wait()
			continue
		}
		return

	}
	return
}

func Init(bot *gotgbot.Bot, logger *slog.Logger) {
	mainBot = bot
	log = logger
}

func ChangeGeminiModel(bot *gotgbot.Bot, ctx *ext.Context) error {
	const flashLite = "gemini-3.1-flash-lite-preview"
	const flash = "gemini-3-flash-preview"
	oldModel := geminiModel
	if geminiModel == flashLite {
		geminiModel = flash
	} else {
		geminiModel = flashLite
	}
	_, err := ctx.EffectiveMessage.Reply(bot, fmt.Sprintf("model: %s => %s", oldModel, geminiModel), nil)
	return err
}
