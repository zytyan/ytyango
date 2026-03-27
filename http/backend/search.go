package backend

import (
	"context"
	"database/sql"
	"errors"
	"main/helpers/meilisearch"
	"net/http"
	"strconv"

	g "main/globalcfg"
	"main/handlers"

	"github.com/gin-gonic/gin"
)

type searchQuery struct {
	Query string
	InsID int64
	Page  int
	Limit int
}

type meiliSearchResult struct {
	Hits []handlers.MeiliMsg `json:"hits"`

	Query              string `json:"query"`
	ProcessingTimeMs   int    `json:"processingTimeMs"`
	Limit              int    `json:"limit"`
	Offset             int    `json:"offset"`
	EstimatedTotalHits int    `json:"estimatedTotalHits"`
}

func (h *Handler) handleSearchMessages(c *gin.Context) {
	var req searchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, securityError{ErrorMessage: err.Error()})
		return
	}
	insID, err := strconv.ParseInt(req.InsID, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiError{Message: "invalid ins_id"})
		return
	}
	page := req.Page
	if page <= 0 {
		page = 1
	}
	limit := 20
	if req.Limit != nil {
		limit = *req.Limit
	}

	res, err := h.meiliSearch(c.Request.Context(), searchQuery{
		Query: req.Q,
		InsID: insID,
		Page:  page,
		Limit: limit,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusBadRequest, apiError{Message: "group not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, apiError{Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) meiliSearch(ctx context.Context, query searchQuery) (*searchResult, error) {
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
	var result meiliSearchResult
	err = g.Meili().Search(meiliQ, &result)
	if err != nil {
		return nil, err
	}
	hits := make([]meiliMsg, 0, len(result.Hits))
	for _, hit := range result.Hits {
		hits = append(hits, mapMeiliMsg(hit))
	}
	return &searchResult{
		Hits:               hits,
		Query:              result.Query,
		ProcessingTimeMs:   result.ProcessingTimeMs,
		Limit:              result.Limit,
		Offset:             result.Offset,
		EstimatedTotalHits: result.EstimatedTotalHits,
	}, nil
}

func mapMeiliMsg(src handlers.MeiliMsg) meiliMsg {
	return meiliMsg{
		MongoID:   src.MongoId,
		PeerID:    src.PeerId,
		FromID:    src.FromId,
		MsgID:     src.MsgId,
		Date:      src.Date,
		Message:   src.Message,
		ImageText: src.ImageText,
		QrResult:  src.QrResult,
	}
}
