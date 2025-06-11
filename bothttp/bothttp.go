package bothttp

import (
	"crypto/hmac"
	"crypto/sha256"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"main/globalcfg"
	"net/http"
	"time"
)

var botVerifyKey = func() []byte {
	var key = globalcfg.GetConfig().BotToken
	mac := hmac.New(sha256.New, []byte("WebAppData"))
	mac.Write([]byte(key))
	return mac.Sum(nil)
}()

func ping(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}

var log *zap.SugaredLogger

func Run() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	err := r.SetTrustedProxies(nil)
	if err != nil {
		panic(err)
		return
	}
	logger := globalcfg.GetLogger("bot-http").Desugar()
	log = logger.Sugar()
	r.Use(
		ginzap.Ginzap(logger, time.RFC3339, false),
		ginzap.RecoveryWithZap(logger, true),
	)

	r.GET("/api/ping", ping)
	g := r.Group("/api/v1/tg")
	g.GET("/username", reqBrowserCache, getUserInfo)
	g.GET("/profile_photo", reqBrowserCache, getUserProfilePhoto)
	g.GET("/group_stat", verifyHeader, groupStat)
	g.Match([]string{http.MethodGet, http.MethodPost},
		"/search", verifyHeader, searchMessage)
	err = r.Run("127.0.0.1:4021")
	if err != nil {
		panic(err)
	}
}
