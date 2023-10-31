package util

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/golden-vcr/ledger"
	"github.com/google/uuid"
)

func BuildTransaction(id uuid.UUID, flowType string, metadata json.RawMessage, deltaPoints int, createdAt time.Time, finalizedAt sql.NullTime, accepted bool) ledger.Transaction {
	timestamp := createdAt
	state := ledger.TransactionStatePending
	if finalizedAt.Valid {
		timestamp = finalizedAt.Time
		if accepted {
			state = ledger.TransactionStateAccepted
		} else {
			state = ledger.TransactionStateRejected
		}
	}
	return ledger.Transaction{
		Id:          id,
		Timestamp:   timestamp,
		Type:        ledger.TransactionType(flowType),
		State:       state,
		DeltaPoints: int(deltaPoints),
		Description: formatTransactionDescription(flowType, metadata),
	}
}

func formatTransactionDescription(flowType string, metadata json.RawMessage) string {
	if flowType == string(ledger.TransactionTypeManualCredit) {
		s := "Manual credit"
		var md manualCreditMetadata
		if err := json.Unmarshal(metadata, &md); err == nil {
			s += fmt.Sprintf(": %s", md.Note)
		}
		return s
	}
	if flowType == string(ledger.TransactionTypeAlertRedemption) {
		s := "Redeemed alert"
		var md alertRedemptionMetadata
		if err := json.Unmarshal(metadata, &md); err == nil {
			s += fmt.Sprintf(" of type '%s'", md.Type)
		}
		return s
	}
	return ""
}

type manualCreditMetadata struct {
	Note string `json:"note"`
}

type alertRedemptionMetadata struct {
	Type string `json:"type"`
}
