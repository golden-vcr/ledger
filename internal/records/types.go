package records

import (
	"context"

	"github.com/golden-vcr/ledger/gen/queries"
)

type Queries interface {
	GetBalance(ctx context.Context, twitchUserID string) (queries.GetBalanceRow, error)
	GetTransactionHistory(ctx context.Context, arg queries.GetTransactionHistoryParams) ([]queries.GetTransactionHistoryRow, error)
}
