package cheer

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golden-vcr/ledger/gen/queries"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func Test_Server_handlePostCheer(t *testing.T) {
	tests := []struct {
		name          string
		q             *mockQueries
		authorization string
		body          string
		wantStatus    int
		wantBody      string
	}{
		{
			"normal usage",
			&mockQueries{},
			"secret",
			`{"twitchUserId":"1337","numPointsToCredit":400,"message":"hello"}`,
			http.StatusOK,
			`{"flowId":"0dc95aba-6f8f-4e13-9081-ba1b2ced8f39"}`,
		},
		{
			"message is optional",
			&mockQueries{},
			"secret",
			`{"twitchUserId":"1337","numPointsToCredit":400}`,
			http.StatusOK,
			`{"flowId":"0dc95aba-6f8f-4e13-9081-ba1b2ced8f39"}`,
		},
		{
			"user ID must be specified",
			&mockQueries{},
			"secret",
			`{"twitchUserId":"","numPointsToCredit":400,"message":"hello"}`,
			http.StatusBadRequest,
			"invalid request payload: 'twitchUserId' is required",
		},
		{
			"points to credit must be positive",
			&mockQueries{},
			"secret",
			`{"twitchUserId":"1337","numPointsToCredit":0}`,
			http.StatusBadRequest,
			"invalid request payload: 'numPointsToCredit' must be set to a positive integer",
		},
		{
			"malformed JSON payload is a 400 error",
			&mockQueries{},
			"secret",
			`{""}`,
			http.StatusBadRequest,
			"invalid request payload: invalid character '}' after object key",
		},
		{
			"invalid secret key is a 401 error",
			&mockQueries{},
			"bad-secret",
			`{"twitchUserId":"1337","numPointsToCredit":400,"message":"hello"}`,
			http.StatusUnauthorized,
			"access denied",
		},
		{
			"missing secret key is a 401 error",
			&mockQueries{},
			"",
			`{"twitchUserId":"1337","numPointsToCredit":400,"message":"hello"}`,
			http.StatusUnauthorized,
			"access denied",
		},
		{
			"failure to update database is a 500 error",
			&mockQueries{
				err: fmt.Errorf("mock error"),
			},
			"secret",
			`{"twitchUserId":"1337","numPointsToCredit":400,"message":"hello"}`,
			http.StatusInternalServerError,
			"mock error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{
				q:                    tt.q,
				authorizeCheerInflow: mockAuthorizeCheerInflow,
			}
			req := httptest.NewRequest(http.MethodPost, "/inflow/cheer", strings.NewReader(tt.body))
			if tt.authorization != "" {
				req.Header.Add("authorization", tt.authorization)
			}
			res := httptest.NewRecorder()
			s.handlePostCheer(res, req)

			b, err := io.ReadAll(res.Body)
			assert.NoError(t, err)
			body := strings.TrimSuffix(string(b), "\n")
			assert.Equal(t, tt.wantStatus, res.Code)
			assert.Equal(t, tt.wantBody, body)

			if tt.wantStatus == http.StatusOK || tt.wantStatus == http.StatusCreated {
				assert.Len(t, tt.q.calls, 1)
			} else {
				assert.Len(t, tt.q.calls, 0)
			}
		})
	}
}

func mockAuthorizeCheerInflow(s string) bool {
	return s == "secret"
}

type mockQueries struct {
	err   error
	calls []queries.RecordCheerInflowParams
}

func (m *mockQueries) RecordCheerInflow(ctx context.Context, arg queries.RecordCheerInflowParams) (uuid.UUID, error) {
	if m.err != nil {
		return uuid.UUID{}, m.err
	}
	m.calls = append(m.calls, arg)
	return uuid.MustParse("0dc95aba-6f8f-4e13-9081-ba1b2ced8f39"), nil
}
