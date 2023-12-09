package cheer

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golden-vcr/auth"
	authmock "github.com/golden-vcr/auth/mock"
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
			"internal-jwt",
			`{"numPointsToCredit":400,"message":"hello"}`,
			http.StatusOK,
			`{"flowId":"0dc95aba-6f8f-4e13-9081-ba1b2ced8f39"}`,
		},
		{
			"message is optional",
			&mockQueries{},
			"internal-jwt",
			`{"numPointsToCredit":400}`,
			http.StatusOK,
			`{"flowId":"0dc95aba-6f8f-4e13-9081-ba1b2ced8f39"}`,
		},
		{
			"points to credit must be positive",
			&mockQueries{},
			"internal-jwt",
			`{"numPointsToCredit":0}`,
			http.StatusBadRequest,
			"invalid request payload: 'numPointsToCredit' must be set to a positive integer",
		},
		{
			"malformed JSON payload is a 400 error",
			&mockQueries{},
			"internal-jwt",
			`{""}`,
			http.StatusBadRequest,
			"invalid request payload: invalid character '}' after object key",
		},
		{
			"invalid JWT is a 401 error",
			&mockQueries{},
			"twitch-user-access-token",
			`{"numPointsToCredit":400,"message":"hello"}`,
			http.StatusUnauthorized,
			"access denied",
		},
		{
			"missing JWT is a 401 error",
			&mockQueries{},
			"",
			`{"numPointsToCredit":400,"message":"hello"}`,
			http.StatusBadRequest,
			"Internal JWT must be supplied in Authorization header",
		},
		{
			"failure to update database is a 500 error",
			&mockQueries{
				err: fmt.Errorf("mock error"),
			},
			"internal-jwt",
			`{"numPointsToCredit":400,"message":"hello"}`,
			http.StatusInternalServerError,
			"mock error",
		},
	}
	for _, tt := range tests {
		c := authmock.NewClient().AllowAuthoritativeJWT("internal-jwt", auth.UserDetails{
			Id:          "1337",
			Login:       "leetman",
			DisplayName: "LEETman",
		}).AllowTwitchUserAccessToken("twitch-user-access-token", auth.RoleViewer, auth.UserDetails{
			Id:          "100",
			Login:       "badman",
			DisplayName: "Badman",
		})
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{
				q: tt.q,
			}
			handler := auth.RequireAuthority(c, http.HandlerFunc(s.handlePostCheer))
			req := httptest.NewRequest(http.MethodPost, "/inflow/cheer", strings.NewReader(tt.body))
			if tt.authorization != "" {
				req.Header.Add("authorization", fmt.Sprintf("Bearer %s", tt.authorization))
			}
			res := httptest.NewRecorder()
			handler.ServeHTTP(res, req)

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
