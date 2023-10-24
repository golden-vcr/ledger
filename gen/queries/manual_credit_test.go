package queries_test

import (
	"context"
	"testing"

	"github.com/golden-vcr/ledger/gen/queries"
	"github.com/golden-vcr/server-common/querytest"
	"github.com/stretchr/testify/assert"
)

func Test_RecordManualCreditInflow(t *testing.T) {
	tx := querytest.PrepareTx(t)
	q := queries.New(tx)

	querytest.AssertCount(t, tx, 0, "SELECT COUNT(*) FROM ledger.flow")

	flowUuid, err := q.RecordManualCreditInflow(context.Background(), queries.RecordManualCreditInflowParams{
		TwitchUserID:      "4444",
		Note:              "Test credit",
		NumPointsToCredit: 1000,
	})
	assert.NoError(t, err)

	querytest.AssertCount(t, tx, 1, `
			SELECT COUNT(*) FROM ledger.flow
				WHERE id = $1
				AND type = 'manual-credit'
				AND metadata = '{"note":"Test credit"}'::jsonb
				AND twitch_user_id = '4444'
				AND delta_points = 1000
				AND created_at = now()
				AND finalized_at = now()
				AND accepted = true
		`, flowUuid)
}
