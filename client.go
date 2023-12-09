package ledger

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

var ErrNotEnoughPoints = errors.New("not enough points")

type TransactionContext interface {
	Accept(ctx context.Context) error
	Finalize(ctx context.Context) error
}

type Client interface {
	RequestCreditFromCheer(ctx context.Context, accessToken string, twitchUserId string, numPointsToCredit int, message string) (uuid.UUID, error)
	RequestAlertRedemption(ctx context.Context, accessToken string, numPointsToDebit int, alertType string, alertMetadata *json.RawMessage) (TransactionContext, error)
}

// NewClient initializes an HTTP client configured to make requests against the
// golden-vcr/ledger server running at the given URL
func NewClient(ledgerUrl string) Client {
	return &client{
		ledgerUrl: ledgerUrl,
	}
}

type client struct {
	http.Client
	ledgerUrl string
}

func (c *client) RequestCreditFromCheer(ctx context.Context, accessToken string, twitchUserId string, numPointsToCredit int, message string) (uuid.UUID, error) {
	// Build a request payload for POST /inflow/cheer
	payload := CheerRequest{
		NumPointsToCredit: numPointsToCredit,
		Message:           message,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return uuid.UUID{}, err
	}

	// Prepare a request to POST /inflow/cheer that creates and finalizes an inflow that
	// credits the requested number of points, authorized with the provided JWT (which
	// also identifies the target user)
	url := c.ledgerUrl + "/inflow/cheer"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payloadBytes))
	if err != nil {
		return uuid.UUID{}, err
	}
	req.Header.Set("authorization", fmt.Sprintf("Bearer %s", accessToken))

	// Initiate the request and make sure it completes successfully
	res, err := c.Do(req)
	if err != nil {
		return uuid.UUID{}, err
	}

	// For any unexpected or non-OK response, propagate an error and halt
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusCreated {
		suffix := ""
		if body, err := io.ReadAll(res.Body); err == nil {
			suffix = fmt.Sprintf(": %s", body)
		}
		return uuid.UUID{}, fmt.Errorf("got response %d from POST %s%s", res.StatusCode, url, suffix)
	}

	// We have an OK response; parse the response body to get our transaction ID
	contentType := res.Header.Get("content/type")
	if contentType != "" && !strings.HasPrefix(contentType, "application/json") {
		return uuid.UUID{}, fmt.Errorf("got unexpected content-type '%s' from POST %s", contentType, url)
	}
	var result TransactionResult
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return uuid.UUID{}, fmt.Errorf("error decoding response body: %w", err)
	}
	return result.FlowId, nil
}

func (c *client) RequestAlertRedemption(ctx context.Context, accessToken string, numPointsToDebit int, alertType string, alertMetadata *json.RawMessage) (TransactionContext, error) {
	// Build a request payload for POST /outflow
	payload := AlertRedemptionRequest{
		Type:             TransactionTypeAlertRedemption,
		NumPointsToDebit: numPointsToDebit,
		AlertType:        alertType,
		AlertMetadata:    alertMetadata,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	// Prepare a request to POST /outflow that create a pending outflow with the
	// requested parameters, then return a flow ID which we can later use to finalize
	// the transaction
	url := c.ledgerUrl + "/outflow"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("authorization", fmt.Sprintf("Bearer %s", accessToken))

	// Initiate the request and make sure it completes successfully
	res, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	// A 409 response indicates that the user identified by the auth token does not have
	// enough points available; propagate that error as ErrNotEnoughPoints
	if res.StatusCode == http.StatusConflict {
		return nil, ErrNotEnoughPoints
	}

	// For any unexpected or non-OK response, propagate an error and halt
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("got response %d from POST %s", res.StatusCode, url)
	}

	// We have an OK response; parse the response body to get our transaction ID
	contentType := res.Header.Get("content/type")
	if contentType != "" && !strings.HasPrefix(contentType, "application/json") {
		return nil, fmt.Errorf("got unexpected content-type '%s' from POST %s", contentType, url)
	}
	var result TransactionResult
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error decoding response body: %w", err)
	}

	return &transactionContext{
		c:           c,
		accessToken: accessToken,
		flowId:      result.FlowId,
	}, nil
}

func (c *client) finalize(ctx context.Context, accessToken string, flowId uuid.UUID, accept bool) error {
	// Prepare a PATCH or DELETE request to accept or reject the transaction
	method := http.MethodDelete
	if accept {
		method = http.MethodPatch
	}
	url := fmt.Sprintf("%s/outflow/%s", c.ledgerUrl, flowId)
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("authorization", fmt.Sprintf("Bearer %s", accessToken))

	// Initiate the request and make sure it completes successfully with a 204 status
	res, err := c.Do(req)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusNoContent {
		return fmt.Errorf("got response %d from %s %s", res.StatusCode, method, url)
	}
	return nil
}

type transactionContext struct {
	c           *client
	accessToken string
	flowId      uuid.UUID
	finalized   bool
}

func (t *transactionContext) Accept(ctx context.Context) error {
	if t.finalized {
		return fmt.Errorf("transaction has already been finalized upon call to Accept")
	}
	if err := t.c.finalize(ctx, t.accessToken, t.flowId, true); err != nil {
		return err
	}
	t.finalized = true
	return nil
}

func (t *transactionContext) Finalize(ctx context.Context) error {
	if !t.finalized {
		if err := t.c.finalize(ctx, t.accessToken, t.flowId, false); err != nil {
			return err
		}
		t.finalized = true
	}
	return nil
}
