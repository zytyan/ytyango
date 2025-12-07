package backend

import (
	"context"

	g "main/globalcfg"
	api "main/http/backend/ogen"
)

func (h *Handler) GetUsersInfo(ctx context.Context, req *api.UserInfoRequest) (api.GetUsersInfoRes, error) {
	if err := req.Validate(); err != nil {
		return &api.GetUsersInfoBadRequest{Message: err.Error()}, nil
	}
	if len(req.UserIds) == 0 {
		return &api.GetUsersInfoBadRequest{Message: "user_ids required"}, nil
	}
	if len(req.UserIds) > 50 {
		return &api.GetUsersInfoBadRequest{Message: "user_ids exceeds 50"}, nil
	}

	users := make([]api.UserInfo, 0, len(req.UserIds))
	for _, userID := range req.UserIds {
		if userID <= 0 {
			return &api.GetUsersInfoBadRequest{Message: "invalid user id"}, nil
		}
		user := g.Q.GetUserById(ctx, userID)
		if user == nil {
			return &api.GetUsersInfoBadRequest{Message: "user not found"}, nil
		}
		users = append(users, api.UserInfo{
			ID:   userID,
			Name: user.Name(),
		})
	}
	resp := &api.UserInfoResponse{Users: users}
	if err := resp.Validate(); err != nil {
		return &api.GetUsersInfoInternalServerError{Message: "failed to validate user info response"}, nil
	}
	return resp, nil
}
