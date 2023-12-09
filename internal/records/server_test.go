package records

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golden-vcr/auth"
	authmock "github.com/golden-vcr/auth/mock"
	"github.com/golden-vcr/ledger/gen/queries"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func Test_Server_handleGetBalance(t *testing.T) {
	tests := []struct {
		name          string
		q             *mockQueries
		authorization string
		wantStatus    int
		wantBody      string
	}{
		{
			"normal usage",
			&mockQueries{
				userId: "1001",
				balance: queries.GetBalanceRow{
					TotalPoints:     2500,
					AvailablePoints: 2300,
				},
			},
			"mock-token",
			http.StatusOK,
			`{"totalPoints":2500,"availablePoints":2300}`,
		},
		{
			"zero values are returned if no balance record exists for auth'd user",
			&mockQueries{},
			"mock-token",
			http.StatusOK,
			`{"totalPoints":0,"availablePoints":0}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authClient := authmock.NewClient().AllowTwitchUserAccessToken("mock-token", auth.RoleViewer, auth.UserDetails{
				Id:          "1001",
				Login:       "testuser",
				DisplayName: "TestUser",
			})
			s := &Server{
				q: tt.q,
			}
			f := http.HandlerFunc(s.handleGetBalance)
			handler := auth.RequireAccess(authClient, auth.RoleViewer, f)

			req := httptest.NewRequest(http.MethodGet, "/balance", nil)
			req.Header.Set("authorization", tt.authorization)
			res := httptest.NewRecorder()
			handler.ServeHTTP(res, req)

			b, err := io.ReadAll(res.Body)
			assert.NoError(t, err)
			body := strings.TrimSuffix(string(b), "\n")
			assert.Equal(t, tt.wantStatus, res.Code)
			assert.Equal(t, tt.wantBody, body)
		})
	}
}

func Test_Server_handleGetHistory(t *testing.T) {
	tests := []struct {
		name          string
		q             *mockQueries
		authorization string
		maxItems      int
		fromCursor    string
		wantStatus    int
		wantBody      string
	}{
		{
			"normal usage",
			&mockQueries{
				userId: "1001",
				historyRows: []queries.GetTransactionHistoryRow{
					{
						ID:          uuid.MustParse("6582a6f6-43e4-4d3d-9d34-0f2e58b41e5f"),
						Type:        "alert-redemption",
						Metadata:    []byte(`{"type":"whatever"}`),
						DeltaPoints: -200,
						CreatedAt:   time.Date(1997, 9, 1, 13, 0, 0, 0, time.UTC),
					},
					{
						ID:          uuid.MustParse("18d3d13c-625e-46df-bd34-e2cc2b7be15e"),
						Type:        "manual-credit",
						Metadata:    []byte(`{"note":"will be rejected"}`),
						DeltaPoints: 5000,
						CreatedAt:   time.Date(1997, 9, 1, 12, 30, 0, 0, time.UTC),
						FinalizedAt: sql.NullTime{Valid: true, Time: time.Date(1997, 9, 1, 12, 30, 0, 0, time.UTC)},
						Accepted:    false,
					},
					{
						ID:          uuid.MustParse("0db47d1c-41f9-4808-bc8d-bf097eeb6319"),
						Type:        "manual-credit",
						Metadata:    []byte(`{"note":"foo"}`),
						DeltaPoints: 2500,
						CreatedAt:   time.Date(1997, 9, 1, 12, 0, 0, 0, time.UTC),
						FinalizedAt: sql.NullTime{Valid: true, Time: time.Date(1997, 9, 1, 12, 1, 0, 0, time.UTC)},
						Accepted:    true,
					},
				},
			},
			"mock-token",
			-1,
			"",
			http.StatusOK,
			`{"items":[{"id":"6582a6f6-43e4-4d3d-9d34-0f2e58b41e5f","timestamp":"1997-09-01T13:00:00Z","type":"alert-redemption","state":"pending","deltaPoints":-200,"description":"Redeemed alert of type 'whatever'"},{"id":"18d3d13c-625e-46df-bd34-e2cc2b7be15e","timestamp":"1997-09-01T12:30:00Z","type":"manual-credit","state":"rejected","deltaPoints":5000,"description":"Manual credit: will be rejected"},{"id":"0db47d1c-41f9-4808-bc8d-bf097eeb6319","timestamp":"1997-09-01T12:01:00Z","type":"manual-credit","state":"accepted","deltaPoints":2500,"description":"Manual credit: foo"}]}`,
		},
		{
			"paginated: first page",
			&mockQueries{
				userId: "1001",
				historyRows: []queries.GetTransactionHistoryRow{
					{
						ID:          uuid.MustParse("6582a6f6-43e4-4d3d-9d34-0f2e58b41e5f"),
						Type:        "alert-redemption",
						Metadata:    []byte(`{"type":"whatever"}`),
						DeltaPoints: -200,
						CreatedAt:   time.Date(1997, 9, 1, 13, 0, 0, 0, time.UTC),
					},
					{
						ID:          uuid.MustParse("18d3d13c-625e-46df-bd34-e2cc2b7be15e"),
						Type:        "manual-credit",
						Metadata:    []byte(`{"note":"will be rejected"}`),
						DeltaPoints: 5000,
						CreatedAt:   time.Date(1997, 9, 1, 12, 30, 0, 0, time.UTC),
						FinalizedAt: sql.NullTime{Valid: true, Time: time.Date(1997, 9, 1, 12, 30, 0, 0, time.UTC)},
						Accepted:    false,
					},
					{
						ID:          uuid.MustParse("0db47d1c-41f9-4808-bc8d-bf097eeb6319"),
						Type:        "manual-credit",
						Metadata:    []byte(`{"note":"foo"}`),
						DeltaPoints: 2500,
						CreatedAt:   time.Date(1997, 9, 1, 12, 0, 0, 0, time.UTC),
						FinalizedAt: sql.NullTime{Valid: true, Time: time.Date(1997, 9, 1, 12, 1, 0, 0, time.UTC)},
						Accepted:    true,
					},
				},
			},
			"mock-token",
			2,
			"",
			http.StatusOK,
			`{"items":[{"id":"6582a6f6-43e4-4d3d-9d34-0f2e58b41e5f","timestamp":"1997-09-01T13:00:00Z","type":"alert-redemption","state":"pending","deltaPoints":-200,"description":"Redeemed alert of type 'whatever'"},{"id":"18d3d13c-625e-46df-bd34-e2cc2b7be15e","timestamp":"1997-09-01T12:30:00Z","type":"manual-credit","state":"rejected","deltaPoints":5000,"description":"Manual credit: will be rejected"}],"nextCursor":"0db47d1c-41f9-4808-bc8d-bf097eeb6319"}`,
		},
		{
			"paginated: second page",
			&mockQueries{
				userId: "1001",
				historyRows: []queries.GetTransactionHistoryRow{
					{
						ID:          uuid.MustParse("6582a6f6-43e4-4d3d-9d34-0f2e58b41e5f"),
						Type:        "alert-redemption",
						Metadata:    []byte(`{"type":"whatever"}`),
						DeltaPoints: -200,
						CreatedAt:   time.Date(1997, 9, 1, 13, 0, 0, 0, time.UTC),
					},
					{
						ID:          uuid.MustParse("18d3d13c-625e-46df-bd34-e2cc2b7be15e"),
						Type:        "manual-credit",
						Metadata:    []byte(`{"note":"will be rejected"}`),
						DeltaPoints: 5000,
						CreatedAt:   time.Date(1997, 9, 1, 12, 30, 0, 0, time.UTC),
						FinalizedAt: sql.NullTime{Valid: true, Time: time.Date(1997, 9, 1, 12, 30, 0, 0, time.UTC)},
						Accepted:    false,
					},
					{
						ID:          uuid.MustParse("0db47d1c-41f9-4808-bc8d-bf097eeb6319"),
						Type:        "manual-credit",
						Metadata:    []byte(`{"note":"foo"}`),
						DeltaPoints: 2500,
						CreatedAt:   time.Date(1997, 9, 1, 12, 0, 0, 0, time.UTC),
						FinalizedAt: sql.NullTime{Valid: true, Time: time.Date(1997, 9, 1, 12, 1, 0, 0, time.UTC)},
						Accepted:    true,
					},
				},
			},
			"mock-token",
			2,
			"0db47d1c-41f9-4808-bc8d-bf097eeb6319",
			http.StatusOK,
			`{"items":[{"id":"0db47d1c-41f9-4808-bc8d-bf097eeb6319","timestamp":"1997-09-01T12:01:00Z","type":"manual-credit","state":"accepted","deltaPoints":2500,"description":"Manual credit: foo"}]}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authClient := authmock.NewClient().AllowTwitchUserAccessToken("mock-token", auth.RoleViewer, auth.UserDetails{
				Id:          "1001",
				Login:       "testuser",
				DisplayName: "TestUser",
			})
			s := &Server{
				q: tt.q,
			}
			f := http.HandlerFunc(s.handleGetHistory)
			handler := auth.RequireAccess(authClient, auth.RoleViewer, f)

			req := httptest.NewRequest(http.MethodGet, "/history", nil)
			req.Header.Set("authorization", tt.authorization)
			q := req.URL.Query()
			if tt.maxItems >= 0 {
				q.Add("max", fmt.Sprintf("%d", tt.maxItems))
			}
			if tt.fromCursor != "" {
				q.Add("from", tt.fromCursor)
			}
			req.URL.RawQuery = q.Encode()
			res := httptest.NewRecorder()
			handler.ServeHTTP(res, req)

			b, err := io.ReadAll(res.Body)
			assert.NoError(t, err)
			body := strings.TrimSuffix(string(b), "\n")
			assert.Equal(t, tt.wantStatus, res.Code)
			assert.Equal(t, tt.wantBody, body)
		})
	}
}

type mockQueries struct {
	userId      string
	balance     queries.GetBalanceRow
	historyRows []queries.GetTransactionHistoryRow
}

func (m *mockQueries) GetBalance(ctx context.Context, twitchUserID string) (queries.GetBalanceRow, error) {
	if twitchUserID != m.userId {
		return queries.GetBalanceRow{}, sql.ErrNoRows
	}
	return m.balance, nil
}

func (m *mockQueries) GetTransactionHistory(ctx context.Context, arg queries.GetTransactionHistoryParams) ([]queries.GetTransactionHistoryRow, error) {
	startIndex := 0
	if arg.StartID.Valid {
		for m.historyRows[startIndex].ID != arg.StartID.UUID {
			startIndex++
		}
	}
	rows := make([]queries.GetTransactionHistoryRow, 0, arg.NumRecords)
	for i := startIndex; i < len(m.historyRows); i++ {
		rows = append(rows, m.historyRows[i])
		if len(rows) == int(arg.NumRecords) {
			break
		}
	}
	return rows, nil
}
