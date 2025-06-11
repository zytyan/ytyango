package bothttp

import (
	"github.com/gin-gonic/gin"
	"main/myhandlers"
	"os"
	"strconv"
)

type User struct {
	UserId int64  `json:"user_id,omitempty"`
	Name   string `json:"name,omitempty"`
}

// 此处均假设用户已经通过了身份认证，无需考虑身份认证的问题
func getUserInfo(ctx *gin.Context) {
	userIdStr, ok := ctx.GetQuery("user_id")
	if !ok {
		ctx.JSON(400, ErrArgInvalid.Msg("user_id not found"))
		return
	}
	userId, err := strconv.ParseInt(userIdStr, 10, 64)
	if err != nil {
		ctx.JSON(400, ErrArgInvalid.Msg("user_id is invalid"))
		return
	}
	if userId == 0 {
		ctx.JSON(400, UserNotFound.Msg("user not found"))
		return
	}
	user := myhandlers.GetUser(userId)
	if user == nil {
		ctx.JSON(400, UserNotFound.Msg("user not found"))
		return
	}
	ctx.JSON(200, User{
		UserId: userId,
		Name:   user.Name(),
	})
}

func fileExists(filename string) bool {
	stat, err := os.Stat(filename)
	if err != nil {
		return false
	}
	return !stat.IsDir()
}
