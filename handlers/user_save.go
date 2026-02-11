package handlers

import (
	"context"
	"database/sql"
	"errors"
	g "main/globalcfg"
	"main/globalcfg/q"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

const profileRefreshInterval = time.Hour

func profileNeedUpdate(user *q.User) bool {
	return time.Since(user.ProfileUpdateAt.Time) > profileRefreshInterval
}

func botGetUserProfilePhotoFileId(bot *gotgbot.Bot, userId int64) (string, error) {
	photo, err := bot.GetUserProfilePhotos(userId, nil)
	if err != nil {
		return "", err
	}
	if len(photo.Photos) == 0 {
		return "", nil
	}
	return photo.Photos[len(photo.Photos)-1][len(photo.Photos[len(photo.Photos)-1])-1].FileId, nil
}

func UpdateUser(bot *gotgbot.Bot, ctx *ext.Context) error {
	if ctx.EffectiveUser == nil {
		return nil
	}
	user, err := g.Q.GetOrCreateUserByTg(context.Background(), ctx.EffectiveUser)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if err = user.TryUpdate(g.Q, ctx.EffectiveUser); err != nil {
		log.Warnf("update user name failed: %v", err)
	}
	if !profileNeedUpdate(user) {
		return nil
	}
	photo, err := botGetUserProfilePhotoFileId(bot, user.UserID)
	if err != nil {
		log.Warnf("get user profile photo failed: %v", err)
		return nil
	}
	err = g.Q.UpdateUserProfilePhoto(context.Background(), user.UserID, photo)
	if err != nil {
		log.Warnf("update profile photo failed: %v", err)
		return nil
	}
	user.ProfileUpdateAt = q.UnixTime{Time: time.Now()}
	user.ProfilePhoto = sql.NullString{String: photo, Valid: photo != ""}

	return nil
}
