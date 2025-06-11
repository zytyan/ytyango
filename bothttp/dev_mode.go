package bothttp

import (
	"github.com/gin-gonic/gin"
	"os"
)

var devMode = os.Getenv("DEV_MODE") == "true"

func devModeCheck(ctx *gin.Context) bool {
	if devMode {
		return ctx.GetHeader("X-Zchan-Dev-Mode") == "true"
	}
	return false
}
