package myhandlers

import (
	"fmt"
	g "main/globalcfg"
	"strconv"
	"strings"
	"time"

	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap/zapcore"
)

type MarsInfo struct {
	GroupID   int64 `json:"group_id"`
	MarsCount int64 `json:"mars_count"`
}

func marsCounter(ctx *gin.Context) {
	var marsInfo MarsInfo
	err := ctx.Bind(&marsInfo)

	if err != nil {
		ctx.AbortWithStatus(400)
		return
	}
	g.Q.ChatStatToday(marsInfo.GroupID).IncMarsCount(marsInfo.MarsCount)

}

const (
	DioBanActionAdd = iota
	DioBanActionBanByWrongButton
	DioBanActionBanByNoButton
	DioBanActionBanByNoMsg
)

type DioBanUser struct {
	UserId  int64 `json:"user_id"`
	GroupId int64 `json:"group_id"`
	Action  int   `json:"action"`
}

func dioBan(ctx *gin.Context) {
	var dioBanUser DioBanUser
	err := ctx.Bind(&dioBanUser)
	if err != nil {
		ctx.AbortWithStatus(400)
		return
	}
	switch dioBanUser.Action {
	case DioBanActionAdd:
		g.Q.ChatStatToday(dioBanUser.GroupId).IncDioAddUserCount()
	case DioBanActionBanByWrongButton, DioBanActionBanByNoButton, DioBanActionBanByNoMsg:
		g.Q.ChatStatToday(dioBanUser.GroupId).IncDioBanUserCount()
	}

}
func formatLoggers() string {
	buf := strings.Builder{}
	for name, logger := range g.GetAllLoggers() {
		level := logger.Level.Level()
		buf.WriteString(
			fmt.Sprintf("%-16s\t[%d]%s\n", name, level, level.String()),
		)
	}
	return buf.String()
}
func showLoggers(ctx *gin.Context) {
	ctx.Writer.WriteHeader(200)
	ctx.Writer.Header().Set("Content-Type", "text/plain")
	_, err := ctx.Writer.WriteString(formatLoggers())
	if err != nil {
		_ = ctx.Error(err)
	}
}
func setLoggerLevel(ctx *gin.Context) {
	loggerName := ctx.Params.ByName("name")

	logger, name := g.GetAllLoggers()[loggerName]
	if !name {
		_, _ = ctx.Writer.WriteString(fmt.Sprintf("logger %s not found\n%s", loggerName, formatLoggers()))
		return
	}
	newLevelS := ctx.Params.ByName("level")
	newLevel, err := strconv.ParseInt(newLevelS, 10, 8)
	if err != nil {
		_, _ = ctx.Writer.WriteString(err.Error())
	}
	logger.Level.SetLevel(zapcore.Level(newLevel))
}

func listAllRoutes(ctx *gin.Context) {
	_, _ = ctx.Writer.WriteString("GET /loggers\nPUT /loggers/<name>/<:level,int8>\n")
}

func HttpListen4019() {
	logger := g.GetLogger("yt-dlp")
	r := gin.New()
	r.Use(
		ginzap.Ginzap(logger.Desugar(), time.RFC3339, false),
		ginzap.RecoveryWithZap(logger.Desugar(), true),
	)
	r.POST("/mars-counter", marsCounter)
	r.POST("/dio-ban", dioBan)
	r.GET("/loggers", showLoggers)
	r.PUT("/loggers/:name/:level", setLoggerLevel)
	r.Any("/", listAllRoutes)

	err := r.Run("127.0.0.1:4019")
	if err != nil {
		log.Fatalf("gin run error %s", err)
	}
}
