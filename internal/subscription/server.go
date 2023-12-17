package subscription

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"

	"github.com/golden-vcr/auth"
	"github.com/golden-vcr/ledger"
	"github.com/golden-vcr/ledger/gen/queries"
	"github.com/gorilla/mux"
)

const MaxStoredMessageLen = 128

type Server struct {
	q Queries
}

func NewServer(q Queries) *Server {
	return &Server{
		q: q,
	}
}

func (s *Server) RegisterRoutes(r *mux.Router, c auth.Client) {
	// Only internal services may call these endpoints by supplying the JWT they've been
	// issued by the auth service (with the 'authoritative' claim)
	r.Path("/inflow/subscription").Methods("POST").Handler(
		auth.RequireAuthority(c, http.HandlerFunc(s.handlePostSubscription)),
	)
	r.Path("/inflow/gift-sub").Methods("POST").Handler(
		auth.RequireAuthority(c, http.HandlerFunc(s.handlePostGiftSub)),
	)
}

func (s *Server) handlePostSubscription(res http.ResponseWriter, req *http.Request) {
	// Identify the user from the supplied JWT
	claims, err := auth.GetClaims(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}

	// The request's Content-Type must indicate JSON if set
	contentType := req.Header.Get("content-type")
	if contentType != "" && !strings.HasPrefix(contentType, "application/json") {
		http.Error(res, "content-type not supported", http.StatusBadRequest)
		return
	}

	// Parse the payload from the request body
	var payload ledger.SubscriptionRequest
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		http.Error(res, fmt.Sprintf("invalid request payload: %v", err), http.StatusBadRequest)
		return
	}
	if payload.BasePointsToCredit <= 0 {
		http.Error(res, "invalid request payload: 'basePointsToCredit' must be set to a positive integer", http.StatusBadRequest)
		return
	}
	if payload.CreditMultiplier <= 0 {
		http.Error(res, "invalid request payload: 'creditMultiplier' must be set to a positive number", http.StatusBadRequest)
		return
	}

	// Truncate the message if necessary
	message := payload.Message
	if len(message) > MaxStoredMessageLen {
		message = message[:MaxStoredMessageLen]
	}

	// Create a finalized flow record representing the inflow transaction that credits
	// our desired number of points to the target user
	numPointsToCredit := int32(math.Round(float64(payload.BasePointsToCredit) * payload.CreditMultiplier))
	flowId, err := s.q.RecordSubscriptionInflow(context.Background(), queries.RecordSubscriptionInflowParams{
		TwitchUserID:      claims.User.Id,
		NumPointsToCredit: numPointsToCredit,
		Message:           message,
		IsInitial:         payload.IsInitial,
		IsGift:            payload.IsGift,
		CreditMultiplier:  float64(payload.CreditMultiplier),
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

func (s *Server) handlePostGiftSub(res http.ResponseWriter, req *http.Request) {
	// Identify the user from the supplied JWT
	claims, err := auth.GetClaims(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}

	// The request's Content-Type must indicate JSON if set
	contentType := req.Header.Get("content-type")
	if contentType != "" && !strings.HasPrefix(contentType, "application/json") {
		http.Error(res, "content-type not supported", http.StatusBadRequest)
		return
	}

	// Parse the payload from the request body
	var payload ledger.GiftSubRequest
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		http.Error(res, fmt.Sprintf("invalid request payload: %v", err), http.StatusBadRequest)
		return
	}
	if payload.BasePointsToCredit <= 0 {
		http.Error(res, "invalid request payload: 'basePointsToCredit' must be set to a positive integer", http.StatusBadRequest)
		return
	}
	if payload.NumSubscriptions <= 0 {
		http.Error(res, "invalid request payload: 'numSubscriptions' must be set to a positive integer", http.StatusBadRequest)
		return
	}
	if payload.CreditMultiplier <= 0 {
		http.Error(res, "invalid request payload: 'creditMultiplier' must be set to a positive number", http.StatusBadRequest)
		return
	}

	// Create a finalized flow record representing the inflow transaction that credits
	// our desired number of points to the target user
	numPointsToCredit := payload.BasePointsToCredit * payload.NumSubscriptions * int(payload.CreditMultiplier)
	flowId, err := s.q.RecordGiftSubInflow(context.Background(), queries.RecordGiftSubInflowParams{
		TwitchUserID:      claims.User.Id,
		NumPointsToCredit: int32(numPointsToCredit),
		NumSubscriptions:  int32(payload.NumSubscriptions),
		CreditMultiplier:  float64(payload.CreditMultiplier),
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
