package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	g "main/globalcfg"
	"main/http/backend/botapi"

	"go.uber.org/zap"
)

// FixtureData mirrors the on-disk JSON stored under http/backend/testdata.
// It deliberately matches the API schema so that front-end developers can
// replay consistent responses without connecting to real services.
type FixtureData struct {
	BotToken      string         `json:"bot_token"`
	AuthInitData  string         `json:"auth_init_data"`
	Groups        []FixtureGroup `json:"groups"`
	Users         []FixtureUser  `json:"users"`
	ProfilePhotos []FixturePhoto `json:"profile_photos"`

	baseDir string `json:"-"`
}

type FixtureGroup struct {
	GroupWebID   int64               `json:"group_web_id"`
	ChatID       int64               `json:"chat_id"`
	Stat         botapi.ChatStat     `json:"stat"`
	SearchResult botapi.SearchResult `json:"search_result"`
}

type FixtureUser struct {
	UserID       int64  `json:"user_id"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Username     string `json:"username"`
	ProfilePhoto string `json:"profile_photo"`
}

type FixturePhoto struct {
	Filename string `json:"filename"`
	Path     string `json:"path"`
}

// LoadFixtureData loads the JSON fixture file and remembers its base path for
// resolving relative assets such as profile photos.
func LoadFixtureData(path string) (FixtureData, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return FixtureData{}, err
	}
	var fixtures FixtureData
	if err := json.Unmarshal(data, &fixtures); err != nil {
		return FixtureData{}, err
	}
	fixtures.baseDir = filepath.Dir(path)
	return fixtures, nil
}

// FixtureBackend implements the OpenAPI handler with static responses.
type FixtureBackend struct {
	log          *zap.SugaredLogger
	expectedAuth string
	groupMap     map[int64]FixtureGroup
	userMap      map[int64]FixtureUser
	photoData    map[string][]byte
}

// NewFixtureBackend builds an in-memory handler from fixture data.
func NewFixtureBackend(fixtures FixtureData, expectedAuth string) (*FixtureBackend, error) {
	photos := make(map[string][]byte, len(fixtures.ProfilePhotos))
	for _, p := range fixtures.ProfilePhotos {
		filePath := p.Path
		if !filepath.IsAbs(filePath) {
			filePath = filepath.Join(fixtures.baseDir, filePath)
		}
		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("load profile photo %s: %w", p.Filename, err)
		}
		photos[p.Filename] = content
	}

	groupMap := make(map[int64]FixtureGroup, len(fixtures.Groups))
	for _, g := range fixtures.Groups {
		groupMap[g.GroupWebID] = g
	}
	userMap := make(map[int64]FixtureUser, len(fixtures.Users))
	for _, u := range fixtures.Users {
		userMap[u.UserID] = u
	}

	logger := g.GetLogger("http-backend-fixture")
	return &FixtureBackend{
		log:          logger,
		expectedAuth: expectedAuth,
		groupMap:     groupMap,
		userMap:      userMap,
		photoData:    photos,
	}, nil
}

// --- shared helpers ---

func (h *FixtureBackend) err(code ErrCode, msg string) *botapi.ErrorResponse {
	return &botapi.ErrorResponse{
		Status: "error",
		Code:   int64(code),
		Error:  msg,
	}
}

func (h *FixtureBackend) errorHandler(ctx context.Context, w http.ResponseWriter, r *http.Request, err error) {
	var secErr interface{ Error() string }
	if errors.As(err, &secErr) && strings.Contains(secErr.Error(), "security") {
		h.writeError(w, http.StatusUnauthorized, *h.err(ErrNoAuth, "unauthorized"))
		return
	}
	h.writeError(w, http.StatusBadRequest, *h.err(ErrArgInvalid, err.Error()))
}

func (h *FixtureBackend) writeError(w http.ResponseWriter, status int, body botapi.ErrorResponse) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func (h *FixtureBackend) parseLimit(opt botapi.OptInt) int {
	if v, ok := opt.Get(); ok {
		return v
	}
	return 20
}

// verifyAuth validates the Authorization header using the standard
// checkTelegramAuth logic but keeps the fixture sample fresh by ignoring
// auth_date expiry. When expectedAuth is non-empty, the raw initData must
// match exactly to pass.
func (h *FixtureBackend) verifyAuth(ctx context.Context) (*AuthInfo, *botapi.ErrorResponse) {
	val := ctx.Value(authContextKey{})
	apiKey, _ := val.(string)
	if apiKey == "" {
		return nil, h.err(ErrNoAuth, "未提供验证信息")
	}
	const telegramPrefix = "Telegram "
	if !strings.HasPrefix(apiKey, telegramPrefix) {
		return nil, h.err(ErrValidFailed, "暂不支持非Telegram验证方式")
	}
	raw := apiKey[len(telegramPrefix):]
	if h.expectedAuth != "" && raw != h.expectedAuth {
		return nil, h.err(ErrValidFailed, "验证数据与fixtures不匹配")
	}
	auth, err := ParseTelegramAuth(raw)
	if err != nil {
		return nil, h.err(ErrValidFailed, "验证用户身份失败"+err.Error())
	}
	return &auth, nil
}

// --- handler implementations ---

func (h *FixtureBackend) PingGet(_ context.Context) (*botapi.PingResponse, error) {
	return &botapi.PingResponse{Message: "pong-fixture"}, nil
}

func (h *FixtureBackend) TgGroupStatGet(ctx context.Context, params botapi.TgGroupStatGetParams) (botapi.TgGroupStatGetRes, error) {
	if _, errResp := h.verifyAuth(ctx); errResp != nil {
		res := botapi.TgGroupStatGetUnauthorized(*errResp)
		return &res, nil
	}
	group, ok := h.groupMap[params.GroupWebID]
	if !ok {
		res := botapi.TgGroupStatGetBadRequest(*h.err(GroupNotFound, "group not found in fixtures"))
		return &res, nil
	}
	stat := group.Stat
	return &stat, nil
}

func (h *FixtureBackend) TgProfilePhotoFilenameGet(_ context.Context, params botapi.TgProfilePhotoFilenameGetParams) (botapi.TgProfilePhotoFilenameGetRes, error) {
	filename, err := sanitizeProfilePhotoFilename(params.Filename)
	if err != nil {
		return h.err(ErrArgInvalid, "invalid filename"), nil
	}
	data, ok := h.photoData[filename]
	if !ok {
		return h.err(ErrNoResource, "user profile photo not found"), nil
	}
	return &botapi.TgProfilePhotoFilenameGetOK{Data: bytes.NewReader(data)}, nil
}

func (h *FixtureBackend) TgSearchPost(ctx context.Context, req botapi.TgSearchPostReq) (botapi.TgSearchPostRes, error) {
	if _, errResp := h.verifyAuth(ctx); errResp != nil {
		res := botapi.TgSearchPostUnauthorized(*errResp)
		return &res, nil
	}
	var body botapi.SearchQuery
	switch v := req.(type) {
	case *botapi.TgSearchPostApplicationJSON:
		body = botapi.SearchQuery(*v)
	case *botapi.TgSearchPostApplicationXWwwFormUrlencoded:
		body = botapi.SearchQuery(*v)
	default:
		br := botapi.TgSearchPostBadRequest(*h.err(ErrArgInvalid, "unsupported content type"))
		return &br, nil
	}
	group, ok := h.groupMap[body.InsID]
	if !ok {
		br := botapi.TgSearchPostBadRequest(*h.err(GroupNotFound, "group not found in fixtures"))
		return &br, nil
	}
	result := group.SearchResult
	// override values according to the incoming query for realism
	result.Query = body.Q
	limit := h.parseLimit(body.Limit)
	result.Limit = limit
	result.Offset = (body.Page - 1) * limit
	return &result, nil
}

func (h *FixtureBackend) TgUserinfoPost(ctx context.Context, ids botapi.UserQuery) (botapi.TgUserinfoPostRes, error) {
	if _, errResp := h.verifyAuth(ctx); errResp != nil {
		res := botapi.TgUserinfoPostUnauthorized(*errResp)
		return &res, nil
	}
	if len(ids) == 0 {
		res := botapi.TgUserinfoPostBadRequest(*h.err(ErrArgInvalid, "user_id list required"))
		return &res, nil
	}
	user, ok := h.userMap[ids[0]]
	if !ok {
		res := botapi.TgUserinfoPostBadRequest(*h.err(UserNotFound, "user not found in fixtures"))
		return &res, nil
	}
	return &botapi.User{
		UserID:       user.UserID,
		FirstName:    user.FirstName,
		LastName:     botapi.OptString{Value: user.LastName, Set: user.LastName != ""},
		Username:     botapi.OptString{Value: user.Username, Set: user.Username != ""},
		ProfilePhoto: botapi.OptString{Value: user.ProfilePhoto, Set: user.ProfilePhoto != ""},
	}, nil
}
