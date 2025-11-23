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
			param := createNewUserParams{
				UpdatedAt: UnixTime{time.Now()},
				UserID:    userId,
				FirstName: tgUser.FirstName,
				LastName:  sql.NullString{String: tgUser.LastName, Valid: tgUser.LastName != ""},
				Timezone:  480,
			}
			u.ID, err = q.createNewUser(ctx, param)
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

func (q *Queries) ChatCfgById(ctx context.Context, id int64) (*ChatCfg, error) {
	var err error
	c, _ := chatCache.LoadOrCompute(id, func() *ChatCfg {
		chat, erri := q.chatCfgById(ctx, id)
		if errors.Is(erri, sql.ErrNoRows) {
			chat, err = q.createNewChatCfgDefault(ctx, id)
			return fromInnerCfg(&chat)
		} else if erri != nil {
			err = erri
			return nil
		}
		return fromInnerCfg(&chat)
	})
	return c, err
}

func (q *Queries) ChatCfgByTg(ctx context.Context, tgChat *gotgbot.Chat) (*ChatCfg, error) {
	if tgChat == nil {
		return nil, errors.New("tgChat is nil")
	}
	return q.ChatCfgById(ctx, tgChat.Id)
}

func (q *Queries) GetChatByWebId(ctx context.Context, webId int64) (*ChatCfg, error) {
	var err error
	webIdQ := sql.NullInt64{
		Int64: webId,
		Valid: true,
	}
	chatId, _ := q.chatIdByWebId(ctx, webIdQ)
	// 这里不能直接用数据库把查找合并为一个，因为对每个群组都需要单例
	chat, err := q.ChatCfgById(ctx, chatId)
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
		_, err := q.updateUserBase(context.Background(), u.UserID, UnixTime{time.Now()}, u.FirstName, u.LastName)
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

func (q *Queries) UpdateUserProfilePhoto(ctx context.Context, userID int64, profilePhoto string) error {
	return q.updateUserProfilePhoto(ctx, userID, UnixTime{time.Now()}, sql.NullString{String: profilePhoto, Valid: profilePhoto != ""})
}

func (q *Queries) UpdateUserTimeZone(ctx context.Context, user *User, zone int64) error {
	if user == nil {
		return errors.New("user is nil")
	}
	user.Timezone = zone
	now := UnixTime{time.Now()}
	return q.updateUserTimeZone(ctx, user.ID, now, zone)
}

func (c *ChatCfg) Save(q *Queries) error {
	return q.updateChatCfg(context.Background(), updateChatCfgParams{
		AutoCvtBili:    c.AutoCvtBili,
		AutoOcr:        c.AutoOcr,
		AutoCalculate:  c.AutoCalculate,
		AutoExchange:   c.AutoExchange,
		AutoCheckAdult: c.AutoCheckAdult,
		SaveMessages:   c.SaveMessages,
		EnableCoc:      c.EnableCoc,
		RespNsfwMsg:    c.RespNsfwMsg,
		ID:             c.ID,
	})

}
