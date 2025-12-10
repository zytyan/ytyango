package backend

import (
	"context"
	"main/globalcfg/q"

	g "main/globalcfg"
	api "main/http/backend/ogen"
)

func (h *Handler) GetUsersInfo(ctx context.Context, req *api.UserInfoRequest) (api.GetUsersInfoRes, error) {
	users := make([]api.UserInfo, 0, len(req.UserIds))
	for _, userID := range req.UserIds {
		apiUser := api.UserInfo{ID: userID}
		var user *q.User
		if userID <= 0 {
			apiUser.Error = api.NewOptNilString("user id invalid")
			goto addUser
		}
		user = g.Q.GetUserById(ctx, userID)
		if user == nil {
			apiUser.Error = api.NewOptNilString("user not found")
			goto addUser
		}
		apiUser.Name = user.Name()
	addUser:
		users = append(users, apiUser)
	}
	resp := &api.UserInfoResponse{Users: users}
	return resp, nil
}
