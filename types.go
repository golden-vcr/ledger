package ledger

import (
	"time"

	"github.com/google/uuid"
)

type TransactionType string

const (
	TransactionTypeManualCredit    TransactionType = "manual-credit"
	TransactionTypeAlertRedemption TransactionType = "alert-redemption"
)

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
	Id          uuid.UUID        `json:"id"`
	Timestamp   time.Time        `json:"timestamp"`
	Type        TransactionType  `json:"type"`
	State       TransactionState `json:"state"`
	DeltaPoints int              `json:"deltaPoints"`
	Description string           `json:"description"`
}
