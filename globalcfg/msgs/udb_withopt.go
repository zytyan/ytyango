package msgs

import (
	"context"

	"go.uber.org/zap"
)

// PrepareWithLogger prepares statements and attaches a logger to the query set.
func PrepareWithLogger(ctx context.Context, db DBTX, logger *zap.Logger) (*Queries, error) {
	q, err := Prepare(ctx, db)
	if err != nil {
		return nil, err
	}
	q.logger = logger
	return q, nil
}
