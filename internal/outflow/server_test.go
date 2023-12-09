package outflow

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
	"github.com/gorilla/mux"
	"github.com/sqlc-dev/pqtype"
	"github.com/stretchr/testify/assert"
)

func Test_Server_handleCreateOutflow(t *testing.T) {
	tests := []struct {
		name                 string
		q                    *mockQueries
		authorization        string
		requestBody          string
		wantStatus           int
		wantBody             string
		wantAlertRedemptions []mockAlertRedemptionOutflow
	}{
		{
			"unrecognized outflow type is error",
			&mockQueries{},
			"mock-token",
			`{"type":"bad-type","numPointsToDebit":250,"alertType":"foo","alertMetadata":{"x":42}}`,
			http.StatusBadRequest,
			"unsupported transaction type",
			nil,
		},
		{
			"normal alert redemption",
			&mockQueries{
				idSequence: []uuid.UUID{
					uuid.MustParse("7784d456-c499-4d50-80ed-7feaa2757409"),
				},
				balancesByUserId: map[string]queries.GetBalanceRow{
					"1001": {
						AvailablePoints: 1000,
						TotalPoints:     1000,
					},
				},
			},
			"mock-token",
			`{"type":"alert-redemption","numPointsToDebit":250,"alertType":"foo","alertMetadata":{"x":42}}`,
			http.StatusOK,
			`{"flowId":"7784d456-c499-4d50-80ed-7feaa2757409"}`,
			[]mockAlertRedemptionOutflow{
				{
					id:               uuid.MustParse("7784d456-c499-4d50-80ed-7feaa2757409"),
					userId:           "1001",
					numPointsToDebit: 250,
					alertType:        "foo",
					alertMetadata:    pqtype.NullRawMessage{Valid: true, RawMessage: []byte(`{"x":42}`)},
					finalized:        false,
					accepted:         false,
				},
			},
		},
		{
			"insufficient point balance results in a 409 error",
			&mockQueries{
				idSequence: []uuid.UUID{
					uuid.MustParse("7784d456-c499-4d50-80ed-7feaa2757409"),
				},
				balancesByUserId: map[string]queries.GetBalanceRow{
					"1001": {
						AvailablePoints: 200,
						TotalPoints:     1000,
					},
				},
			},
			"mock-token",
			`{"type":"alert-redemption","numPointsToDebit":250,"alertType":"foo","alertMetadata":{"x":42}}`,
			http.StatusConflict,
			"not enough points",
			nil,
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
			f := http.HandlerFunc(s.handleCreateOutflow)
			handler := auth.RequireAccess(authClient, auth.RoleViewer, f)

			req := httptest.NewRequest(http.MethodPost, "/outflow", strings.NewReader(tt.requestBody))
			req.Header.Set("authorization", tt.authorization)
			res := httptest.NewRecorder()
			handler.ServeHTTP(res, req)

			b, err := io.ReadAll(res.Body)
			assert.NoError(t, err)
			body := strings.TrimSuffix(string(b), "\n")
			assert.Equal(t, tt.wantStatus, res.Code)
			assert.Equal(t, tt.wantBody, body)

			assert.Equal(t, tt.wantAlertRedemptions, tt.q.alertRedemptions)
		})
	}
}

