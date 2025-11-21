package q

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/puzpuzpuz/xsync/v3"
)

// TODO: 将这里的xsync.MapOf替换为WeakMap(若可能）或LRU Map，避免内存泄漏的问题
var userCache = xsync.NewMapOf[int64, *User]()

func (q *Queries) GetUserByTg(ctx context.Context, tgUser *gotgbot.User) (*User, error) {
	var err error
	userId := tgUser.Id
	user, _ := userCache.LoadOrCompute(userId, func() *User {
		user, erri := q.getUserById(ctx, userId)
		if errors.Is(erri, sql.ErrNoRows) {
			u := &User{
				UpdatedAt:       UnixTime{time.Now()},
				UserID:          tgUser.Id,
				FirstName:       tgUser.FirstName,
				LastName:        sql.NullString{String: tgUser.LastName, Valid: tgUser.LastName != ""},
				ProfileUpdateAt: UnixTime{time.Unix(0, 0)},
				ProfilePhoto:    sql.NullString{},
			}
			id, _ := q.updateUserBase(ctx, updateUserBaseParams{
				UpdatedAt: u.UpdatedAt,
				UserID:    u.UserID,
				FirstName: u.FirstName,
				LastName:  u.LastName,
				TimeZone:  u.TimeZone,
			})
			u.ID = id
			return u
		}
		if erri != nil {
			err = erri
			return nil
		}
		return &user
	})
	return user, err
}

func (q *Queries) GetUserById(ctx context.Context, id int64) *User {
	user, _ := q.getUserById(ctx, id)
	return &user
}

// TODO: 将这里的xsync.MapOf替换为WeakMap(若可能）或LRU Map，避免内存泄漏的问题
var chatCache = xsync.NewMapOf[int64, *ChatCfg]()

func (q *Queries) GetChatById(ctx context.Context, id int64) (*ChatCfg, error) {
	var err error
	c, _ := chatCache.LoadOrCompute(id, func() *ChatCfg {
		chat, erri := q.getChatById(ctx, id)
		if erri != nil {
			err = erri
			return nil
		}
		return &chat
	})
	return c, err
}

func (q *Queries) GetChatCfg(ctx context.Context, tgChat *gotgbot.Chat) (*ChatCfg, error) {
	var err error
	chatId := tgChat.Id
	c, _ := chatCache.LoadOrCompute(chatId, func() *ChatCfg {
		chat, erri := q.getChatById(ctx, chatId)
		if errors.Is(erri, sql.ErrNoRows) {
			chat, err = q.CreateNewChatDefaultCfg(ctx, chatId)
			return &chat
		} else if erri != nil {
			err = erri
			return nil
		}
		return &chat
	})
	return c, err

}

func (q *Queries) GetChatByWebId(ctx context.Context, webId int64) (*ChatCfg, error) {
	var err error
	webIdQ := sql.NullInt64{
		Int64: webId,
		Valid: true,
	}
	chatId, _ := q.getChatIdByWebId(ctx, webIdQ)
	// 这里不能直接用数据库把查找合并为一个，因为需要单例
	chat, err := q.GetChatById(ctx, chatId)
	return chat, err
}

func (u *User) TryUpdate(q *Queries, tgUser *gotgbot.User) error {
	needCommit := false
	if u.FirstName != tgUser.FirstName {
		u.FirstName = tgUser.FirstName
		needCommit = true
	}
	if tgUser.LastName != "" && u.LastName.String != tgUser.LastName {
		u.LastName.Valid = true
		u.LastName.String = tgUser.LastName
		needCommit = true
	}
	if needCommit {
		_, err := q.updateUserBase(context.Background(), updateUserBaseParams{
			UpdatedAt: UnixTime{time.Now()},
			UserID:    u.UserID,
			FirstName: u.FirstName,
			LastName:  u.LastName,
			TimeZone:  u.TimeZone,
		})
		return err
	}
	return nil
}

func (u *User) Name() string {
	if u == nil {
		return "<unknown>"
	}
	if !u.LastName.Valid || u.LastName.String == "" {
		return u.FirstName
	}
	return u.FirstName + " " + u.LastName.String
}
