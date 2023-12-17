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
	if flowType == string(ledger.TransactionTypeCheer) {
		s := "Thank you for cheering"
		var md cheerMetadata
		if err := json.Unmarshal(metadata, &md); err == nil && md.Message != "" {
			s += fmt.Sprintf(" with the message '%s'", md.Message)
		}
		s += "!"
		return s
	}
	if flowType == string(ledger.TransactionTypeSubscription) {
		var md subscriptionMetadata
		if err := json.Unmarshal(metadata, &md); err != nil {
			return "Thank you for being a subscriber!"
		}
		s := ""
		if md.IsGift {
			s = "You received a gift sub"
		} else if md.IsInitial {
			s = "Thank you for becoming a subscriber"
		} else {
			s = "Thank you for renewing your subscription"
		}
		if md.CreditMultiplier > 1.001 {
			s += fmt.Sprintf(" (at a tier with %.fx credit)", md.CreditMultiplier)
		}
		if md.Message != "" {
			s += fmt.Sprintf(" with the message '%s'", md.Message)
		}
		s += "!"
		return s
	}
	if flowType == string(ledger.TransactionTypeGiftSub) {
		var md giftSubMetadata
		if err := json.Unmarshal(metadata, &md); err != nil {
			return "Thank you for gifting subs!"
		}
		s := ""
		if md.NumSubscriptions == 1 {
			s = "Thank you for gifting a sub"
		} else {
			s = fmt.Sprintf("Thank you for gifting %d subs", md.NumSubscriptions)
		}
		if md.CreditMultiplier > 1.001 {
			s += fmt.Sprintf(" (at a tier with %.fx credit)", md.CreditMultiplier)
		}
		s += "!"
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

type cheerMetadata struct {
	Message string `json:"message"`
}

type subscriptionMetadata struct {
	Message          string  `json:"message"`
	IsInitial        bool    `json:"is_initial"`
	IsGift           bool    `json:"is_gift"`
	CreditMultiplier float64 `json:"credit_multiplier"`
}

type giftSubMetadata struct {
	NumSubscriptions int     `json:"num_subscriptions"`
	CreditMultiplier float64 `json:"credit_multiplier"`
}
