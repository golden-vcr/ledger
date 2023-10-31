package notifications

import (
	"context"
	"encoding/json"
	"time"

	"github.com/golden-vcr/ledger/gen/queries"
	"github.com/google/uuid"
)

type Queries interface {
	StoreSseToken(ctx context.Context, arg queries.StoreSseTokenParams) error
	PurgeSseTokensForUser(ctx context.Context, twitchUserID string) error
	IdentifyUserFromSseToken(ctx context.Context, tokenValue string) (string, error)
}

type FlowChangeNotification struct {
	TwitchUserId string          `json:"twitch_user_id"`
	Id           uuid.UUID       `json:"id"`
	Type         string          `json:"type"`
	Metadata     json.RawMessage `json:"metadata"`
	DeltaPoints  int             `json:"delta_points"`
	CreatedAt    time.Time       `json:"created_at"`
	FinalizedAt  *time.Time      `json:"finalized_at"`
	Accepted     bool            `json:"accepted"`
}
