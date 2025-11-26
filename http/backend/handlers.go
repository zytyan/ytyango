package backend

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	g "main/globalcfg"
	"main/globalcfg/q"
	"main/http/backend/botapi"

	jsoniter "github.com/json-iterator/go"
	"go.uber.org/zap"
)

type Backend struct {
	log        *zap.SugaredLogger
	httpClient *http.Client
}

// --- auth helpers ---

func (h *Backend) requireAuth(ctx context.Context) (*AuthInfo, *botapi.ErrorResponse) {
	if g.GetConfig().TestMode {
		h.log.Debug("test mode enabled; skipping token verification")
		return &AuthInfo{}, nil
	}
	val := ctx.Value(authContextKey{})
	apiKey, _ := val.(string)
	if apiKey == "" {
		return nil, h.err(ErrNoAuth, "未提供验证信息")
	}
	const telegramPrefix = "Telegram "
	if !strings.HasPrefix(apiKey, telegramPrefix) {
		return nil, h.err(ErrValidFailed, "暂不支持非Telegram验证方式")
	}
	auth, err := checkTelegramAuth(apiKey[len(telegramPrefix):])
	if err != nil {
		return nil, h.err(ErrValidFailed, "验证用户身份失败"+err.Error())
	}
	if time.Since(auth.AuthDate) > 4*time.Hour {
		return nil, h.err(ErrExpired, "数据过期，该网页验证时长已超过4小时，需要重新打开网页验证")
	}
	return &auth, nil
}

// --- helpers ---

func (h *Backend) err(code ErrCode, msg string) *botapi.ErrorResponse {
	return &botapi.ErrorResponse{
		Status: "error",
		Code:   int64(code),
		Error:  msg,
	}
}

func copyTenMinuteStats(src q.TenMinuteStats) botapi.TenMinuteStats {
	out := make(botapi.TenMinuteStats, len(src))
	copy(out, src[:])
	return out
}

func copyUserMsgStatMap(src q.UserMsgStatMap) botapi.UserMsgStatMap {
	if len(src) == 0 {
		return botapi.UserMsgStatMap{}
	}
	dst := make(botapi.UserMsgStatMap, len(src))
	for k, v := range src {
		if v == nil {
			continue
		}
		dst[strconv.FormatInt(k, 10)] = botapi.UserMsgStat{
			MsgCount: v.MsgCount,
			MsgLen:   v.MsgLen,
		}
	}
	return dst
}

func (h *Backend) convertChatStat(stat *q.ChatStat) *botapi.ChatStat {
	if stat == nil {
		return nil
	}
	return &botapi.ChatStat{
		ChatID:             stat.ChatID,
		StatDate:           stat.StatDate,
		MessageCount:       stat.MessageCount,
		PhotoCount:         stat.PhotoCount,
		VideoCount:         stat.VideoCount,
		StickerCount:       stat.StickerCount,
		ForwardCount:       stat.ForwardCount,
		MarsCount:          stat.MarsCount,
		MaxMarsCount:       stat.MaxMarsCount,
		RacyCount:          stat.RacyCount,
		AdultCount:         stat.AdultCount,
		DownloadVideoCount: stat.DownloadVideoCount,
		DownloadAudioCount: stat.DownloadAudioCount,
		DioAddUserCount:    stat.DioAddUserCount,
		DioBanUserCount:    stat.DioBanUserCount,
		UserMsgStat:        copyUserMsgStatMap(stat.UserMsgStat),
		MsgCountByTime:     copyTenMinuteStats(stat.MsgCountByTime),
		MsgIDAtTimeStart:   copyTenMinuteStats(stat.MsgIDAtTimeStart),
	}
}

func (h *Backend) parseLimit(opt botapi.OptInt) int {
	if v, ok := opt.Get(); ok {
		return v
	}
	return 20
}

func sanitizeProfilePhotoFilename(name string) (string, error) {
	clean := filepath.Base(name)
	switch {
	case clean != name:
		return "", errors.New("invalid filename")
	case clean == "", !strings.HasSuffix(clean, ".webp"):
		return "", errors.New("invalid filename")
	default:
		return clean, nil
	}
}

// --- handler implementations ---

func (h *Backend) PingGet(_ context.Context) (*botapi.PingResponse, error) {
	return &botapi.PingResponse{Message: "pong"}, nil
}

func (h *Backend) TgGroupStatGet(ctx context.Context, params botapi.TgGroupStatGetParams) (botapi.TgGroupStatGetRes, error) {
	if _, errResp := h.requireAuth(ctx); errResp != nil {
		res := botapi.TgGroupStatGetUnauthorized(*errResp)
		return &res, nil
	}
	chat, err := g.Q.GetChatByWebId(ctx, params.GroupWebID)
	if err != nil {
		res := botapi.TgGroupStatGetBadRequest(*h.err(GroupNotFound, "group not found"))
		return &res, nil
	}
	stat := g.Q.ChatStatToday(chat.ID)
	if stat == nil {
		res := botapi.TgGroupStatGetBadRequest(*h.err(GroupNotFound, "group not found"))
		return &res, nil
	}
	return h.convertChatStat(stat), nil
}

