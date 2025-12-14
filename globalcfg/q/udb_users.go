package q

import (
	"context"
	"database/sql"
	"errors"
	"main/helpers/lrusf"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
)

var userCache *lrusf.Cache[int64, *User]

func (q *Queries) GetUserById(ctx context.Context, id int64) (*User, error) {
	initCaches(q)
	return userCache.Get(id, func() (*User, error) {
		user, err := q.getUserById(ctx, id)
		if err != nil {
			return nil, err
		}
		return &user, nil
	})
}

func (q *Queries) GetUserByTg(ctx context.Context, tgUser *gotgbot.User) (*User, error) {
	if tgUser == nil {
		return nil, errors.New("tgUser is nil")
	}
	return q.GetUserById(ctx, tgUser.Id)
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

// DownloadProfilePhoto downloads the user's profile photo using provided bot and returns the local file path.
func (u *User) DownloadProfilePhoto(bot *gotgbot.Bot) (string, error) {
	if bot == nil {
		return "", errors.New("bot is nil")
	}
	if !u.ProfilePhoto.Valid || u.ProfilePhoto.String == "" {
		return "", errors.New("no profile photo")
	}
	file, err := bot.GetFile(u.ProfilePhoto.String, nil)
	if err != nil {
		return "", err
	}
	return file.FilePath, nil
}
