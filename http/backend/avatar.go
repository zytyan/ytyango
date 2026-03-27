package backend

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	"io"
	"net/http"
	"os"

	g "main/globalcfg"

	"github.com/gin-gonic/gin"
	"github.com/kolesa-team/go-webp/encoder"
	"github.com/kolesa-team/go-webp/webp"
)

func (h *Handler) handleGetUserAvatar(c *gin.Context) {
	var params avatarURIParams
	if err := c.ShouldBindUri(&params); err != nil {
		c.JSON(http.StatusBadRequest, apiError{Message: err.Error()})
		return
	}
	var query avatarQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusUnauthorized, apiError{Message: err.Error()})
		return
	}
	if err := h.verifyTgAuth(query.TgAuth); err != nil {
		c.JSON(http.StatusUnauthorized, apiError{Message: err.Error()})
		return
	}
	path, err := h.getUserProfilePhotoWebp(c.Request.Context(), params.UserID)
	if err != nil {
		switch {
		case errors.Is(err, errUserNotFound):
			c.JSON(http.StatusNotFound, apiError{Message: "user not found"})
		case errors.Is(err, errUserNoPhoto):
			c.JSON(http.StatusNotFound, apiError{Message: "user has no profile photo"})
		case errors.Is(err, errBotUnavailable):
			c.JSON(http.StatusInternalServerError, apiError{Message: err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, apiError{Message: err.Error()})
		}
		return
	}
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			c.JSON(http.StatusNotFound, apiError{Message: "avatar not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, apiError{Message: err.Error()})
		return
	}
	c.Header("Content-Type", "image/webp")
	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, file)
	_ = file.Close()
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
