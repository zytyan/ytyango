package bothttp

import (
	"github.com/gin-gonic/gin"
	"main/myhandlers"
	"strconv"
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
	group := myhandlers.GetGroupInfoUseWebId(groupWebId)
	if group == nil || group.GroupID == 0 {
		log.Warnf("group not found: query:%s", q)
		ctx.JSON(400, GroupNotFound.Msg("group not found"))
		return
	}

	stat, ok := myhandlers.GetGroupStat(group.GroupID)
	if !ok {
		ctx.JSON(400, GroupNotFound.Msg("group not found"))
		return
	}
	ctx.JSON(200, stat)
}
