package cheer

import (
	"context"

	"github.com/golden-vcr/ledger/gen/queries"
	"github.com/google/uuid"
)

type Queries interface {
	RecordCheerInflow(ctx context.Context, arg queries.RecordCheerInflowParams) (uuid.UUID, error)
}

type TransactionResult struct {
	FlowId uuid.UUID `json:"flowId"`
}