func Test_Server_handleFinalizeOutflow(t *testing.T) {
	tests := []struct {
		name                 string
		q                    *mockQueries
		method               string
		flowId               string
		authorization        string
		wantStatus           int
		wantBody             string
		wantAlertRedemptions []mockAlertRedemptionOutflow
	}{
		{
			"pending outflow can be finalized as accepted via PATCH",
			&mockQueries{
				alertRedemptions: []mockAlertRedemptionOutflow{
					{
						id:               uuid.MustParse("7784d456-c499-4d50-80ed-7feaa2757409"),
						userId:           "1001",
						numPointsToDebit: 250,
						alertType:        "foo",
						alertMetadata:    pqtype.NullRawMessage{Valid: true, RawMessage: []byte(`{"x":42}`)},
						finalized:        false,
						accepted:         false,
					},
				},
			},
			http.MethodPatch,
			"7784d456-c499-4d50-80ed-7feaa2757409",
			"mock-token",
			http.StatusNoContent,
			"",
			[]mockAlertRedemptionOutflow{
				{
					id:               uuid.MustParse("7784d456-c499-4d50-80ed-7feaa2757409"),
					userId:           "1001",
					numPointsToDebit: 250,
					alertType:        "foo",
					alertMetadata:    pqtype.NullRawMessage{Valid: true, RawMessage: []byte(`{"x":42}`)},
					finalized:        true,
					accepted:         true,
				},
			},
		},
		{
			"pending outflow can be finalized as rejected via DELETE",
			&mockQueries{
				alertRedemptions: []mockAlertRedemptionOutflow{
					{
						id:               uuid.MustParse("7784d456-c499-4d50-80ed-7feaa2757409"),
						userId:           "1001",
						numPointsToDebit: 250,
						alertType:        "foo",
						alertMetadata:    pqtype.NullRawMessage{Valid: true, RawMessage: []byte(`{"x":42}`)},
						finalized:        false,
						accepted:         false,
					},
				},
			},
			http.MethodDelete,
			"7784d456-c499-4d50-80ed-7feaa2757409",
			"mock-token",
			http.StatusNoContent,
			"",
			[]mockAlertRedemptionOutflow{
				{
					id:               uuid.MustParse("7784d456-c499-4d50-80ed-7feaa2757409"),
					userId:           "1001",
					numPointsToDebit: 250,
					alertType:        "foo",
					alertMetadata:    pqtype.NullRawMessage{Valid: true, RawMessage: []byte(`{"x":42}`)},
					finalized:        true,
					accepted:         false,
				},
			},
		},
		{
			"attempting to finalize already-finalized outflow results in 409",
			&mockQueries{
				alertRedemptions: []mockAlertRedemptionOutflow{
					{
						id:               uuid.MustParse("7784d456-c499-4d50-80ed-7feaa2757409"),
						userId:           "1001",
						numPointsToDebit: 250,
						alertType:        "foo",
						alertMetadata:    pqtype.NullRawMessage{Valid: true, RawMessage: []byte(`{"x":42}`)},
						finalized:        true,
						accepted:         true,
					},
				},
			},
			http.MethodPatch,
			"7784d456-c499-4d50-80ed-7feaa2757409",
			"mock-token",
			http.StatusConflict,
			"transaction is not pending",
			[]mockAlertRedemptionOutflow{
				{
					id:               uuid.MustParse("7784d456-c499-4d50-80ed-7feaa2757409"),
					userId:           "1001",
					numPointsToDebit: 250,
					alertType:        "foo",
					alertMetadata:    pqtype.NullRawMessage{Valid: true, RawMessage: []byte(`{"x":42}`)},
					finalized:        true,
					accepted:         true,
				},
			},
		},
		{
			"attempting to finalize nonexistent outflow results in 404",
			&mockQueries{},
			http.MethodPatch,
			"7784d456-c499-4d50-80ed-7feaa2757409",
			"mock-token",
			http.StatusNotFound,
			"no such transaction",
			nil,
		},
		{
			"attempting to finalize another user's outflow results in 409",
			&mockQueries{
				alertRedemptions: []mockAlertRedemptionOutflow{
					{
						id:               uuid.MustParse("7784d456-c499-4d50-80ed-7feaa2757409"),
						userId:           "2002",
						numPointsToDebit: 250,
						alertType:        "foo",
						alertMetadata:    pqtype.NullRawMessage{Valid: true, RawMessage: []byte(`{"x":42}`)},
						finalized:        false,
						accepted:         false,
					},
				},
			},
			http.MethodPatch,
			"7784d456-c499-4d50-80ed-7feaa2757409",
			"mock-token",
			http.StatusNotFound,
			"no such transaction",
			[]mockAlertRedemptionOutflow{
				{
					id:               uuid.MustParse("7784d456-c499-4d50-80ed-7feaa2757409"),
					userId:           "2002",
					numPointsToDebit: 250,
					alertType:        "foo",
					alertMetadata:    pqtype.NullRawMessage{Valid: true, RawMessage: []byte(`{"x":42}`)},
					finalized:        false,
					accepted:         false,
				},
			},
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
			f := http.HandlerFunc(s.handleFinalizeOutflow)
			handler := auth.RequireAccess(authClient, auth.RoleViewer, f)

			req := httptest.NewRequest(tt.method, fmt.Sprintf("/outflow/%s", tt.flowId), nil)
			req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%s", tt.flowId)})
			req.Header.Set("authorization", tt.authorization)
			res := httptest.NewRecorder()
			handler.ServeHTTP(res, req)

			b, err := io.ReadAll(res.Body)
			assert.NoError(t, err)
			body := strings.TrimSuffix(string(b), "\n")
			assert.Equal(t, tt.wantStatus, res.Code)
			assert.Equal(t, tt.wantBody, body)

			assert.Equal(t, tt.wantAlertRedemptions, tt.q.alertRedemptions)
		})
	}
}

