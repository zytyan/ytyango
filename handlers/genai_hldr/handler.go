package genai_hldr

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	g "main/globalcfg"
	"main/mdnormalizer"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"go.uber.org/zap"
	"google.golang.org/genai"
)

const (
	defaultModel               = "gemini-3-flash-preview"
	geminiSessionContentLimit  = 100
	geminiInterval             = time.Minute * 3
	defaultRequestTimeout      = time.Second * 120
	defaultMaxScriptBytes      = 4 * 1024
	defaultMaxReplyBytes       = 2 * 1024
	defaultMaxCallStack        = 256
	defaultExecStatusNoteLimit = 160
)

var defaultAllowedChats = []int64{-1001471592463, -1001282155019, -1001126241898, -1001170816274}

type ExecLimits struct {
	Timeout        time.Duration
	MaxScriptBytes int
	MaxReplyBytes  int
	MaxCallStack   int
}

type Config struct {
	Model               string
	SessionContentLimit int
	AllowedChatIDs      []int64
	RequestTimeout      time.Duration
	Exec                ExecLimits
}

type Handler struct {
	cfg         Config
	prompt      *promptRenderer
	log         *zap.SugaredLogger
	logD        *zap.Logger
	clientOnce  sync.Once
	client      *genai.Client
	clientErr   error
	sessions    *sessionCache
	clientSetup func() (*genai.Client, error)
}

func New(cfg Config) (*Handler, error) {
	if cfg.Model == "" {
		cfg.Model = defaultModel
	}
	if cfg.SessionContentLimit == 0 {
		cfg.SessionContentLimit = geminiSessionContentLimit
	}
	if cfg.RequestTimeout == 0 {
		cfg.RequestTimeout = defaultRequestTimeout
	}
	if cfg.Exec.Timeout == 0 {
		cfg.Exec.Timeout = 2 * time.Second
	}
	if cfg.Exec.MaxScriptBytes == 0 {
		cfg.Exec.MaxScriptBytes = defaultMaxScriptBytes
	}
	if cfg.Exec.MaxReplyBytes == 0 {
		cfg.Exec.MaxReplyBytes = defaultMaxReplyBytes
	}
	if cfg.Exec.MaxCallStack == 0 {
		cfg.Exec.MaxCallStack = defaultMaxCallStack
	}
	if len(cfg.AllowedChatIDs) == 0 {
		cfg.AllowedChatIDs = defaultAllowedChats
	}
	pr, err := newPromptRenderer()
	if err != nil {
		return nil, err
	}
	logger := g.GetLogger("genai_hldr", zap.InfoLevel)
	return &Handler{
		cfg:         cfg,
		prompt:      pr,
		log:         logger,
		logD:        logger.Desugar(),
		sessions:    newSessionCache(),
		clientSetup: initGenaiClient,
	}, nil
}

func initGenaiClient() (*genai.Client, error) {
	ctx := context.Background()
	return genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  g.GetConfig().GeminiKey,
		Backend: genai.BackendGeminiAPI,
	})
}

func (h *Handler) getClient() (*genai.Client, error) {
	h.clientOnce.Do(func() {
		h.client, h.clientErr = h.clientSetup()
	})
	return h.client, h.clientErr
}

func getChatName(chat *gotgbot.Chat) string {
	if chat.Title != "" {
		return chat.Title
	}
	if chat.LastName == "" {
		return chat.FirstName
	}
	return chat.FirstName + " " + chat.LastName
}

