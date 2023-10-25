package queries_test

import (
	"context"
	"testing"

	"github.com/golden-vcr/ledger/gen/queries"
	"github.com/golden-vcr/server-common/querytest"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func Test_GetTransactionHistory(t *testing.T) {
	tx := querytest.PrepareTx(t)
	q := queries.New(tx)

	_, err := tx.Exec(`
		INSERT INTO ledger.flow (id, type, metadata, twitch_user_id, delta_points, created_at, finalized_at, accepted) VALUES
			('03270514-a9e8-4c6c-97e2-78fa9d72ab8c', 'manual-credit', '{"note":"test1"}'::jsonb, '12345', 111, now() - '1h'::interval, now() - '1h'::interval, true),
			('f0c6f086-6885-4ed2-a693-25b5a4205e91', 'manual-credit', '{"note":"test2"}'::jsonb, '67890', 1000, now() - '2h'::interval, now() - '2h'::interval, true),
			('acbcb23e-b037-428a-9df4-b04e7f8da6a3', 'manual-credit', '{"note":"test3"}'::jsonb, '12345', 222, now() - '3h'::interval, now() - '3h'::interval, false),
			('398893a4-1d18-42c4-80fd-d2cd48e27a83', 'manual-credit', '{"note":"test4"}'::jsonb, '12345', 333, now() - '4h'::interval, now() - '4h'::interval, true),
			('c03e6b2b-0e18-4ee1-ab74-04b9616ce8a4', 'manual-credit', '{"note":"test5"}'::jsonb, '12345', 444, now() - '5h'::interval, NULL, false),
			('dd9348ef-7277-47aa-9d40-bd67a5909a07', 'alert-redemption', '{"type":"test"}'::jsonb, '12345', -50, now() - '6h'::interval, now() - '6h'::interval, true),
			('f1eb1e52-592f-4dd1-85a3-da7fbc3889e1', 'alert-redemption', '{"type":"test"}'::jsonb, '12345', -25, now() - '7h'::interval, NULL, false);
	`)
	assert.NoError(t, err)

	rows, err := q.GetTransactionHistory(context.Background(), queries.GetTransactionHistoryParams{
		TwitchUserID: "12345",
		NumRecords:   5,
	})
	assert.NoError(t, err)

	ids := make([]uuid.UUID, 0)
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	assert.Equal(t, []uuid.UUID{
		uuid.MustParse("03270514-a9e8-4c6c-97e2-78fa9d72ab8c"),
		uuid.MustParse("acbcb23e-b037-428a-9df4-b04e7f8da6a3"),
		uuid.MustParse("398893a4-1d18-42c4-80fd-d2cd48e27a83"),
		uuid.MustParse("c03e6b2b-0e18-4ee1-ab74-04b9616ce8a4"),
		uuid.MustParse("dd9348ef-7277-47aa-9d40-bd67a5909a07"),
	}, ids)

	first := rows[0]
	assert.Equal(t, "manual-credit", first.Type)
	assert.Equal(t, `{"note": "test1"}`, string(first.Metadata))
	assert.Equal(t, int32(111), first.DeltaPoints)
	assert.True(t, first.FinalizedAt.Valid)
	assert.True(t, first.Accepted)

	rows, err = q.GetTransactionHistory(context.Background(), queries.GetTransactionHistoryParams{
		TwitchUserID: "12345",
		StartID:      uuid.NullUUID{Valid: true, UUID: ids[len(ids)-1]},
		NumRecords:   5,
	})
	assert.NoError(t, err)
	ids = make([]uuid.UUID, 0)
	for _, row := range rows {
		ids = append(ids, row.ID)
	}

	assert.Equal(t, []uuid.UUID{
		uuid.MustParse("dd9348ef-7277-47aa-9d40-bd67a5909a07"),
		uuid.MustParse("f1eb1e52-592f-4dd1-85a3-da7fbc3889e1"),
	}, ids)

	last := rows[len(rows)-1]
	assert.Equal(t, "alert-redemption", last.Type)
	assert.Equal(t, `{"type": "test"}`, string(last.Metadata))
	assert.Equal(t, int32(-25), last.DeltaPoints)
	assert.False(t, last.FinalizedAt.Valid)
	assert.False(t, last.Accepted)
}
