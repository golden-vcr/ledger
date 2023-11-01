package outflow

import (
	"context"
	"database/sql"

	"github.com/golden-vcr/ledger/gen/queries"
	"github.com/google/uuid"
)

type Queries interface {
	GetBalance(ctx context.Context, twitchUserID string) (queries.GetBalanceRow, error)
	RecordPendingAlertRedemptionOutflow(ctx context.Context, arg queries.RecordPendingAlertRedemptionOutflowParams) (uuid.UUID, error)
	GetFlow(ctx context.Context, flowID uuid.UUID) (queries.GetFlowRow, error)
	FinalizeFlow(ctx context.Context, arg queries.FinalizeFlowParams) (sql.Result, error)
}