func (h *Backend) TgProfilePhotoFilenameGet(ctx context.Context, params botapi.TgProfilePhotoFilenameGetParams) (botapi.TgProfilePhotoFilenameGetRes, error) {
	filename, err := sanitizeProfilePhotoFilename(params.Filename)
	if err != nil {
		return h.err(ErrArgInvalid, "invalid filename"), nil
	}
	path := filepath.Join("data/profile_photo", filename)
	fp, err := os.Open(path)
	if err != nil {
		h.log.Warnf("open profile photo %s failed: %v", filename, err)
		return h.err(ErrNoResource, "user profile photo not found"), nil
	}
	return &botapi.TgProfilePhotoFilenameGetOK{Data: fp}, nil
}

func (h *Backend) TgSearchPost(ctx context.Context, req botapi.TgSearchPostReq) (botapi.TgSearchPostRes, error) {
	if _, errResp := h.requireAuth(ctx); errResp != nil {
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
	res, errResp := h.performSearch(ctx, body)
	if errResp != nil {
		br := botapi.TgSearchPostBadRequest(*errResp)
		return &br, nil
	}
	return res, nil
}

func (h *Backend) TgUserinfoPost(ctx context.Context, ids botapi.UserQuery) (botapi.TgUserinfoPostRes, error) {
	if _, errResp := h.requireAuth(ctx); errResp != nil {
		res := botapi.TgUserinfoPostUnauthorized(*errResp)
		return &res, nil
	}
	if len(ids) == 0 {
		res := botapi.TgUserinfoPostBadRequest(*h.err(ErrArgInvalid, "user_id list required"))
		return &res, nil
	}
	user := g.Q.GetUserById(ctx, ids[0])
	if user == nil {
		res := botapi.TgUserinfoPostBadRequest(*h.err(UserNotFound, "user not found"))
		return &res, nil
	}
	return &botapi.User{
		UserID:       user.UserID,
		FirstName:    user.FirstName,
		LastName:     botapi.OptString{Value: user.LastName.String, Set: user.LastName.Valid},
		Username:     botapi.OptString{},
		ProfilePhoto: botapi.OptString{Value: user.ProfilePhoto.String, Set: user.ProfilePhoto.Valid},
	}, nil
}

// --- core logic ---

type meiliQuery struct {
	Q      string   `json:"q"`
	Filter string   `json:"filter"`
	Limit  int      `json:"limit"`
	Offset int      `json:"offset"`
	Sort   []string `json:"sort"`
}

func (h *Backend) performSearch(ctx context.Context, qy botapi.SearchQuery) (*botapi.SearchResult, *botapi.ErrorResponse) {
	chat, err := g.Q.GetChatByWebId(ctx, qy.InsID)
	if err != nil {
		return nil, h.err(GroupNotFound, "group not found")
	}
	limit := h.parseLimit(qy.Limit)
	mq := meiliQuery{
		Q:      qy.Q,
		Filter: "peer_id = " + strconv.FormatInt(chat.ID, 10),
		Limit:  limit,
		Offset: (qy.Page - 1) * limit,
		Sort:   []string{"date:desc"},
	}
	data, err := jsoniter.Marshal(mq)
	if err != nil {
		return nil, h.err(ErrSearchFailed, err.Error())
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.GetConfig().MeiliConfig.GetSearchUrl(), bytes.NewReader(data))
	if err != nil {
		return nil, h.err(ErrSearchFailed, err.Error())
	}
	if mk := g.GetConfig().MeiliConfig.MasterKey; mk != "" {
		req.Header.Set("Authorization", "Bearer "+mk)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, h.err(ErrSearchFailed, err.Error())
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		h.log.Warnf("meili search status=%d body=%s", resp.StatusCode, string(body))
		return nil, h.err(ErrSearchFailed, "search failed")
	}
	var res botapi.SearchResult
	if err := jsoniter.Unmarshal(body, &res); err != nil {
		return nil, h.err(ErrSearchFailed, err.Error())
	}
	if res.Hits == nil {
		res.Hits = make([]botapi.MeiliMsg, 0)
	}
	res.Limit = limit
	res.Offset = (qy.Page - 1) * limit
	return &res, nil
}

// --- error handling ---

func (h *Backend) errorHandler(ctx context.Context, w http.ResponseWriter, r *http.Request, err error) {
	var secErr interface{ Error() string }
	if errors.As(err, &secErr) && strings.Contains(secErr.Error(), "security") {
		h.writeError(w, http.StatusUnauthorized, *h.err(ErrNoAuth, "unauthorized"))
		return
	}
	h.writeError(w, http.StatusBadRequest, *h.err(ErrArgInvalid, err.Error()))
}

func (h *Backend) writeError(w http.ResponseWriter, status int, body botapi.ErrorResponse) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = jsoniter.NewEncoder(w).Encode(body)
}
