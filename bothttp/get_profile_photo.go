package bothttp

import (
	"image"
	_ "image/jpeg"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/kolesa-team/go-webp/encoder"
	"github.com/kolesa-team/go-webp/webp"
)

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
	err = webp.Encode(outFp, img, opt)
	if err != nil {
		return err
	}
	return nil
}
func getUserProfilePhotoWebp(userId int64) (string, error) {
	panic("TODO")
	/*
		user := g.Q.GetUserById(context.Background(), userId)
		if user == nil {
			return "", UserNotFound
		}
		if !user.ProfilePhoto.Valid {
			return "", UserNoProfilePhoto
		}
		photoPath := fmt.Sprintf("data/profile_photo/p_%s.webp", user.ProfilePhoto.String)
		if fileExists(photoPath) {
			return photoPath, nil
		}
		path, err := user.DownloadProfilePhoto(myhandlers.GetMainBot())
		if err != nil {
			return "", err
		}
		err = webpConvert(path, photoPath)
		if err != nil {
			return "", err
		}
		return photoPath, nil
	*/
}

func getUserProfilePhoto(ctx *gin.Context) {
	userIdStr, ok := ctx.GetQuery("user_id")
	if !ok {
		ctx.AbortWithStatusJSON(400, ErrArgInvalid.Msg("user_id is required"))
		return
	}
	userId, err := strconv.ParseInt(userIdStr, 10, 64)
	if userId <= 0 {
		ctx.AbortWithStatusJSON(400, ErrNoResource.Msg("user profile photo not found"))
		return
	}
	file, err := getUserProfilePhotoWebp(userId)
	if err != nil {
		ctx.AbortWithStatusJSON(400, ErrNoResource.Msg("user profile photo not found"))
		log.Warnf("get user profile photo error: %s", err.Error())
		return
	}
	ctx.File(file)
}

func reqBrowserCache(ctx *gin.Context) {
	ctx.Header("Cache-Control", "max-age=86400")
}
