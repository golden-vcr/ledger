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
	TransactionTypeSubscription    TransactionType = "subscription"
	TransactionTypeGiftSub         TransactionType = "gift-sub"
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

// SubscriptionRequest is the payload sent with a POST /inflow/subscription request
type SubscriptionRequest struct {
	// BasePointsToCredit is the number of points that a user should receive for
	// purchasing a Tier 1 subscription for a single month
	BasePointsToCredit int `json:"basePointsToCredit"`
	// IsInitial indicates that points are being credit for an initial subscription
	// purchase, as opposed to a subscription renewal / resub message
	IsInitial bool `json:"isInitial"`
	// IsGift is true if the target user received the subscription as a gift from
	// another user, as opposed to purchasing it themself
	IsGift bool `json:"isGift"`
	// Message indicates the resub message sent with the originating event, if the event
	// was a resub and the user provided a message. This value may always be empty.
	Message string `json:"message"`
	// CreditMulitplier is an additional scale factor applied based on the Tier of the
	// subscription purchased; e.g. 2.0 for a Tier 2 sub, 5.0 for a Tier 3 sub
	CreditMultiplier int `json:"creditMultiplier"`
}

// GiftSubRequest is the payload sent with a POST /inflow/gift-sub request
type GiftSubRequest struct {
	// BasePointsToCredit is the number of points that a user should receive for
	// purchasing a single Tier 1 gift sub and granting it to a recipient
	BasePointsToCredit int `json:"basePointsToCredit"`
	// NumSubscriptions indicates the number of subscriptions gifted, used as a scale
	// factor to compute the number of points to credit for this transaction
	// (i.e. BasePointsToCredit * NumSubscriptions)
	NumSubscriptions int `json:"numSubscriptions"`
	// CreditMulitplier is an additional scale factor applied based on the Tier of the
	// subscriptions gifted; e.g. 2.0 for a Tier 2 sub, 5.0 for a Tier 3 sub
	CreditMultiplier float64 `json:"creditMultiplier"`
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