func (h *Handler) Handle(bot *gotgbot.Bot, ctx *ext.Context) error {
	if ctx == nil || ctx.EffectiveMessage == nil || ctx.EffectiveChat == nil {
		return nil
	}
	if len(h.cfg.AllowedChatIDs) > 0 && !slices.Contains(h.cfg.AllowedChatIDs, ctx.EffectiveChat.Id) {
		return nil
	}
	client, err := h.getClient()
	if err != nil {
		return err
	}
	_, _ = ctx.EffectiveMessage.SetReaction(bot, &gotgbot.SetMessageReactionOpts{
		Reaction: []gotgbot.ReactionType{gotgbot.ReactionTypeEmoji{Emoji: "👀"}},
	})
	genCtx, cancel := context.WithTimeout(context.Background(), h.cfg.RequestTimeout)
	defer cancel()
	session, err := h.sessions.Get(genCtx, ctx.EffectiveMessage, bot, h.cfg)
	if err != nil {
		return err
	}
	if session == nil {
		return nil
	}
	session.mu.Lock()
	defer session.mu.Unlock()

	if err := session.AddTelegramMessage(bot, ctx.EffectiveMessage.ReplyToMessage); err != nil {
		return err
	}
	if err := session.AddTelegramMessage(bot, ctx.EffectiveMessage); err != nil {
		return err
	}
	prompt, err := h.prompt.Render(PromptData{
		ChatName:    session.ChatName,
		ChatType:    session.ChatType,
		BotName:     bot.FirstName + bot.LastName,
		BotUsername: bot.Username,
	})
	if err != nil {
		return err
	}
	cfg := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(prompt, genai.RoleModel),
		Tools: []*genai.Tool{
			{GoogleSearch: &genai.GoogleSearch{}},
		},
	}

	ticker := time.NewTicker(time.Second * 4)
	tickerCtx, tickerCancel := context.WithCancel(context.Background())
	defer func() {
		tickerCancel()
		ticker.Stop()
	}()
	go func() {
		_, _ = bot.SendChatAction(ctx.EffectiveChat.Id, "typing", nil)
		for {
			select {
			case <-ticker.C:
				_, _ = bot.SendChatAction(ctx.EffectiveChat.Id, "typing", nil)
			case <-tickerCtx.Done():
				return
			}
		}
	}()

	res, err := client.Models.GenerateContent(genCtx, h.cfg.Model, session.ToGenaiContents(), cfg)
	if err != nil {
		return h.handleError(bot, ctx, err)
	}
	cleanText, blocks := splitExecBlocks(res.Text())
	execNotes, execErr := h.runExecBlocks(genCtx, session, blocks)
	if execErr != nil {
		h.log.Warnw("execjs failed", "error", execErr, "chat_id", session.ChatID, "session_id", session.ID)
	}
	replyText := strings.TrimSpace(cleanText)
	if len(execNotes) > 0 {
		if replyText != "" {
			replyText += "\n\n"
		}
		replyText += strings.Join(execNotes, "\n")
	}
	if replyText == "" {
		replyText = "模型没有返回任何信息"
		if res.PromptFeedback != nil {
			replyText += "，原因: " + string(res.PromptFeedback.BlockReason) + res.PromptFeedback.BlockReasonMessage
		}
		if _, e := ctx.EffectiveMessage.SetReaction(bot, &gotgbot.SetMessageReactionOpts{
			Reaction: []gotgbot.ReactionType{gotgbot.ReactionTypeEmoji{Emoji: "😭"}},
		}); e != nil {
			h.log.Warnf("set reaction emoji failed: %v", e)
		}
	}
	normTxt, normErr := mdnormalizer.Normalize(replyText)
	var respMsg *gotgbot.Message
	if normErr != nil {
		respMsg, err = ctx.EffectiveMessage.Reply(bot, replyText, nil)
		h.logD.Warn("parse markdown failed", zap.Error(normErr))
	} else {
		respMsg, err = ctx.EffectiveMessage.Reply(bot, normTxt.Text, &gotgbot.SendMessageOpts{Entities: normTxt.Entities})
	}
	if err != nil {
		return err
	}
	if err := session.AddTelegramMessage(bot, respMsg); err != nil {
		return err
	}
	return session.PersistTmpUpdates(genCtx)
}

func (h *Handler) handleError(bot *gotgbot.Bot, ctx *ext.Context, err error) error {
	_, _ = ctx.EffectiveMessage.SetReaction(bot, &gotgbot.SetMessageReactionOpts{
		Reaction: []gotgbot.ReactionType{gotgbot.ReactionTypeEmoji{Emoji: "😭"}},
	})
	_, _ = ctx.EffectiveMessage.Reply(bot, fmt.Sprintf("error:%s", err), nil)
	return err
}

type execBlock struct {
	Summary string
	Script  string
}

func splitExecBlocks(text string) (string, []execBlock) {
	const start = "//execjs+"
	const end = "//execjs-"
	var blocks []execBlock
	var clean strings.Builder
	cursor := 0
	for {
		startIdx := strings.Index(text[cursor:], start)
		if startIdx == -1 {
			clean.WriteString(text[cursor:])
			break
		}
		startIdx += cursor
		clean.WriteString(text[cursor:startIdx])
		rest := text[startIdx+len(start):]
		endIdx := strings.Index(rest, end)
		if endIdx == -1 {
			clean.WriteString(text[startIdx:])
			break
		}
		blockRaw := rest[:endIdx]
		summary, script := parseExecBlock(blockRaw)
		if script != "" {
			blocks = append(blocks, execBlock{Summary: summary, Script: script})
		}
		cursor = startIdx + len(start) + endIdx + len(end)
	}
	return clean.String(), blocks
}

