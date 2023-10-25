package admin

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

func Test_Server_handlePostManualCredit(t *testing.T) {
	tests := []struct {
		name       string
		q          *mockQueries
		body       string
		wantStatus int
		wantBody   string
	}{
		{
			"points can be awarded via user ID",
			&mockQueries{},
			`{"twitchUserId":"1337","numPointsToCredit":400,"note":"test"}`,
			http.StatusOK,
			`{"flowId":"59c7fe68-b49e-42cc-a2c7-dbc4ddc6f9c8"}`,
		},
		{
			"points can be awarded via username",
			&mockQueries{},
			`{"twitchDisplayName":"somebody","numPointsToCredit":400,"note":"test"}`,
			http.StatusOK,
			`{"flowId":"59c7fe68-b49e-42cc-a2c7-dbc4ddc6f9c8"}`,
		},
		{
			"failure to resolve user ID from twitch username is a 500 error",
			&mockQueries{},
			`{"twitchDisplayName":"nobody","numPointsToCredit":400,"note":"test"}`,
			http.StatusInternalServerError,
			"failed to resolve twitch user ID from username: no such user",
		},
		{
			"supplying both display name and username is an error",
			&mockQueries{},
			`{"twitchUserId":"1337","twitchDisplayName":"somebody","numPointsToCredit":400,"note":"test"}`,
			http.StatusBadRequest,
			"invalid request payload: exactly one of 'twitchDisplayName' and 'twitchUserId' is required",
		},
		{
			"supplying neither display name nor username is an error",
			&mockQueries{},
			`{"numPointsToCredit":400,"note":"test"}`,
			http.StatusBadRequest,
			"invalid request payload: exactly one of 'twitchDisplayName' and 'twitchUserId' is required",
		},
		{
			"failing to supply a non-empty note is an error",
			&mockQueries{},
			`{"twitchUserId":"1337","numPointsToCredit":400,"note":""}`,
			http.StatusBadRequest,
			"invalid request payload: 'note' must be set to a non-empty string",
		},
		{
			"failure to update database is a 500 error",
			&mockQueries{
				err: fmt.Errorf("mock error"),
			},
			`{"twitchUserId":"1337","numPointsToCredit":400,"note":"test"}`,
			http.StatusInternalServerError,
			"mock error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{
				q:                   tt.q,
				resolveTwitchUserId: mockResolveTwitchUserId,
			}
			req := httptest.NewRequest(http.MethodPost, "/inflow/manual-credit", strings.NewReader(tt.body))
			res := httptest.NewRecorder()
			s.handlePostManualCredit(res, req)

			b, err := io.ReadAll(res.Body)
			assert.NoError(t, err)
			body := strings.TrimSuffix(string(b), "\n")
			assert.Equal(t, tt.wantStatus, res.Code)
			assert.Equal(t, tt.wantBody, body)
		})
	}
}

func mockResolveTwitchUserId(ctx context.Context, username string) (string, error) {
	if strings.ToLower(username) == "somebody" {
		return "1337", nil
	}
	return "", fmt.Errorf("no such user")
}

type mockQueries struct {
	err   error
	calls []queries.RecordManualCreditInflowParams
}

func (m *mockQueries) RecordManualCreditInflow(ctx context.Context, arg queries.RecordManualCreditInflowParams) (uuid.UUID, error) {
	if m.err != nil {
		return uuid.UUID{}, m.err
	}
	m.calls = append(m.calls, arg)
	return uuid.MustParse("59c7fe68-b49e-42cc-a2c7-dbc4ddc6f9c8"), nil
}
