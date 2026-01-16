package genbot

import (
	"context"
	_ "embed"
	"fmt"
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
	"go.uber.org/zap"
	"google.golang.org/genai"
)

var mainBot *gotgbot.Bot
var log *zap.Logger

var getGenAiClient = sync.OnceValue(func() *genai.Client {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  g.GetConfig().GeminiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		panic(err)
	}
	return client
})

const (
	geminiSessionContentLimit = 150
	geminiModel               = "gemini-3-flash-preview"
	geminiInterval            = time.Second * 30
	geminiMemoriesLimit       = 60
)

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
	if !slices.Contains([]int64{-1001471592463, -1001282155019, -1001126241898,
		-1001170816274, -1003612476571}, ctx.EffectiveChat.Id) {
		return nil
	}
	msg := ctx.EffectiveMessage
	topic := newTopic(msg)
	genCtx, cancel := context.WithTimeout(context.Background(), time.Minute*15)
	defer cancel()
	session := GeminiGetSession(genCtx, msg)
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
	text := res.Text()
	if text == "" {
		text = "模型没有返回任何信息"
		if res.PromptFeedback != nil {
			text += "，原因: " + string(res.PromptFeedback.BlockReason) + res.PromptFeedback.BlockReasonMessage
		}
		setReaction(bot, msg, "🤯")
		session.DiscardTmpUpdates()
	}
	text = reLabelHeader.ReplaceAllString(text, "")
	normTxt, err := mdnormalizer.Normalize(text)
	var respMsg *gotgbot.Message
	if err != nil {
		respMsg, err = ctx.EffectiveMessage.Reply(bot, text, nil)
		log.Warn("parse markdown failed", zap.Error(err))
	} else {
		respMsg, err = ctx.EffectiveMessage.Reply(bot, normTxt.Text, &gotgbot.SendMessageOpts{Entities: normTxt.Entities})
	}
	if err != nil {
		j, _ := res.MarshalJSON()
		log.Warn("gemini response",
			zap.ByteString("resp", j),
			zap.Error(err))
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
	base := 30.0
	jitter := 0.1
	multiplier := 2.0
	maxDelay := 180.0

	current := base * (rand.Float64()*jitter + 1)
	wait := func() {
		time.Sleep(time.Duration(current) * time.Second)
		current = base * (rand.Float64()*jitter + multiplier)
		if current > maxDelay {
			current = maxDelay * (rand.Float64()*jitter + 1)
		}
	}
	for range 5 {
		res, err = client.Models.GenerateContent(ctx, geminiModel, session.ToGenaiContents(), config)
		if err == nil && res.Text() != "" {
			return res, nil
		}
		wait()
	}
	return
}
func Init(bot *gotgbot.Bot, logger *zap.Logger) {
	mainBot = bot
	log = logger
}