func parseExecBlock(raw string) (string, string) {
	lines := strings.Split(raw, "\n")
	for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}
	if len(lines) == 0 {
		return "", ""
	}
	summaryLine := strings.TrimSpace(lines[0])
	summary := summaryLine
	scriptLines := lines
	if strings.HasPrefix(strings.ToLower(summaryLine), "// summary:") {
		summary = strings.TrimSpace(summaryLine[len("// summary:"):])
		scriptLines = lines[1:]
	} else if strings.HasPrefix(strings.ToLower(summaryLine), "// summary") {
		summary = strings.TrimSpace(summaryLine[len("// summary"):])
		scriptLines = lines[1:]
	} else if strings.HasPrefix(summaryLine, "//") {
		summary = strings.TrimSpace(summaryLine[2:])
		scriptLines = lines[1:]
	} else {
		summary = ""
	}
	script := strings.TrimSpace(strings.Join(scriptLines, "\n"))
	if summary == "" {
		summary = limitString(summaryLine, 60)
	}
	return summary, script
}

func (h *Handler) runExecBlocks(ctx context.Context, session *Session, blocks []execBlock) ([]string, error) {
	notes := make([]string, 0, len(blocks))
	var firstErr error
	for _, blk := range blocks {
		summary := strings.TrimSpace(blk.Summary)
		if summary == "" {
			summary = "未提供简介"
		}
		output, err := session.RunJS(ctx, blk.Script, h.cfg.Exec)
		status := "成功"
		errMsg := ""
		if err != nil {
			status = "失败"
			errMsg = limitString(err.Error(), defaultExecStatusNoteLimit)
			if firstErr == nil {
				firstErr = err
			}
		}
		note := fmt.Sprintf("> 模型在此%s执行了一段脚本：%s", status, summary)
		if errMsg != "" {
			note += fmt.Sprintf("（%s）", errMsg)
		}
		notes = append(notes, note)
		session.appendModelText(note, "js_exec")
		if output != "" {
			session.appendModelText(output, "js_reply")
		}
	}
	return notes, firstErr
}

func limitString(s string, max int) string {
	if max <= 0 || len([]rune(s)) <= max {
		return s
	}
	runes := []rune(s)
	return string(runes[:max]) + "..."
}

type sessionCache struct {
	mu         sync.RWMutex
	sidToSess  map[int64]*Session
	chatIdSess map[int64]*Session
}

func newSessionCache() *sessionCache {
	return &sessionCache{
		sidToSess:  make(map[int64]*Session),
		chatIdSess: make(map[int64]*Session),
	}
}

func (c *sessionCache) Get(ctx context.Context, msg *gotgbot.Message, bot *gotgbot.Bot, cfg Config) (*Session, error) {
	if msg == nil {
		return nil, errors.New("nil message")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if msg.ReplyToMessage != nil {
		sessionId, err := g.Q.GetSessionIdByMessage(ctx, msg.Chat.Id, msg.ReplyToMessage.MessageId)
		if err == nil {
			if sess, ok := c.sidToSess[sessionId]; ok {
				return sess, nil
			}
		} else if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		if sessionId != 0 {
			data, err := g.Q.GetSessionById(ctx, sessionId)
			if err != nil {
				if !errors.Is(err, sql.ErrNoRows) {
					return nil, err
				}
			} else {
				sess := newSession(data, bot.FirstName+bot.LastName, bot.Username, cfg.Exec.MaxCallStack, cfg.Exec.MaxReplyBytes)
				if err := sess.loadContentFromDatabase(ctx, int64(cfg.SessionContentLimit)); err != nil && !errors.Is(err, sql.ErrNoRows) {
					return nil, err
				}
				c.sidToSess[sessionId] = sess
				c.chatIdSess[msg.Chat.Id] = sess
				return sess, nil
			}
		}
	}
	if sess, ok := c.chatIdSess[msg.Chat.Id]; ok {
		if time.Since(sess.updateTime) < geminiInterval {
			return sess, nil
		}
		delete(c.sidToSess, sess.ID)
	}
	delete(c.chatIdSess, msg.Chat.Id)
	data, err := g.Q.CreateNewGeminiSession(ctx, msg.Chat.Id, getChatName(&msg.Chat), msg.Chat.Type)
	if err != nil {
		return nil, err
	}
	sess := newSession(data, bot.FirstName+bot.LastName, bot.Username, cfg.Exec.MaxCallStack, cfg.Exec.MaxReplyBytes)
	if err := sess.loadContentFromDatabase(ctx, int64(cfg.SessionContentLimit)); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	c.sidToSess[sess.ID] = sess
	c.chatIdSess[msg.Chat.Id] = sess
	return sess, nil
}
