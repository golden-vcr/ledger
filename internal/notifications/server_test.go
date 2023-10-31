package notifications

import (
	"context"
	"database/sql"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golden-vcr/auth"
	authmock "github.com/golden-vcr/auth/mock"
	"github.com/golden-vcr/ledger"
	"github.com/golden-vcr/ledger/gen/queries"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func Test_Server_handlePostNotifications(t *testing.T) {
	tests := []struct {
		name                string
		q                   *mockQueries
		authorization       string
		wantStatus          int
		wantBody            string
		wantNumTokensStored int
	}{
		{
			"returns a short-lived SSE token (stored in DB) if properly authorized",
			&mockQueries{},
			"mock-token",
			http.StatusOK,
			"mock-sse-token",
			1,
		},
		{
			"purges outdated tokens for auth'd user",
			&mockQueries{
				tokens: []mockSseToken{
					{
						userId:    "1001",
						value:     "old-sse-token",
						expiresAt: time.Now().Add(-4 * time.Hour),
					},
				},
			},
			"mock-token",
			http.StatusOK,
			"mock-sse-token",
			1,
		},
		{
			"refuses to issue a token if not auth'd",
			&mockQueries{},
			"bad-token",
			http.StatusUnauthorized,
			"access token was not accepted",
			0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authClient := authmock.NewClient().Allow("mock-token", auth.RoleViewer, auth.UserDetails{
				Id:          "1001",
				Login:       "testuser",
				DisplayName: "TestUser",
			})
			s := &Server{
				ctx:           context.Background(),
				q:             tt.q,
				generateToken: mockGenerateToken,
			}
			f := http.HandlerFunc(s.handlePostNotifications)
			handler := auth.RequireAccess(authClient, auth.RoleViewer, f)

			req := httptest.NewRequest(http.MethodPost, "/notifications", nil)
			req.Header.Set("authorization", tt.authorization)
			res := httptest.NewRecorder()
			handler.ServeHTTP(res, req)

			b, err := io.ReadAll(res.Body)
			assert.NoError(t, err)
			body := strings.TrimSuffix(string(b), "\n")
			assert.Equal(t, tt.wantStatus, res.Code)
			assert.Equal(t, tt.wantBody, body)

			if tt.wantNumTokensStored > 0 {
				assert.Len(t, tt.q.tokens, tt.wantNumTokensStored)
				assert.Equal(t, "1001", tt.q.tokens[0].userId)
				assert.Equal(t, "mock-sse-token", tt.q.tokens[0].value)
			} else {
				assert.Empty(t, tt.q.tokens)
			}
		})
	}
}

