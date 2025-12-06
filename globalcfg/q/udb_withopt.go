package q

import (
	"context"

	"go.uber.org/zap"
)

func PrepareWithLogger(ctx context.Context, db DBTX, logger *zap.Logger) (*Queries, error) {
	query, err := Prepare(ctx, db)
	if err != nil {
		return nil, err
	}
	query.logger = logger
	return query, nil
}
