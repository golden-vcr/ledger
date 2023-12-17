package subscription

import (
	"context"

	"github.com/golden-vcr/ledger/gen/queries"
	"github.com/google/uuid"
)

type Queries interface {
	RecordSubscriptionInflow(ctx context.Context, arg queries.RecordSubscriptionInflowParams) (uuid.UUID, error)
	RecordGiftSubInflow(ctx context.Context, arg queries.RecordGiftSubInflowParams) (uuid.UUID, error)
}

type TransactionResult struct {
	FlowId uuid.UUID `json:"flowId"`
}
