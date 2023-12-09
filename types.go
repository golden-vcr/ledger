package ledger

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type TransactionType string

const (
	TransactionTypeManualCredit    TransactionType = "manual-credit"
	TransactionTypeCheer           TransactionType = "cheer"
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

type CheerRequest struct {
	NumPointsToCredit int    `json:"numPointsToCredit"`
	Message           string `json:"message"`
}

type AlertRedemptionRequest struct {
	Type             TransactionType  `json:"type"`
	NumPointsToDebit int              `json:"numPointsToDebit"`
	AlertType        string           `json:"alertType"`
	AlertMetadata    *json.RawMessage `json:"alertMetadata,omitempty"`
}

type TransactionResult struct {
	FlowId uuid.UUID `json:"flowId"`
}
