package q

import (
	"context"
)

func (y *YtDlResult) Save(ctx context.Context, q *Queries) error {
	return q.UpdateYtDlpCache(ctx, UpdateYtDlpCacheParams{
		Url:         y.Url,
		AudioOnly:   y.AudioOnly,
		Resolution:  y.Resolution,
		FileID:      y.FileID,
		Title:       y.Title,
		Description: y.Description,
		Uploader:    y.Uploader,
	})
}
