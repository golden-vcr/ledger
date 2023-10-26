package queries_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/golden-vcr/ledger/gen/queries"
	"github.com/golden-vcr/server-common/querytest"
	"github.com/stretchr/testify/assert"
)

func Test_GetBalance(t *testing.T) {
	tx := querytest.PrepareTx(t)
	q := queries.New(tx)

	_, err := q.GetBalance(context.Background(), "31337")
	assert.ErrorIs(t, err, sql.ErrNoRows)

	// 00488e6a-75b5-4b4b-a56e-d3ffca36d186 is a pending inflow of 1200 points
	_, err = tx.Exec(`
		INSERT INTO ledger.flow (
			id,
			type,
			metadata,
			twitch_user_id,
			delta_points
		) VALUES (
			'00488e6a-75b5-4b4b-a56e-d3ffca36d186',
			'manual-credit',
			'{"note":"unit test"}'::jsonb,
			'31337',
			1200
		)
	`)
	assert.NoError(t, err)

	// Pending inflow counts toward total but is not yet available to spend
	balance, err := q.GetBalance(context.Background(), "31337")
	assert.NoError(t, err)
	assert.Equal(t, int32(1200), balance.TotalPoints)
	assert.Equal(t, int32(0), balance.AvailablePoints)

	// Finalize the inflow as accepted to make it available
	_, err = tx.Exec(`
		UPDATE ledger.flow SET finalized_at = now(), accepted = true
			WHERE id = '00488e6a-75b5-4b4b-a56e-d3ffca36d186'
	`)
	assert.NoError(t, err)
	balance, err = q.GetBalance(context.Background(), "31337")
	assert.NoError(t, err)
	assert.Equal(t, int32(1200), balance.TotalPoints)
	assert.Equal(t, int32(1200), balance.AvailablePoints)

	// 10ce4abc-442c-48ca-87ba-d4ce0e26b6cb is a pending outflow of 250 points
	_, err = tx.Exec(`
		INSERT INTO ledger.flow (
			id,
			type,
			metadata,
			twitch_user_id,
			delta_points
		) VALUES (
			'10ce4abc-442c-48ca-87ba-d4ce0e26b6cb',
			'alert-redemption',
			'{"type":"foo"}'::jsonb,
			'31337',
			-250
		)
	`)
	assert.NoError(t, err)

	// Pending outflow reduces available balance but does not yet affect total
	balance, err = q.GetBalance(context.Background(), "31337")
	assert.NoError(t, err)
	assert.Equal(t, int32(1200), balance.TotalPoints)
	assert.Equal(t, int32(950), balance.AvailablePoints)

	// Finalize the outflow as accepted to make it official
	_, err = tx.Exec(`
		UPDATE ledger.flow SET finalized_at = now(), accepted = true
			WHERE id = '10ce4abc-442c-48ca-87ba-d4ce0e26b6cb'
	`)
	assert.NoError(t, err)
	balance, err = q.GetBalance(context.Background(), "31337")
	assert.NoError(t, err)
	assert.Equal(t, int32(950), balance.TotalPoints)
	assert.Equal(t, int32(950), balance.AvailablePoints)

	// Add an inflow and an outflow that are both recorded as rejected: they should not
	// affect balances in any way once rejected
	_, err = tx.Exec(`
		INSERT INTO ledger.flow (
			id,
			type,
			metadata,
			twitch_user_id,
			delta_points,
			created_at,
			finalized_at,
			accepted
		) VALUES (
			gen_random_uuid(),
			'manual-credit',
			'{"note":"unit test"}'::jsonb,
			'31337',
			500,
			now(),
			now(),
			false
		)
	`)
	assert.NoError(t, err)
	_, err = tx.Exec(`
		INSERT INTO ledger.flow (
			id,
			type,
			metadata,
			twitch_user_id,
			delta_points,
			created_at,
			finalized_at,
			accepted
		) VALUES (
			gen_random_uuid(),
			'alert-redemption',
			'{"type":"foo"}'::jsonb,
			'31337',
			-250,
			now(),
			now(),
			false
		)
	`)
	assert.NoError(t, err)
	balance, err = q.GetBalance(context.Background(), "31337")
	assert.NoError(t, err)
	assert.Equal(t, int32(950), balance.TotalPoints)
	assert.Equal(t, int32(950), balance.AvailablePoints)
}
