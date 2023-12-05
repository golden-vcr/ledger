package cheer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/golden-vcr/ledger"
	"github.com/golden-vcr/ledger/gen/queries"
	"github.com/gorilla/mux"
)

const MaxStoredMessageLen = 128

type Server struct {
	q                    Queries
	authorizeCheerInflow AuthorizeCheerInflowFunc
}

func NewServer(q Queries, secretKey string) *Server {
	return &Server{
		q:                    q,
		authorizeCheerInflow: makeAuthorizeCheerInflowFunc(secretKey),
	}
}

func (s *Server) RegisterRoutes(r *mux.Router) {
	r.Path("/inflow/cheer").Methods("POST").HandlerFunc(s.handlePostCheer)
}

func (s *Server) handlePostCheer(res http.ResponseWriter, req *http.Request) {
	// The caller (showtime) must supply the super secret magic value that's known only
	// to it and the ledger service
	if !s.authorizeCheerInflow(req.Header.Get("authorization")) {
		http.Error(res, "access denied", http.StatusUnauthorized)
		return
	}

	// The request's Content-Type must indicate JSON if set
	contentType := req.Header.Get("content-type")
	if contentType != "" && !strings.HasPrefix(contentType, "application/json") {
		http.Error(res, "content-type not supported", http.StatusBadRequest)
		return
	}

	// Parse the payload from the request body
	var payload ledger.CheerRequest
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		http.Error(res, fmt.Sprintf("invalid request payload: %v", err), http.StatusBadRequest)
		return
	}
	if payload.TwitchUserId == "" {
		http.Error(res, "invalid request payload: 'twitchUserId' is required", http.StatusBadRequest)
		return
	}
	if payload.NumPointsToCredit <= 0 {
		http.Error(res, "invalid request payload: 'numPointsToCredit' must be set to a positive integer", http.StatusBadRequest)
		return
	}

	// Truncate the message if necessary
	message := payload.Message
	if len(message) > MaxStoredMessageLen {
		message = message[:MaxStoredMessageLen]
	}

	// Create a finalized flow record representing the inflow transaction that credits
	// our desired number of points to the target user
	flowId, err := s.q.RecordCheerInflow(context.Background(), queries.RecordCheerInflowParams{
		TwitchUserID:      payload.TwitchUserId,
		NumPointsToCredit: int32(payload.NumPointsToCredit),
		Message:           message,
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
