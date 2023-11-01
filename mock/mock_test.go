package ledgermock

import (
	"context"
	"testing"

	"github.com/golden-vcr/auth"
	"github.com/golden-vcr/ledger"
	"github.com/stretchr/testify/assert"
)

func Test_Client(t *testing.T) {
	c := NewClient().Grant("token-a", 1000).Grant("token-b", 200)

	assertCurrentBalance(t, c, "token-a", 1000)

	transaction, err := c.RequestAlertRedemption(context.Background(), "token-b", 300, "foo", nil)
	assert.ErrorIs(t, err, ledger.ErrNotEnoughPoints)
	assert.Nil(t, transaction)

	transaction, err = c.RequestAlertRedemption(context.Background(), "bad-token", 300, "foo", nil)
	assert.ErrorIs(t, err, auth.ErrUnauthorized)
	assert.Nil(t, transaction)

	transaction, err = c.RequestAlertRedemption(context.Background(), "token-a", 300, "foo", nil)
	assert.NoError(t, err)
	assert.NotNil(t, transaction)
	err = transaction.Accept(context.Background())
	assert.NoError(t, err)
	assertCurrentBalance(t, c, "token-a", 700)

	transaction, err = c.RequestAlertRedemption(context.Background(), "token-a", 300, "foo", nil)
	assert.NoError(t, err)
	assert.NotNil(t, transaction)
	assertCurrentBalance(t, c, "token-a", 400)
	err = transaction.Finalize(context.Background())
	assert.NoError(t, err)
	assertCurrentBalance(t, c, "token-a", 700)
}

func assertCurrentBalance(t *testing.T, c *Client, token string, want int) {
	got, err := c.getAvailableBalance(token)
	assert.NoError(t, err)
	assert.Equal(t, want, got)
}
