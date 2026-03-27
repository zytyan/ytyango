package backend

import (
	"main/globalcfg/q"
	"net/http"

	g "main/globalcfg"

	"github.com/gin-gonic/gin"
)

func (h *Handler) handleGetUsersInfo(c *gin.Context) {
	var req userInfoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, securityError{ErrorMessage: err.Error()})
		return
	}

	users := make([]userInfo, 0, len(req.UserIDs))
	for _, userID := range req.UserIDs {
		apiUser := userInfo{ID: userID}
		var user *q.User
		var err error
		if userID <= 0 {
			apiUser.Error = stringPtr("user id invalid")
			goto addUser
		}
		user, err = g.Q.GetUserById(c.Request.Context(), userID)
		if err != nil {
			apiUser.Error = stringPtr("user not found")
			goto addUser
		}
		apiUser.Name = user.Name()
	addUser:
		users = append(users, apiUser)
	}
	c.JSON(http.StatusOK, userInfoResponse{Users: users})
}
