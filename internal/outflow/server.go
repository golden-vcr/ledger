package outflow

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/golden-vcr/auth"
	"github.com/golden-vcr/ledger"
	"github.com/golden-vcr/ledger/gen/queries"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type Server struct {
	q Queries
}

func NewServer(q Queries, twitchClientId string, twitchClientSecret string) *Server {
	return &Server{
		q: q,
	}
}

func (s *Server) RegisterRoutes(c auth.Client, r *mux.Router) {
	r.Path("/outflow").Methods("POST").Handler(
		auth.RequireAccess(c, auth.RoleViewer,
			http.HandlerFunc(s.handleCreateOutflow),
		),
	)
	r.Path("/outflow/{id}").Methods("PATCH", "DELETE").Handler(
		auth.RequireAccess(c, auth.RoleViewer,
			http.HandlerFunc(s.handleFinalizeOutflow),
		),
	)
}

func (s *Server) handleCreateOutflow(res http.ResponseWriter, req *http.Request) {
	// Identify the user from the provided auth token: we can't create an outflow unless
	// we have an authenticated user to take the points from
	claims, err := auth.GetClaims(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	// Parse the request payload: we currently only support a single outflow type
	// ('alert-redemption'), so just expect that payload universally
	contentType := req.Header.Get("content-type")
	if contentType != "" && !strings.HasPrefix(contentType, "application/json") {
		http.Error(res, "content-type not supported", http.StatusBadRequest)
		return
	}
	var payload ledger.AlertRedemptionRequest
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		http.Error(res, fmt.Sprintf("invalid request payload: %v", err), http.StatusBadRequest)
		return
	}
	if payload.NumPointsToDebit <= 0 {
		http.Error(res, "numPointsToDebit must be positive", http.StatusBadRequest)
		return
	}

	// Sanity-check: we only support 'alert-redemption' at the moment
	if payload.Type != ledger.TransactionTypeAlertRedemption {
		http.Error(res, "unsupported transaction type", http.StatusBadRequest)
		return
	}

	// Verify that the auth'd user has enough points in their available balance
	balance, err := s.q.GetBalance(req.Context(), claims.User.Id)
	if errors.Is(err, sql.ErrNoRows) {
		balance.AvailablePoints = 0
		balance.TotalPoints = 0
		err = nil
	}
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	if balance.AvailablePoints < int32(payload.NumPointsToDebit) {
		http.Error(res, "not enough points", http.StatusConflict)
		return
	}

	// Record a new pending outflow in the database
	params := queries.RecordPendingAlertRedemptionOutflowParams{
		AlertType:        payload.AlertType,
		TwitchUserID:     claims.User.Id,
		NumPointsToDebit: int32(payload.NumPointsToDebit),
	}
	if payload.AlertMetadata != nil {
		params.AlertMetadata.Valid = true
		params.AlertMetadata.RawMessage = *payload.AlertMetadata
	}
	flowId, err := s.q.RecordPendingAlertRedemptionOutflow(req.Context(), params)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	// Build a response that includes the UUID of the newly-created transaction
	result := ledger.TransactionResult{
		FlowId: flowId,
	}
	if err := json.NewEncoder(res).Encode(result); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleFinalizeOutflow(res http.ResponseWriter, req *http.Request) {
	// Parse the target flowId from the URL
	idStr := mux.Vars(req)["id"]
	flowId, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	// Identify the user from the provided auth token: we only want to comply with the
	// request if the given transaction is associated with the auth'd user
	claims, err := auth.GetClaims(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get the details of the transaction with the given ID, and validate that it
	// belongs to that user and is not yet finalized
	row, err := s.q.GetFlow(req.Context(), flowId)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(res, "no such transaction", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	if row.TwitchUserID != claims.User.Id {
		http.Error(res, "no such transaction", http.StatusNotFound)
		return
	}
	if row.FinalizedAt.Valid {
		http.Error(res, "transaction is not pending", http.StatusConflict)
		return
	}

	// The HTTP method (PATCH or DELETE) indicates whether the transaction should be
	// accepted or rejected
	accepted := true
	if req.Method == http.MethodDelete {
		accepted = false
	}

	// Attempt to finalize the transaction, either making its effect permanent (if
	// accepted) or reverting any pending effect (if rejected)
	result, err := s.q.FinalizeFlow(context.Background(), queries.FinalizeFlowParams{
		Accepted: accepted,
		FlowID:   flowId,
	})
	numRows, err := result.RowsAffected()
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	if numRows != 1 {
		http.Error(res, fmt.Sprintf("FinalizeFlow affected %d rows; expected 1", numRows), http.StatusInternalServerError)
		return
	}
	res.WriteHeader(http.StatusNoContent)
}
