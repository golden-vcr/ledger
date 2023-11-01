package queries_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/golden-vcr/ledger/gen/queries"
	"github.com/golden-vcr/server-common/querytest"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func Test_GetFlow(t *testing.T) {
	tx := querytest.PrepareTx(t)
	q := queries.New(tx)

	row, err := q.GetFlow(context.Background(), uuid.MustParse("ca7c92e1-cc99-4e2a-b1d2-5109f1a6a9aa"))
	assert.ErrorIs(t, err, sql.ErrNoRows)

	_, err = tx.Exec(`
			INSERT INTO ledger.flow (
				id,
				type,
				metadata,
				twitch_user_id,
				delta_points
			) VALUES (
				'ca7c92e1-cc99-4e2a-b1d2-5109f1a6a9aa',
				'manual-credit',
				'{"note":"unit test"}'::jsonb,
				'54321',
				100
			)
		`)
	assert.NoError(t, err)

	row, err = q.GetFlow(context.Background(), uuid.MustParse("ca7c92e1-cc99-4e2a-b1d2-5109f1a6a9aa"))
	assert.NoError(t, err)
	assert.Equal(t, queries.GetFlowRow{
		TwitchUserID: "54321",
		FinalizedAt:  sql.NullTime{},
		Accepted:     false,
	}, row)

	_, err = tx.Exec("UPDATE ledger.flow SET finalized_at = now(), accepted = true WHERE flow.id = 'ca7c92e1-cc99-4e2a-b1d2-5109f1a6a9aa'")
	assert.NoError(t, err)

	row, err = q.GetFlow(context.Background(), uuid.MustParse("ca7c92e1-cc99-4e2a-b1d2-5109f1a6a9aa"))
	assert.NoError(t, err)
	assert.True(t, row.FinalizedAt.Valid)
	assert.True(t, row.Accepted)
}

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
