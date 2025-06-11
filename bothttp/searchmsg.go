package bothttp

import (
	"bytes"
	"errors"
	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	"io"
	"main/globalcfg"
	"main/myhandlers"
	"net/http"
	"strconv"
)

type SearchQuery struct {
	Query string    `json:"q" form:"q" binding:"required"`
	InsID JsonInt64 `json:"ins_id" form:"ins_id" binding:"required"`
	Page  int       `json:"page" form:"page" binding:"required"`
	Limit int       `json:"limit,omitempty" form:"limit,omitempty"`
}

func (s *SearchQuery) GetLimit() int {
	if s.Limit <= 0 {
		return 20
	}
	if s.Limit > 50 {
		return 50
	}
	return s.Limit
}

type SearchResult struct {
	Hits []myhandlers.MeiliMsg `json:"hits"`

	Query              string `json:"query"`
	ProcessingTimeMs   int    `json:"processingTimeMs"`
	Limit              int    `json:"limit"`
	Offset             int    `json:"offset"`
	EstimatedTotalHits int    `json:"estimatedTotalHits"`
}

type MeiliQuery struct {
	Q      string   `json:"q"`
	Filter string   `json:"filter"`
	Limit  int      `json:"limit"`
	Offset int      `json:"offset"`
	Sort   []string `json:"sort"`
}

func meiliSearch(query SearchQuery) (SearchResult, error) {
	var result SearchResult
	searchUrl := globalcfg.GetConfig().MeiliConfig.GetSearchUrl()
	g := myhandlers.GetGroupInfoUseWebId(int64(query.InsID))
	if g == nil {
		return result, GroupNotFound
	}
	limit := query.GetLimit()
	meiliQuery := MeiliQuery{
		Q:      query.Query,
		Filter: "peer_id = " + strconv.FormatInt(g.GroupID, 10),
		Limit:  limit,
		Offset: (query.Page - 1) * limit,
		Sort:   []string{"date:desc"},
	}
	data, err := jsoniter.Marshal(meiliQuery)
	if err != nil {
		return result, err
	}
	preparedPost, err := http.NewRequest(http.MethodPost, searchUrl, bytes.NewReader(data))
	if err != nil {
		return result, err
	}
	preparedPost.Header.Set("Authorization", "Bearer "+globalcfg.GetConfig().MeiliConfig.MasterKey)
	preparedPost.Header.Set("Content-Type", "application/json")
	post, err := http.DefaultClient.Do(preparedPost)
	if err != nil {
		return result, err
	}
	data, _ = io.ReadAll(post.Body)
	log.Infof("search result %s", data)
	err = jsoniter.Unmarshal(data, &result)
	if err != nil {
		return result, err
	}
	return result, nil
}

func searchMessage(ctx *gin.Context) {
	var searchQuery SearchQuery
	err := ctx.ShouldBind(&searchQuery)
	if err != nil {
		ctx.JSON(400, ErrArgInvalid.Msg(err.Error()))
		return
	}
	result, err := meiliSearch(searchQuery)
	if err != nil {
		if errors.Is(err, GroupNotFound) {
			ctx.JSON(400, GroupNotFound.Msg("group not found"))
			return
		}
		ctx.JSON(400, ErrSearchFailed.Msg(err.Error()))
		return
	}
	ctx.JSON(200, result)
}
