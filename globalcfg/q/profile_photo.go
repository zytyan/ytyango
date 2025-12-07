package q

import (
	"errors"

	"github.com/PaulSonOfLars/gotgbot/v2"
)

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