func Test_Server_handleGetNotifications(t *testing.T) {
	tests := []struct {
		name                     string
		q                        *mockQueries
		url                      string
		generateNotificationFunc func(ch chan *FlowChangeNotification)
		wantStatus               int
		wantBody                 string
	}{
		{
			"returns 401 if no sse token is provided",
			&mockQueries{},
			"/notifications",
			func(ch chan *FlowChangeNotification) {},
			http.StatusUnauthorized,
			"'token' URL parameter must be set",
		},
		{
			"returns 401 if provided sse token is invalid",
			&mockQueries{},
			"/notifications?token=bad-sse-token",
			func(ch chan *FlowChangeNotification) {},
			http.StatusUnauthorized,
			"invalid token",
		},
		{
			"accepts sse token in URL and provides access to associated user's real-time transaction events",
			&mockQueries{
				tokens: []mockSseToken{
					{
						userId:    "1001",
						value:     "mock-sse-token",
						expiresAt: time.Now().Add(5 * time.Minute),
					},
				},
			},
			"/notifications?token=mock-sse-token",
			func(ch chan *FlowChangeNotification) {
				ch <- &FlowChangeNotification{
					TwitchUserId: "1001",
					Id:           uuid.MustParse("ffc921c7-24da-4f1d-9d0d-0d7c17d0a8b6"),
					Type:         "manual-credit",
					Metadata:     []byte(`{"note":"foo"}`),
					DeltaPoints:  150,
					CreatedAt:    time.Date(1997, 9, 1, 12, 0, 0, 0, time.UTC),
				}
			},
			http.StatusOK,
			":\n\ndata: {\"id\":\"ffc921c7-24da-4f1d-9d0d-0d7c17d0a8b6\",\"timestamp\":\"1997-09-01T12:00:00Z\",\"type\":\"manual-credit\",\"state\":\"pending\",\"deltaPoints\":150,\"description\":\"Manual credit: foo\"}\n\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventsChan := make(chan *FlowChangeNotification)
			s := &Server{
				ctx:        context.Background(),
				q:          tt.q,
				eventsChan: eventsChan,
				subscribers: subscriberChannels{
					chans: make(map[string][]chan *ledger.Transaction),
				},
			}

			// Prepare a context that we can cancel in order to shut down all server
			// processing
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Read from eventsChan and fan out to all connected SSE clients for as long
			// as that context is alive
			go s.ReadPostgresNotifications(ctx)

			// Preemptively clear our status code, then run our SSE request handler in
			// another goroutine until our context is canceled
			req := httptest.NewRequest(http.MethodGet, tt.url, nil).WithContext(ctx)
			res := httptest.NewRecorder()
			res.Code = 0
			done := make(chan struct{})
			go func() {
				s.handleGetNotifications(res, req)
				done <- struct{}{}
			}()

			// Wait until we get an initial response from the server
			for res.Code == 0 {
				time.Sleep(10 * time.Nanosecond)
			}

			// If this test expects an error response, validate it and go no further
			if tt.wantStatus != http.StatusOK {
				cancel()
				<-done
				assert.Equal(t, tt.wantStatus, res.Code)
				b, err := io.ReadAll(res.Body)
				assert.NoError(t, err)
				body := strings.TrimSuffix(string(b), "\n")
				assert.Equal(t, tt.wantBody, body)
				return
			}
			if res.Code != http.StatusOK {
				t.Fatalf("did not get 200 response")
			}

			// We got a 200 response as expected; validate that we're receiving SSE data
			contentType := res.Header().Get("content-type")
			assert.Equal(t, "text/event-stream", contentType)

			// Simulate postgres notifications, then wait for the response to propagate
			// over HTTP and verify that we got the expected message(s)
			tt.generateNotificationFunc(eventsChan)
			time.Sleep(10 * time.Millisecond)
			cancel()
			<-done
			b, err := io.ReadAll(res.Body)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantBody, string(b))
		})
	}
}

func mockGenerateToken() (string, error) {
	return "mock-sse-token", nil
}

type mockQueries struct {
	tokens []mockSseToken
}

type mockSseToken struct {
	userId    string
	value     string
	expiresAt time.Time
}

func (m *mockQueries) StoreSseToken(ctx context.Context, arg queries.StoreSseTokenParams) error {
	m.tokens = append(m.tokens, mockSseToken{
		userId:    arg.TwitchUserID,
		value:     arg.TokenValue,
		expiresAt: time.Now().Add(time.Duration(arg.TtlSeconds * int32(time.Second))),
	})
	return nil
}

func (m *mockQueries) PurgeSseTokensForUser(ctx context.Context, twitchUserID string) error {
	tokensToKeep := make([]mockSseToken, 0, len(m.tokens))
	for _, token := range m.tokens {
		if token.expiresAt.After(time.Now()) {
			tokensToKeep = append(tokensToKeep, token)
		}
	}
	m.tokens = tokensToKeep
	return nil
}

func (m *mockQueries) IdentifyUserFromSseToken(ctx context.Context, tokenValue string) (string, error) {
	for _, token := range m.tokens {
		if token.value == tokenValue && token.expiresAt.After(time.Now()) {
			return token.userId, nil
		}
	}
	return "", sql.ErrNoRows
}
