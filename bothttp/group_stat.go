package bothttp

import (
	"context"
	g "main/globalcfg"
	"main/myhandlers"
	"strconv"

	"github.com/gin-gonic/gin"
)

func groupStat(ctx *gin.Context) {
	q, ok := ctx.GetQuery("group_web_id")
	if !ok {
		ctx.JSON(400, ErrArgInvalid.Msg("group_web_id not found"))
		return
	}
	groupWebId, err := strconv.ParseInt(q, 10, 64)
	if err != nil || groupWebId == 0 {
		log.Warnf("group_id is invalid: query:%s, err:%s", q, err)
		ctx.JSON(400, ErrArgInvalid.Msg("group_web_id is invalid"))
		return
	}
	group, err := g.Q.GetChatByWebId(context.Background(), groupWebId)
	if err != nil {
		log.Warnf("group not found: query:%s", q)
		ctx.JSON(400, GroupNotFound.Msg("group not found"))
		return
	}

	stat, ok := myhandlers.GetGroupStat(group.ID)
	if !ok {
		ctx.JSON(400, GroupNotFound.Msg("group not found"))
		return
	}
	ctx.JSON(200, stat)
}
