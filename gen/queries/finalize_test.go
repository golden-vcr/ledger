package queries_test

import (
	"context"
	"testing"

	"github.com/golden-vcr/ledger/gen/queries"
	"github.com/golden-vcr/server-common/querytest"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func Test_FinalizeFlow(t *testing.T) {
	tx := querytest.PrepareTx(t)
	q := queries.New(tx)

	flows := []struct {
		id       uuid.UUID
		accepted bool
	}{
		{
			uuid.MustParse("14645be0-c59a-4d7e-b6b7-5bc5fcef865e"),
			true,
		},
		{
			uuid.MustParse("55b792d3-5ddf-48b1-9f32-fa0338532264"),
			false,
		},
	}
	for _, flow := range flows {
		_, err := tx.Exec(`
			INSERT INTO ledger.flow (
				id,
				type,
				metadata,
				twitch_user_id,
				delta_points
			) VALUES (
				$1,
				'manual-credit',
				'{"note":"unit test"}'::jsonb,
				'54321',
				100
			)
		`, flow.id)
		assert.NoError(t, err)

		querytest.AssertCount(t, tx, 1, `
			SELECT COUNT(*) FROM ledger.flow
				WHERE id = $1
				AND type = 'manual-credit'
				AND metadata = '{"note":"unit test"}'::jsonb
				AND twitch_user_id = '54321'
				AND delta_points = 100
				AND created_at = now()
				AND finalized_at IS NULL
				AND accepted = false
		`, flow.id)

		res, err := q.FinalizeFlow(context.Background(), queries.FinalizeFlowParams{
			FlowID:   flow.id,
			Accepted: flow.accepted,
		})
		assert.NoError(t, err)
		querytest.AssertNumRowsChanged(t, res, 1)

		querytest.AssertCount(t, tx, 1, `
			SELECT COUNT(*) FROM ledger.flow
				WHERE id = $1
				AND type = 'manual-credit'
				AND metadata = '{"note":"unit test"}'::jsonb
				AND twitch_user_id = '54321'
				AND delta_points = 100
				AND created_at = now()
				AND finalized_at = now()
				AND accepted = $2
		`, flow.id, flow.accepted)

		// Attempting to finalizing an already-finalized transaction has no effect
		res, err = q.FinalizeFlow(context.Background(), queries.FinalizeFlowParams{
			FlowID:   flow.id,
			Accepted: flow.accepted,
		})
		assert.NoError(t, err)
		querytest.AssertNumRowsChanged(t, res, 0)
	}
}
