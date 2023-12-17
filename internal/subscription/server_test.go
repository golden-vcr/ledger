package subscription

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

func Test_Server_handlePostSubscription(t *testing.T) {
	tests := []struct {
		name                  string
		q                     *mockQueries
		authorization         string
		body                  string
		wantStatus            int
		wantBody              string
		wantNumPointsCredited int32
	}{
		{
			"user purchases an initial sub",
			&mockQueries{},
			"internal-jwt",
			`{"basePointsToCredit":600,"isInitial":true,"isGift":false,"message":"","creditMultiplier":1}`,
			http.StatusOK,
			`{"flowId":"0dc95aba-6f8f-4e13-9081-ba1b2ced8f39"}`,
			600,
		},
		{
			"user receives a gift sub",
			&mockQueries{},
			"internal-jwt",
			`{"basePointsToCredit":600,"isInitial":true,"isGift":true,"message":"","creditMultiplier":1}`,
			http.StatusOK,
			`{"flowId":"0dc95aba-6f8f-4e13-9081-ba1b2ced8f39"}`,
			600,
		},
		{
			"user purchases an initial sub at Tier 3",
			&mockQueries{},
			"internal-jwt",
			`{"basePointsToCredit":600,"isInitial":true,"isGift":false,"message":"","creditMultiplier":5}`,
			http.StatusOK,
			`{"flowId":"0dc95aba-6f8f-4e13-9081-ba1b2ced8f39"}`,
			3000,
		},
		{
			"user receives a gift sub at Tier 2",
			&mockQueries{},
			"internal-jwt",
			`{"basePointsToCredit":600,"isInitial":true,"isGift":true,"message":"","creditMultiplier":2}`,
			http.StatusOK,
			`{"flowId":"0dc95aba-6f8f-4e13-9081-ba1b2ced8f39"}`,
			1200,
		},
		{
			"user resubscribes at Tier 2",
			&mockQueries{},
			"internal-jwt",
			`{"basePointsToCredit":600,"isInitial":false,"isGift":false,"message":"","creditMultiplier":2}`,
			http.StatusOK,
			`{"flowId":"0dc95aba-6f8f-4e13-9081-ba1b2ced8f39"}`,
			1200,
		},
	}
	for _, tt := range tests {
		c := authmock.NewClient().AllowAuthoritativeJWT("internal-jwt", auth.UserDetails{
			Id:          "1337",
			Login:       "leetman",
			DisplayName: "LEETman",
		})
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{
				q: tt.q,
			}
			handler := auth.RequireAuthority(c, http.HandlerFunc(s.handlePostSubscription))
			req := httptest.NewRequest(http.MethodPost, "/inflow/subscription", strings.NewReader(tt.body))
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
				assert.Len(t, tt.q.subscriptionCalls, 1)
				assert.Equal(t, tt.wantNumPointsCredited, tt.q.subscriptionCalls[0].NumPointsToCredit)
			} else {
				assert.Len(t, tt.q.subscriptionCalls, 0)
			}
		})
	}
}

func Test_Server_handlePostGiftSub(t *testing.T) {
	tests := []struct {
		name                  string
		q                     *mockQueries
		authorization         string
		body                  string
		wantStatus            int
		wantBody              string
		wantNumPointsCredited int32
	}{
		{
			"user gifts a single sub",
			&mockQueries{},
			"internal-jwt",
			`{"basePointsToCredit":200,"numSubscriptions":1,"creditMultiplier":1}`,
			http.StatusOK,
			`{"flowId":"0dc95aba-6f8f-4e13-9081-ba1b2ced8f39"}`,
			200,
		},
		{
			"user gifts ten subs",
			&mockQueries{},
			"internal-jwt",
			`{"basePointsToCredit":200,"numSubscriptions":10,"creditMultiplier":1}`,
			http.StatusOK,
			`{"flowId":"0dc95aba-6f8f-4e13-9081-ba1b2ced8f39"}`,
			2000,
		},
		{
			"user gifts four Tier 3 subs",
			&mockQueries{},
			"internal-jwt",
			`{"basePointsToCredit":200,"numSubscriptions":4,"creditMultiplier":5}`,
			http.StatusOK,
			`{"flowId":"0dc95aba-6f8f-4e13-9081-ba1b2ced8f39"}`,
			4000,
		},
		{
			"user gifts eight Tier 2 subs",
			&mockQueries{},
			"internal-jwt",
			`{"basePointsToCredit":200,"numSubscriptions":8,"creditMultiplier":2}`,
			http.StatusOK,
			`{"flowId":"0dc95aba-6f8f-4e13-9081-ba1b2ced8f39"}`,
			3200,
		},
	}
	for _, tt := range tests {
		c := authmock.NewClient().AllowAuthoritativeJWT("internal-jwt", auth.UserDetails{
			Id:          "1337",
			Login:       "leetman",
			DisplayName: "LEETman",
		})
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{
				q: tt.q,
			}
			handler := auth.RequireAuthority(c, http.HandlerFunc(s.handlePostGiftSub))
			req := httptest.NewRequest(http.MethodPost, "/inflow/gift-sub", strings.NewReader(tt.body))
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
				assert.Len(t, tt.q.giftSubCalls, 1)
				assert.Equal(t, tt.wantNumPointsCredited, tt.q.giftSubCalls[0].NumPointsToCredit)
			} else {
				assert.Len(t, tt.q.giftSubCalls, 0)
			}
		})
	}
}

type mockQueries struct {
	err               error
	subscriptionCalls []queries.RecordSubscriptionInflowParams
	giftSubCalls      []queries.RecordGiftSubInflowParams
}

func (m *mockQueries) RecordSubscriptionInflow(ctx context.Context, arg queries.RecordSubscriptionInflowParams) (uuid.UUID, error) {
	if m.err != nil {
		return uuid.UUID{}, m.err
	}
	m.subscriptionCalls = append(m.subscriptionCalls, arg)
	return uuid.MustParse("0dc95aba-6f8f-4e13-9081-ba1b2ced8f39"), nil
}

func (m *mockQueries) RecordGiftSubInflow(ctx context.Context, arg queries.RecordGiftSubInflowParams) (uuid.UUID, error) {
	if m.err != nil {
		return uuid.UUID{}, m.err
	}
	m.giftSubCalls = append(m.giftSubCalls, arg)
	return uuid.MustParse("0dc95aba-6f8f-4e13-9081-ba1b2ced8f39"), nil
}
