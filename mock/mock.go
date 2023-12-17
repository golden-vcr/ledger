package ledgermock

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/golden-vcr/auth"
	"github.com/golden-vcr/ledger"
	"github.com/google/uuid"
)

type Client struct {
	statesByUserAccessToken map[string]*mockUserState
}

type mockUserState struct {
	initialBalance int
	debits         []*mockDebit
}

type mockDebit struct {
	id               uuid.UUID
	numPointsToDebit int
	isFinalized      bool
	isAccepted       bool
}

func NewClient() *Client {
	return &Client{
		statesByUserAccessToken: make(map[string]*mockUserState),
	}
}

func (c *Client) Grant(accessToken string, initialBalance int) *Client {
	c.statesByUserAccessToken[accessToken] = &mockUserState{
		initialBalance: initialBalance,
	}
	return c
}

func (c *Client) RequestCreditFromCheer(ctx context.Context, accessToken string, numPointsToCredit int, message string) (uuid.UUID, error) {
	return uuid.UUID{}, fmt.Errorf("not mocked")
}

func (c *Client) RequestCreditFromSubscription(ctx context.Context, accessToken string, basePointsToCredit int, isInitial bool, isGift bool, message string, creditMultiplier float64) (uuid.UUID, error) {
	return uuid.UUID{}, fmt.Errorf("not mocked")
}

func (c *Client) RequestCreditFromGiftSub(ctx context.Context, accessToken string, basePointsToCredit int, numSubscriptions int, creditMultiplier float64) (uuid.UUID, error) {
	return uuid.UUID{}, fmt.Errorf("not mocked")
}

func (c *Client) RequestAlertRedemption(ctx context.Context, accessToken string, numPointsToDebit int, alertType string, alertMetadata *json.RawMessage) (ledger.TransactionContext, error) {
	balance, err := c.getAvailableBalance(accessToken)
	if err != nil {
		return nil, err
	}
	if balance < numPointsToDebit {
		return nil, ledger.ErrNotEnoughPoints
	}
	debit := &mockDebit{
		numPointsToDebit: numPointsToDebit,
	}
	c.statesByUserAccessToken[accessToken].debits = append(c.statesByUserAccessToken[accessToken].debits, debit)
	return debit, nil
}

func (c *Client) getAvailableBalance(accessToken string) (int, error) {
	state, ok := c.statesByUserAccessToken[accessToken]
	if !ok {
		return -1, auth.ErrUnauthorized
	}
	balance := state.initialBalance
	for _, debit := range state.debits {
		if debit.isFinalized && !debit.isAccepted {
			continue
		}
		balance -= debit.numPointsToDebit
	}
	return balance, nil
}

func (m *mockDebit) Accept(ctx context.Context) error {
	if m.isFinalized {
		return fmt.Errorf("already finalized")
	}
	m.isAccepted = true
	m.isFinalized = true
	return nil
}

func (m *mockDebit) Finalize(ctx context.Context) error {
	if !m.isFinalized {
		m.isFinalized = true
	}
	return nil
}

var _ ledger.Client = (*Client)(nil)
