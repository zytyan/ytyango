package backend

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	g "main/globalcfg"
	api "main/http/backend/ogen"
	"main/myhandlers"

	jsoniter "github.com/json-iterator/go"
	"go.uber.org/zap"
)

type searchQuery struct {
	Query string
	InsID int64
	Page  int
	Limit int
}

type searchResult struct {
	Hits []myhandlers.MeiliMsg `json:"hits"`

	Query              string `json:"query"`
	ProcessingTimeMs   int    `json:"processingTimeMs"`
	Limit              int    `json:"limit"`
	Offset             int    `json:"offset"`
	EstimatedTotalHits int    `json:"estimatedTotalHits"`
}

type meiliQuery struct {
	Q      string   `json:"q"`
	Filter string   `json:"filter"`
	Limit  int      `json:"limit"`
	Offset int      `json:"offset"`
	Sort   []string `json:"sort"`
}

func (h *Handler) SearchMessages(ctx context.Context, req *api.SearchRequest) (api.SearchMessagesRes, error) {
	insID, err := strconv.ParseInt(req.InsID, 10, 64)
	if err != nil {
		return &api.SearchMessagesBadRequest{Message: "invalid ins_id"}, nil
	}
	page := int(req.Page)
	if page <= 0 {
		page = 1
	}
	limit := int(req.Limit.Or(20))

	res, err := h.meiliSearch(ctx, searchQuery{
		Query: req.Q,
		InsID: insID,
		Page:  page,
		Limit: limit,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &api.SearchMessagesBadRequest{Message: "group not found"}, nil
		}
		return &api.SearchMessagesInternalServerError{Message: err.Error()}, nil
	}
	return res, nil
}

func (h *Handler) meiliSearch(ctx context.Context, query searchQuery) (*api.SearchResult, error) {
	searchURL := g.GetConfig().MeiliConfig.GetSearchUrl()
	chat, err := g.Q.GetChatByWebId(ctx, query.InsID)
	if err != nil {
		return nil, err
	}
	limit := query.Limit
	meiliPayload := meiliQuery{
		Q:      query.Query,
		Filter: "peer_id = " + strconv.FormatInt(chat.ID, 10),
		Limit:  limit,
		Offset: (query.Page - 1) * limit,
		Sort:   []string{"date:desc"},
	}
	data, err := jsoniter.Marshal(meiliPayload)
	if err != nil {
		return nil, err
	}
	preparedPost, err := http.NewRequestWithContext(ctx, http.MethodPost, searchURL, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if token := g.GetConfig().MeiliConfig.MasterKey; token != "" {
		preparedPost.Header.Set("Authorization", "Bearer "+token)
	}
	preparedPost.Header.Set("Content-Type", "application/json")
	post, err := http.DefaultClient.Do(preparedPost)
	if err != nil {
		return nil, err
	}
	defer post.Body.Close()

	body, err := io.ReadAll(post.Body)
	if err != nil {
		return nil, err
	}
	if post.StatusCode < 200 || post.StatusCode >= 300 {
		h.log.Error("status code not ok",
			zap.Int("status_code", post.StatusCode),
			zap.ByteString("body", body),
		)
		return nil, fmt.Errorf("search request failed: status=%d", post.StatusCode)
	}

	var result searchResult
	if err := jsoniter.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	hits := make([]api.MeiliMsg, 0, len(result.Hits))
	for _, hit := range result.Hits {
		hits = append(hits, mapMeiliMsg(hit))
	}

	return &api.SearchResult{
		Hits:               hits,
		Query:              result.Query,
		ProcessingTimeMs:   int32(result.ProcessingTimeMs),
		Limit:              int32(result.Limit),
		Offset:             int32(result.Offset),
		EstimatedTotalHits: int32(result.EstimatedTotalHits),
	}, nil
}

func mapMeiliMsg(src myhandlers.MeiliMsg) api.MeiliMsg {
	return api.MeiliMsg{
		MongoID:   api.NewOptString(src.MongoId),
		PeerID:    api.NewOptInt64(src.PeerId),
		FromID:    api.NewOptInt64(src.FromId),
		MsgID:     api.NewOptInt64(src.MsgId),
		Date:      api.NewOptFloat64(src.Date),
		Message:   api.NewOptString(src.Message),
		ImageText: api.NewOptString(src.ImageText),
		QrResult:  api.NewOptString(src.QrResult),
	}
}
