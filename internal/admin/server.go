package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/golden-vcr/auth"
	"github.com/golden-vcr/ledger/gen/queries"
	"github.com/gorilla/mux"
)

type Server struct {
	q                   Queries
	resolveTwitchUserId ResolveTwitchUserIdFunc
}

func NewServer(q Queries, twitchClientId string, twitchClientSecret string) *Server {
	return &Server{
		q:                   q,
		resolveTwitchUserId: makeResolveTwitchUserIdFunc(twitchClientId, twitchClientSecret),
	}
}

func (s *Server) RegisterRoutes(c auth.Client, r *mux.Router) {
	r.Path("/inflow/manual-credit").Methods("POST").Handler(
		auth.RequireAccess(c, auth.RoleBroadcaster,
			http.HandlerFunc(s.handlePostManualCredit),
		),
	)
}

func (s *Server) handlePostManualCredit(res http.ResponseWriter, req *http.Request) {
	// The request's Content-Type must indicate JSON if set
	contentType := req.Header.Get("content-type")
	if contentType != "" && !strings.HasPrefix(contentType, "application/json") {
		http.Error(res, "content-type not supported", http.StatusBadRequest)
		return
	}

	// Parse the payload from the request body
	var payload ManualCreditRequest
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		http.Error(res, fmt.Sprintf("invalid request payload: %v", err), http.StatusBadRequest)
		return
	}
	hasDisplayName := payload.TwitchDisplayName != ""
	hasUserId := payload.TwitchUserId != ""
	if hasDisplayName == hasUserId {
		http.Error(res, "invalid request payload: exactly one of 'twitchDisplayName' and 'twitchUserId' is required", http.StatusBadRequest)
		return
	}
	if payload.NumPointsToCredit <= 0 {
		http.Error(res, "invalid request payload: 'numPointsToCredit' must be set to a positive integer", http.StatusBadRequest)
		return
	}
	if payload.Note == "" {
		http.Error(res, "invalid request payload: 'note' must be set to a non-empty string", http.StatusBadRequest)
		return
	}

	// If the caller supplied a username instead of a user ID, resolve the corresponding
	// user ID using the Twitch API
	twitchUserId := payload.TwitchUserId
	if twitchUserId == "" {
		resolved, err := s.resolveTwitchUserId(req.Context(), payload.TwitchDisplayName)
		if err != nil {
			http.Error(res, fmt.Sprintf("failed to resolve twitch user ID from username: %v", err), http.StatusInternalServerError)
			return
		}
		twitchUserId = resolved
	}

	// Create a finalized flow record representing the inflow transaction that credits
	// our desired number of points to the target user
	flowId, err := s.q.RecordManualCreditInflow(context.Background(), queries.RecordManualCreditInflowParams{
		Note:              payload.Note,
		TwitchUserID:      twitchUserId,
		NumPointsToCredit: int32(payload.NumPointsToCredit),
	})
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return a JSON-serialized TransactionResult struct to the user
	result := &TransactionResult{FlowId: flowId}
	if err := json.NewEncoder(res).Encode(result); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
}
