package queries_test

import (
	"context"
	"testing"

	"github.com/golden-vcr/ledger/gen/queries"
	"github.com/golden-vcr/server-common/querytest"
	"github.com/sqlc-dev/pqtype"
	"github.com/stretchr/testify/assert"
)

func Test_RecordPendingAlertRedemptionOutflow(t *testing.T) {
	tx := querytest.PrepareTx(t)
	q := queries.New(tx)

	querytest.AssertCount(t, tx, 0, "SELECT COUNT(*) FROM ledger.flow")

	flowUuid, err := q.RecordPendingAlertRedemptionOutflow(context.Background(), queries.RecordPendingAlertRedemptionOutflowParams{
		TwitchUserID: "4444",
		AlertType:    "foo",
		AlertMetadata: pqtype.NullRawMessage{
			Valid:      true,
			RawMessage: []byte(`{"bar":"baz"}`),
		},
		NumPointsToDebit: 350,
	})
	assert.NoError(t, err)

	querytest.AssertCount(t, tx, 1, `
			SELECT COUNT(*) FROM ledger.flow
				WHERE id = $1
				AND type = 'alert-redemption'
				AND metadata = '{"bar":"baz","type":"foo"}'::jsonb
				AND twitch_user_id = '4444'
				AND delta_points = -350
				AND created_at = now()
				AND finalized_at IS NULL
				AND accepted = false
		`, flowUuid)
}
