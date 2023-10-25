package records

import (
	"encoding/json"
	"fmt"

	"github.com/golden-vcr/ledger"
	"github.com/golden-vcr/ledger/gen/queries"
)

func buildHistoryItem(row *queries.GetTransactionHistoryRow) Transaction {
	timestamp := row.CreatedAt
	state := TransactionStatePending
	if row.FinalizedAt.Valid {
		timestamp = row.FinalizedAt.Time
		if row.Accepted {
			state = TransactionStateAccepted
		} else {
			state = TransactionStateRejected
		}
	}
	return Transaction{
		Id:          row.ID,
		Timestamp:   timestamp,
		Type:        ledger.TransactionType(row.Type),
		State:       state,
		DeltaPoints: int(row.DeltaPoints),
		Description: formatTransactionDescription(row),
	}
}

func formatTransactionDescription(row *queries.GetTransactionHistoryRow) string {
	if row.Type == string(ledger.TransactionTypeManualCredit) {
		s := "Manual credit"
		var md manualCreditMetadata
		if err := json.Unmarshal(row.Metadata, &md); err == nil {
			s += fmt.Sprintf(": %s", md.Note)
		}
		return s
	}
	if row.Type == string(ledger.TransactionTypeAlertRedemption) {
		s := "Redeemed alert"
		var md alertRedemptionMetadata
		if err := json.Unmarshal(row.Metadata, &md); err == nil {
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