type mockQueries struct {
	idSequence       []uuid.UUID
	nextIdIndex      int
	balancesByUserId map[string]queries.GetBalanceRow
	alertRedemptions []mockAlertRedemptionOutflow
}

type mockAlertRedemptionOutflow struct {
	id               uuid.UUID
	userId           string
	numPointsToDebit int32
	alertType        string
	alertMetadata    pqtype.NullRawMessage
	finalized        bool
	accepted         bool
}

func (m *mockQueries) GetBalance(ctx context.Context, twitchUserID string) (queries.GetBalanceRow, error) {
	balance, ok := m.balancesByUserId[twitchUserID]
	if !ok {
		return queries.GetBalanceRow{}, sql.ErrNoRows
	}
	return balance, nil
}

func (m *mockQueries) RecordPendingAlertRedemptionOutflow(ctx context.Context, arg queries.RecordPendingAlertRedemptionOutflowParams) (uuid.UUID, error) {
	id := m.generateId()
	m.alertRedemptions = append(m.alertRedemptions, mockAlertRedemptionOutflow{
		id:               id,
		userId:           arg.TwitchUserID,
		numPointsToDebit: arg.NumPointsToDebit,
		alertType:        arg.AlertType,
		alertMetadata:    arg.AlertMetadata,
	})
	return id, nil
}

func (m *mockQueries) GetFlow(ctx context.Context, flowID uuid.UUID) (queries.GetFlowRow, error) {
	for _, flow := range m.alertRedemptions {
		if flow.id == flowID {
			finalizedAt := sql.NullTime{}
			if flow.finalized {
				finalizedAt.Valid = true
				finalizedAt.Time = time.Now()
			}
			return queries.GetFlowRow{
				TwitchUserID: flow.userId,
				FinalizedAt:  finalizedAt,
				Accepted:     flow.accepted,
			}, nil
		}
	}
	return queries.GetFlowRow{}, sql.ErrNoRows
}

func (m *mockQueries) FinalizeFlow(ctx context.Context, arg queries.FinalizeFlowParams) (sql.Result, error) {
	for i := range m.alertRedemptions {
		flow := &m.alertRedemptions[i]
		if flow.id == arg.FlowID {
			if flow.finalized {
				return &mockSqlResult{0}, nil
			}
			flow.finalized = true
			flow.accepted = arg.Accepted
			return &mockSqlResult{1}, nil
		}
	}
	return &mockSqlResult{0}, nil
}

func (m *mockQueries) generateId() uuid.UUID {
	if m.nextIdIndex < len(m.idSequence) {
		i := m.nextIdIndex
		m.nextIdIndex++
		return m.idSequence[i]
	}
	return uuid.New()
}

type mockSqlResult struct {
	numRows int64
}

func (m *mockSqlResult) LastInsertId() (int64, error) {
	return 0, fmt.Errorf("not mocked")
}

func (m *mockSqlResult) RowsAffected() (int64, error) {
	return m.numRows, nil
}
