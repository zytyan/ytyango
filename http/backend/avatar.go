package backend

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	"os"

	g "main/globalcfg"
	api "main/http/backend/ogen"

	"github.com/kolesa-team/go-webp/encoder"
	"github.com/kolesa-team/go-webp/webp"
)

func (h *Handler) GetUserAvatar(ctx context.Context, params api.GetUserAvatarParams) (api.GetUserAvatarRes, error) {
	if err := h.verifyTgAuth(params.Tgauth); err != nil {
		return &api.GetUserAvatarUnauthorized{Message: err.Error()}, nil
	}
	path, err := h.getUserProfilePhotoWebp(ctx, params.UserId)
	if err != nil {
		switch {
		case errors.Is(err, errUserNotFound):
			return &api.GetUserAvatarNotFound{Message: "user not found"}, nil
		case errors.Is(err, errUserNoPhoto):
			return &api.GetUserAvatarNotFound{Message: "user has no profile photo"}, nil
		case errors.Is(err, errBotUnavailable):
			return &api.GetUserAvatarInternalServerError{Message: err.Error()}, nil
		default:
			return &api.GetUserAvatarInternalServerError{Message: err.Error()}, nil
		}
	}
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &api.GetUserAvatarNotFound{Message: "avatar not found"}, nil
		}
		return &api.GetUserAvatarInternalServerError{Message: err.Error()}, nil
	}
	return &api.GetUserAvatarOK{Data: file}, nil
}

func (h *Handler) getUserProfilePhotoWebp(ctx context.Context, userId int64) (string, error) {
	user, err := g.Q.GetUserById(ctx, userId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", errUserNoPhoto
		}
		return "", errUserNotFound
	}
	if !user.ProfilePhoto.Valid || user.ProfilePhoto.String == "" {
		return "", errUserNoPhoto
	}
	photoPath := fmt.Sprintf("data/profile_photo/p_%s.webp", user.ProfilePhoto.String)
	if fileExists(photoPath) {
		return photoPath, nil
	}
	if h.bot == nil {
		return "", errBotUnavailable
	}
	if err := os.MkdirAll("data/profile_photo", 0o755); err != nil {
		return "", err
	}
	path, err := user.DownloadProfilePhoto(h.bot)
	if err != nil {
		return "", err
	}
	if err := webpConvert(path, photoPath); err != nil {
		return "", err
	}
	return photoPath, nil
}

func webpConvert(in, out string) error {
	fp, err := os.Open(in)
	if err != nil {
		return err
	}
	defer fp.Close()
	img, _, err := image.Decode(fp)
	if err != nil {
		return err
	}
	outFp, err := os.Create(out)
	if err != nil {
		return err
	}
	defer outFp.Close()
	opt, err := encoder.NewLossyEncoderOptions(encoder.PresetDefault, 80)
	if err != nil {
		return err
	}
	if err := webp.Encode(outFp, img, opt); err != nil {
		return err
	}
	return nil
}

func fileExists(filename string) bool {
	stat, err := os.Stat(filename)
	if err != nil {
		return false
	}
	return !stat.IsDir()
}
