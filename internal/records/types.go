package records

import (
	"context"
	"time"

	"github.com/golden-vcr/ledger"
	"github.com/golden-vcr/ledger/gen/queries"
	"github.com/google/uuid"
)

type Queries interface {
	GetBalance(ctx context.Context, twitchUserID string) (queries.GetBalanceRow, error)
	GetTransactionHistory(ctx context.Context, arg queries.GetTransactionHistoryParams) ([]queries.GetTransactionHistoryRow, error)
}

type TransactionState string

const (
	TransactionStatePending  TransactionState = "pending"
	TransactionStateAccepted TransactionState = "accepted"
	TransactionStateRejected TransactionState = "rejected"
)

type Balance struct {
	TotalPoints     int `json:"totalPoints"`
	AvailablePoints int `json:"availablePoints"`
}

type TransactionHistory struct {
	Items      []Transaction `json:"items"`
	NextCursor string        `json:"nextCursor,omitempty"`
}

type Transaction struct {
	Id          uuid.UUID              `json:"id"`
	Timestamp   time.Time              `json:"timestamp"`
	Type        ledger.TransactionType `json:"type"`
	State       TransactionState       `json:"state"`
	DeltaPoints int                    `json:"deltaPoints"`
	Description string                 `json:"description"`
}
