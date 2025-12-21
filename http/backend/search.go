package backend

import (
	"context"
	"database/sql"
	"errors"
	"main/helpers/meilisearch"
	"strconv"

	g "main/globalcfg"
	"main/handlers"
	api "main/http/backend/ogen"
)

type searchQuery struct {
	Query string
	InsID int64
	Page  int
	Limit int
}

type searchResult struct {
	Hits []handlers.MeiliMsg `json:"hits"`

	Query              string `json:"query"`
	ProcessingTimeMs   int    `json:"processingTimeMs"`
	Limit              int    `json:"limit"`
	Offset             int    `json:"offset"`
	EstimatedTotalHits int    `json:"estimatedTotalHits"`
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
	limit := query.Limit
	chat, err := g.Q.GetChatByWebId(ctx, query.InsID)
	if err != nil {
		return nil, err
	}
	meiliQ := meilisearch.SearchQuery{
		Q:      query.Query,
		Filter: "peer_id = " + strconv.FormatInt(chat.ID, 10),
		Limit:  limit,
		Offset: (query.Page - 1) * limit,
		Sort:   []string{"date:desc"},
	}
	var result searchResult
	err = g.MeiliClient.Search(meiliQ, &result)
	if err != nil {
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

func mapMeiliMsg(src handlers.MeiliMsg) api.MeiliMsg {
	return api.MeiliMsg{
		MongoID:   src.MongoId,
		PeerID:    src.PeerId,
		FromID:    src.FromId,
		MsgID:     src.MsgId,
		Date:      src.Date,
		Message:   api.NewOptString(src.Message),
		ImageText: api.NewOptString(src.ImageText),
		QrResult:  api.NewOptString(src.QrResult),
	}
}
