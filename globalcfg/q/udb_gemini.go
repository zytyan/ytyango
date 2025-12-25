package q

import "context"

// GeminiContentV2WithParts bundles a content row with its ordered parts.
type GeminiContentV2WithParts struct {
	Content GeminiContentV2
	Parts   []GeminiContentV2Part
}

// ListGeminiHistory loads v2 contents (ascending seq) with their parts.
func (q *Queries) ListGeminiHistory(ctx context.Context, sessionID int64, limit int64) ([]GeminiContentV2WithParts, error) {
	rows, err := q.ListGeminiContentV2(ctx, sessionID, limit)
	if err != nil {
		return nil, err
	}
	out := make([]GeminiContentV2WithParts, len(rows))
	for i := range rows {
		parts, err := q.ListGeminiContentV2Parts(ctx, rows[i].ID)
		if err != nil {
			return nil, err
		}
		out[i] = GeminiContentV2WithParts{
			Content: rows[i],
			Parts:   parts,
		}
	}
	return out, nil
}

// NextGeminiSeq returns the next available seq for the session.
func (q *Queries) NextGeminiSeq(ctx context.Context, sessionID int64) (int64, error) {
	return q.GetNextGeminiSeq(ctx, sessionID)
}
