package admin

import (
	"context"

	"github.com/golden-vcr/ledger/gen/queries"
	"github.com/google/uuid"
)

type Queries interface {
	RecordManualCreditInflow(ctx context.Context, arg queries.RecordManualCreditInflowParams) (uuid.UUID, error)
}

type ManualCreditRequest struct {
	TwitchUserId      string `json:"twitchUserId,omitempty"`
	TwitchDisplayName string `json:"twitchDisplayName,omitempty"`
	NumPointsToCredit int    `json:"numPointsToCredit"`
	Note              string `json:"note"`
}

type TransactionResult struct {
	FlowId uuid.UUID `json:"flowId"`
}
