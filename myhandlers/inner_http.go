package myhandlers

import (
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"main/globalcfg"
	"time"
)

func ytDlpGin(ctx *gin.Context) {
	uid := ctx.GetHeader(ytDlpUuidHeader)
	if uid == "" {
		ctx.AbortWithStatus(400)
		return
	}
	if _, ok := ytDlpMap.Load(uid); !ok {
		ctx.AbortWithStatus(404)
		return
	}
	value := ctx.PostForm("filepath")
	dlp, ok := ytDlpMap.Load(uid)
	if !ok {
		ctx.AbortWithStatus(404)
		return
	}
	dlp.Path = value
	ctx.AbortWithStatus(200)
}

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
	WithGroupLockToday(marsInfo.GroupID, func(g *GroupStatDaily) {
		g.MarsCount++
		g.MaxMarsCount = max(g.MaxMarsCount, marsInfo.MarsCount)
	})
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
	WithGroupLockToday(dioBanUser.GroupId, func(g *GroupStatDaily) {
		switch dioBanUser.Action {
		case DioBanActionAdd:
			g.DioAddUserCount++
		case DioBanActionBanByWrongButton, DioBanActionBanByNoButton, DioBanActionBanByNoMsg:
			g.DioBanUserCount++
		}
	})
}

func HttpListen4019() {
	logger := globalcfg.GetLogger("yt-dlp")
	r := gin.New()
	r.Use(
		ginzap.Ginzap(logger.Desugar(), time.RFC3339, false),
		ginzap.RecoveryWithZap(logger.Desugar(), true),
	)
	r.POST("/yt-dlp", ytDlpGin)
	r.POST("/mars-counter", marsCounter)
	r.POST("/dio-ban", dioBan)
	err := r.Run("127.0.0.1:4019")
	if err != nil {
		log.Fatalf("gin run error %s", err)
	}
}
