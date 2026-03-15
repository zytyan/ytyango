package msgs

import (
	"context"
	"log/slog"
)

// PrepareWithLogger prepares statements and attaches a logger to the query set.
func PrepareWithLogger(ctx context.Context, db DBTX, logger *slog.Logger) (*Queries, error) {
	q, err := Prepare(ctx, db)
	if err != nil {
		return nil, err
	}
	return q, nil
}
