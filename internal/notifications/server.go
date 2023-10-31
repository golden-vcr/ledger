package notifications

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golden-vcr/auth"
	"github.com/golden-vcr/ledger"
	"github.com/golden-vcr/ledger/gen/queries"
	"github.com/golden-vcr/ledger/internal/util"
	"github.com/gorilla/mux"
)

type Server struct {
	ctx           context.Context
	q             Queries
	generateToken GenerateTokenFunc
	eventsChan    <-chan *FlowChangeNotification
	subscribers   subscriberChannels
}

func NewServer(ctx context.Context, q Queries, eventsChan <-chan *FlowChangeNotification) *Server {
	return &Server{
		ctx:           ctx,
		q:             q,
		generateToken: generateToken,
		eventsChan:    eventsChan,
		subscribers: subscriberChannels{
			chans: make(map[string][]chan *ledger.Transaction),
		},
	}
}

func (s *Server) RegisterRoutes(c auth.Client, r *mux.Router) {
	r.Path("/notifications").Methods("POST").Handler(
		auth.RequireAccess(c, auth.RoleViewer,
			http.HandlerFunc(s.handlePostNotifications),
		),
	)
	r.Path("/notifications").Methods("GET").HandlerFunc(s.handleGetNotifications)
}

func (s *Server) ReadPostgresNotifications(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-s.eventsChan:
			finalizedAt := sql.NullTime{}
			if event.FinalizedAt != nil {
				finalizedAt.Valid = true
				finalizedAt.Time = *event.FinalizedAt
			}
			transaction := util.BuildTransaction(event.Id, event.Type, event.Metadata, event.DeltaPoints, event.CreatedAt, finalizedAt, event.Accepted)
			s.subscribers.broadcast(event.TwitchUserId, &transaction)
		}
	}
}

func (s *Server) handlePostNotifications(res http.ResponseWriter, req *http.Request) {
	// Identify the user from their Twitch user access token
	claims, err := auth.GetClaims(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	// Generate a random, cryptographically secure token which can be used as a
	// short-lived auth mechanism: the user can supply this to the SSE endpoint
	// (GET /notifications) as a URL parameter, bypassing EventSource API's lack of
	// support for Authorization header. The token expires after a few minutes and can
	// only be used to subscribe to transaction history events; it does not grant access
	// to any other resources.
	token, err := s.generateToken()
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	// Delete any old SSE tokens that have since expired, then store this new token in
	// the database so we can look up our user ID when presented with the same token
	// later (as long as it's within our TTL window)
	if err := s.q.PurgeSseTokensForUser(req.Context(), claims.User.Id); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := s.q.StoreSseToken(req.Context(), queries.StoreSseTokenParams{
		TwitchUserID: claims.User.Id,
		TokenValue:   token,
		TtlSeconds:   600,
	}); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	// Respond with the plain-text token value in the response body
	res.Header().Add("content-type", "text/plain")
	res.Write([]byte(token))
}

func (s *Server) handleGetNotifications(res http.ResponseWriter, req *http.Request) {
	// If a content-type is explicitly requested, require that it's text/event-stream
	accept := req.Header.Get("accept")
	if accept != "" && accept != "*/*" && !strings.HasPrefix(accept, "text/event-stream") {
		message := fmt.Sprintf("content-type %s is not supported", accept)
		http.Error(res, message, http.StatusBadRequest)
		return
	}

	token := req.URL.Query().Get("token")
	if token == "" {
		http.Error(res, "'token' URL parameter must be set", http.StatusUnauthorized)
		return
	}
	twitchUserId, err := s.q.IdentifyUserFromSseToken(context.Background(), token)
	if err == sql.ErrNoRows {
		http.Error(res, "invalid token", http.StatusUnauthorized)
		return
	}
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	transactionsChan := s.subscribers.register(twitchUserId)
	defer s.subscribers.unregister(twitchUserId, transactionsChan)

	// Keep the connection alive and open a text/event-stream response body
	res.Header().Set("content-type", "text/event-stream")
	res.Header().Set("cache-control", "no-cache")
	res.Header().Set("connection", "keep-alive")
	res.WriteHeader(http.StatusOK)
	res.(http.Flusher).Flush()

	// Send an initial empty value to flush the connection and ensure that any
	// intermediaries (Cloudflare etc) will send the initial HTTP response promptly
	res.Write([]byte(":\n\n"))
	res.(http.Flusher).Flush()

	// Send all incoming messages to the client for as long as the connection is open
	fmt.Printf("Opened SSE connection to %s...\n", req.RemoteAddr)
	for {
		select {
		case <-time.After(30 * time.Second):
			res.Write([]byte(":\n\n"))
			res.(http.Flusher).Flush()
		case transaction := <-transactionsChan:
			data, err := json.Marshal(transaction)
			if err != nil {
				fmt.Printf("Failed to serialize transaction as JSON: %v\n", err)
				continue
			}
			fmt.Fprintf(res, "data: %s\n\n", data)
			res.(http.Flusher).Flush()
		case <-s.ctx.Done():
			fmt.Printf("Server is shutting down; abandoning SSE connection to %s.\n", req.RemoteAddr)
			return
		case <-req.Context().Done():
			fmt.Printf("SSE connection to %s has been closed.\n", req.RemoteAddr)
			return
		}
	}
}
