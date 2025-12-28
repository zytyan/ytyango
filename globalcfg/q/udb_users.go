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
	return userCache.Get(id, func() (*User, error) {
		user, err := q.getUserById(ctx, id)
		if err != nil {
			return nil, err
		}
		return &user, nil
	})
}

func (q *Queries) GetOrCreateUserByTg(ctx context.Context, tgUser *gotgbot.User) (*User, error) {
	if tgUser == nil {
		return nil, errors.New("tgUser is nil")
	}
	user, err := q.GetUserById(ctx, tgUser.Id)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if user != nil {
		return user, nil
	}
	return q.CreateNewUserByTg(ctx, tgUser, nil)
}
func (q *Queries) CreateNewUserByTg(ctx context.Context, tgUser *gotgbot.User, bot *gotgbot.Bot) (*User, error) {
	if tgUser == nil {
		return nil, errors.New("tgUser is nil")
	}
	v, err := userCache.Get(tgUser.Id, func() (*User, error) {
		_, err := q.createNewUser(ctx, createNewUserParams{
			UpdatedAt:    UnixTime{time.Now()},
			UserID:       tgUser.Id,
			FirstName:    tgUser.FirstName,
			LastName:     sql.NullString{String: tgUser.LastName, Valid: tgUser.LastName != ""},
			Username:     sql.NullString{String: tgUser.Username, Valid: tgUser.Username != ""},
			ProfilePhoto: sql.NullString{},
			Timezone:     8 * 60 * 60,
		})
		if err != nil {
			return nil, err
		}
		user, err := q.getUserById(ctx, tgUser.Id)
		if err != nil {
			return nil, err
		}
		if bot == nil {
			return &user, nil
		}
		photoList, err := tgUser.GetProfilePhotos(bot, nil)
		if err != nil || len(photoList.Photos) == 0 {
			return &user, nil
		}
		back := len(photoList.Photos) - 1
		lastPhoto := photoList.Photos[back]
		if len(lastPhoto) == 0 {
			return &user, nil
		}
		photo := lastPhoto[len(lastPhoto)-1]
		_ = q.updateUserProfilePhoto(ctx, tgUser.Id, UnixTime{time.Now()}, sql.NullString{String: photo.FileId, Valid: true})
		return &user, nil
	})
	return v, err

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
	if tgUser.Username != "" && u.Username.String != tgUser.Username {
		u.Username = sql.NullString{String: tgUser.Username, Valid: true}
		needCommit = true
	}
	if needCommit {
		_, err := q.updateUserBase(context.Background(), updateUserBaseParams{
			UserID:    u.UserID,
			UpdatedAt: UnixTime{Time: time.Now()},
			FirstName: u.FirstName,
			LastName:  u.LastName,
			Username:  u.Username,
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

func (q *Queries) UpdateUserProfilePhoto(ctx context.Context, userID int64, profilePhoto string) error {
	return q.updateUserProfilePhoto(ctx, userID, UnixTime{time.Now()}, sql.NullString{String: profilePhoto, Valid: profilePhoto != ""})
}

func (q *Queries) UpdateUserTimeZone(ctx context.Context, user *User, zone int64) error {
	if user == nil {
		return errors.New("user is nil")
	}
	user.Timezone = zone
	return q.updateUserTimeZone(ctx, user.ID, zone)
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
