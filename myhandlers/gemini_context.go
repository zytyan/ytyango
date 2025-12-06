package myhandlers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	g "main/globalcfg"
	"main/globalcfg/q"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"google.golang.org/genai"
)

const (
	geminiHistoryLimit = 120
	geminiMentionTTL   = time.Minute
	geminiBackoffBase  = time.Second
	geminiMaxRetry     = 3
	geminiRoleUser     = "user"
	geminiRoleModel    = "model"
)

type recentGeminiSession struct {
	sessionID int64
	expiresAt time.Time
}

var geminiRecentSessions = struct {
	mu     sync.Mutex
	recent map[int64]recentGeminiSession
}{recent: make(map[int64]recentGeminiSession)}

func rememberGeminiSession(chatID, sessionID int64) {
	geminiRecentSessions.mu.Lock()
	defer geminiRecentSessions.mu.Unlock()
	geminiRecentSessions.recent[chatID] = recentGeminiSession{
		sessionID: sessionID,
		expiresAt: time.Now().Add(geminiMentionTTL),
	}
}

func pickGeminiSessionFromCache(chatID int64) (int64, bool) {
	geminiRecentSessions.mu.Lock()
	defer geminiRecentSessions.mu.Unlock()
	ref, ok := geminiRecentSessions.recent[chatID]
	if !ok {
		return 0, false
	}
	if time.Now().After(ref.expiresAt) {
		delete(geminiRecentSessions.recent, chatID)
		return 0, false
	}
	return ref.sessionID, true
}

func messageMentionsBot(msg *gotgbot.Message) bool {
	if msg == nil || mainBot == nil {
		return false
	}
	text := getTextMsg(msg)
	return strings.Contains(text, "@"+mainBot.Username)
}

func isReplyToBotOrSelf(msg *gotgbot.Message) bool {
	if msg == nil || msg.ReplyToMessage == nil || msg.ReplyToMessage.From == nil {
		return false
	}
	from := msg.ReplyToMessage.From.Id
	if mainBot != nil && from == mainBot.Id {
		return true
	}
	if msg.From != nil && from == msg.From.Id {
		return true
	}
	return false
}

func resolveGeminiSession(ctx context.Context, msg *gotgbot.Message, userID int64, mentioned bool) (q.GeminiSession, sql.NullInt64, error) {
	if msg == nil {
		return q.GeminiSession{}, sql.NullInt64{}, errors.New("empty message")
	}
	if session, replySeq, found, err := sessionFromReply(ctx, msg); err != nil {
		return q.GeminiSession{}, sql.NullInt64{}, err
	} else if found {
		return session, replySeq, nil
	}
	if mentioned {
		if sessionID, ok := pickGeminiSessionFromCache(msg.Chat.Id); ok {
			session, err := g.Q.GetGeminiSessionByID(ctx, sessionID)
			if err == nil {
				return session, sql.NullInt64{}, nil
			}
			if !errors.Is(err, sql.ErrNoRows) {
				return q.GeminiSession{}, sql.NullInt64{}, err
			}
		}
	}
	session, err := newGeminiSession(ctx, msg, userID)
	return session, sql.NullInt64{}, err
}

func sessionFromReply(ctx context.Context, msg *gotgbot.Message) (q.GeminiSession, sql.NullInt64, bool, error) {
	if msg.ReplyToMessage == nil || !isReplyToBotOrSelf(msg) {
		return q.GeminiSession{}, sql.NullInt64{}, false, nil
	}
	replied := msg.ReplyToMessage
	stored, err := g.Q.GetGeminiMessageByTgMsg(ctx, msg.Chat.Id, replied.MessageId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return q.GeminiSession{}, sql.NullInt64{}, false, nil
		}
		return q.GeminiSession{}, sql.NullInt64{}, false, err
	}
	session, err := g.Q.GetGeminiSessionByID(ctx, stored.SessionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return q.GeminiSession{}, sql.NullInt64{}, false, nil
		}
		return q.GeminiSession{}, sql.NullInt64{}, false, err
	}
	return session, sql.NullInt64{Int64: stored.Seq, Valid: true}, true, nil
}

func newGeminiSession(ctx context.Context, msg *gotgbot.Message, userID int64) (q.GeminiSession, error) {
	now := time.Now()
	return g.Q.CreateGeminiSession(ctx, q.CreateGeminiSessionParams{
		ChatID:       msg.Chat.Id,
		StarterID:    userID,
		RootMsgID:    msg.MessageId,
		StartedAt:    q.UnixTime{Time: now},
		LastActiveAt: q.UnixTime{Time: now},
	})
}

func compactGeminiHistory(history []q.GeminiMessage) []*genai.Content {
	if len(history) == 0 {
		return nil
	}
	contents := make([]*genai.Content, 0, len(history))
	for i := len(history) - 1; i >= 0; i-- {
		msg := history[i]
		role := genai.Role(genai.RoleUser)
		prefix := "u"
		if msg.Role != geminiRoleUser {
			role = genai.RoleModel
			prefix = "b"
		}
		label := fmt.Sprintf("[%s%d", prefix, msg.Seq)
		if msg.ReplyToSeq.Valid {
			label += "->" + strconv.FormatInt(msg.ReplyToSeq.Int64, 10)
		}
		label += "]"
		user := g.Q.GetUserById(context.Background(), msg.FromID)
		label += fmt.Sprintf("(name:%s)\n", user.Name())
		contents = append(contents, genai.NewContentFromText(label+msg.Content, role))
	}
	return contents
}

func sanitizeGeminiText(text string) string {
	if mainBot != nil {
		text = strings.ReplaceAll(text, "@"+mainBot.Username, "")
	}
	return strings.TrimSpace(text)
}
