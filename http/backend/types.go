package backend

type apiError struct {
	Message string `json:"message"`
}

type securityError struct {
	ErrorMessage string `json:"error_message"`
}

type searchRequest struct {
	Q     string `json:"q" binding:"required,min=1"`
	InsID string `json:"ins_id" binding:"required,int64str"`
	Page  int    `json:"page" binding:"required,min=1"`
	Limit *int   `json:"limit,omitempty" binding:"omitempty,min=1,max=50"`
}

type meiliMsg struct {
	MongoID   string  `json:"mongo_id"`
	PeerID    int64   `json:"peer_id"`
	FromID    int64   `json:"from_id"`
	MsgID     int64   `json:"msg_id"`
	Date      float64 `json:"date"`
	Message   string  `json:"message"`
	ImageText string  `json:"image_text"`
	QrResult  string  `json:"qr_result"`
}

type searchResult struct {
	Hits               []meiliMsg `json:"hits"`
	Query              string     `json:"query"`
	ProcessingTimeMs   int        `json:"processingTimeMs"`
	Limit              int        `json:"limit"`
	Offset             int        `json:"offset"`
	EstimatedTotalHits int        `json:"estimatedTotalHits"`
}

type userInfoRequest struct {
	UserIDs []int64 `json:"user_ids" binding:"required,min=1,max=50,dive"`
}

type userInfo struct {
	ID       int64   `json:"id"`
	Name     string  `json:"name"`
	Username *string `json:"username,omitempty"`
	Error    *string `json:"error,omitempty"`
}

type userInfoResponse struct {
	Users []userInfo `json:"users"`
}

type avatarURIParams struct {
	UserID int64 `uri:"userId" binding:"required,min=1"`
}

type avatarQuery struct {
	TgAuth string `form:"tgauth" binding:"required,min=8"`
}

func stringPtr(s string) *string {
	return &s
}
