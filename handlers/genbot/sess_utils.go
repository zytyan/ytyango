package genbot

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	g "main/globalcfg"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"go.uber.org/zap"
)

func NewGeminiSession(bot *gotgbot.Bot, ctx *ext.Context) error {
	geminiSessions.mu.Lock()
	delete(geminiSessions.chatIdToSess, newTopic(ctx.EffectiveMessage))
	geminiSessions.mu.Unlock()
	_, err := ctx.EffectiveMessage.Reply(bot, "已重新开始session，新建会话不会携带历史记录。", nil)
	return err
}

func GetGeminiSessionId(bot *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	if rMsg := msg.ReplyToMessage; rMsg != nil {
		sessId, err := g.Q.GetSessionIdByMessage(context.Background(), rMsg.Chat.Id, rMsg.MessageId)
		if errors.Is(err, sql.ErrNoRows) {
			_, err := ctx.EffectiveMessage.Reply(bot, "您回复的消息不在会话中。", nil)
			return err
		} else if err != nil {
			_, err := ctx.EffectiveMessage.Reply(bot, "错误: "+err.Error(), nil)
			return err
		}
		_, err = ctx.EffectiveMessage.Reply(bot, fmt.Sprintf("Session ID: %d", sessId), nil)
		return err
	}
	geminiSessions.mu.Lock()
	sess, ok := geminiSessions.chatIdToSess[newTopic(ctx.EffectiveMessage)]
	geminiSessions.mu.Unlock()
	if !ok {
		_, err := ctx.EffectiveMessage.Reply(bot, "当前没有session，您可以回复特定消息来获取该消息的 session ID", nil)
		return err
	}
	_, err := ctx.EffectiveMessage.Reply(bot, fmt.Sprintf("Session ID: %d", sess.ID), nil)
	return err
}

func GetMemories(bot *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	topicId := int64(0)
	if msg.IsTopicMessage {
		topicId = msg.MessageThreadId
	}
	memories, err := g.Q.ListGeminiMemory(context.Background(), msg.Chat.Id, topicId, geminiMemoriesLimit)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		_, _ = msg.Reply(bot, err.Error(), nil)
		return err
	}
	if len(memories) == 0 {
		_, err = msg.Reply(bot, "当前没有任何记忆", nil)
		return err
	}
	buf := bytes.NewBuffer(nil)
	for i, m := range memories {
		_, _ = fmt.Fprintf(buf, "%d. %s\n", i+1, m.Content)
	}
	_, err = msg.Reply(bot, buf.String(), nil)
	return err
}

func SessionHelp(bot *gotgbot.Bot, ctx *ext.Context) error {
	text := `会话相关帮助：
/new_session 停止当前会话，创建新会话
/session_id 获取当前会话ID，若回复特定消息，则获取该消息的会话ID
/get_memories 获取Bot记忆`
	_, err := ctx.EffectiveMessage.Reply(bot, text, nil)
	return err
}

func setReaction(bot *gotgbot.Bot, msg *gotgbot.Message, emoji string) {
	_, err := msg.SetReaction(bot, &gotgbot.SetMessageReactionOpts{
		Reaction: []gotgbot.ReactionType{gotgbot.ReactionTypeEmoji{Emoji: emoji}},
	})
	if err != nil {
		log.Warn("set reaction", zap.Error(err))
	}
}
