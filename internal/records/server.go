package records

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/golden-vcr/auth"
	"github.com/golden-vcr/ledger/gen/queries"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type Server struct {
	q Queries
}

func NewServer(q Queries) *Server {
	return &Server{
		q: q,
	}
}

func (s *Server) RegisterRoutes(c auth.Client, r *mux.Router) {
	r.Path("/balance").Methods("GET").Handler(
		auth.RequireAccess(c, auth.RoleViewer,
			http.HandlerFunc(s.handleGetBalance),
		),
	)
	r.Path("/history").Methods("GET").Handler(
		auth.RequireAccess(c, auth.RoleViewer,
			http.HandlerFunc(s.handleGetHistory),
		),
	)
}

func (s *Server) handleGetBalance(res http.ResponseWriter, req *http.Request) {
	// Identify the user making the request
	claims, err := auth.GetClaims(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	// Query their balance, defaulting to 0 if no record exists
	balance := &Balance{
		TotalPoints:     0,
		AvailablePoints: 0,
	}
	row, err := s.q.GetBalance(req.Context(), claims.User.Id)
	if err == nil {
		balance.TotalPoints = int(row.TotalPoints)
		balance.AvailablePoints = int(row.AvailablePoints)
	} else if err != sql.ErrNoRows {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return the Balance struct as a JSON object
	if err := json.NewEncoder(res).Encode(balance); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleGetHistory(res http.ResponseWriter, req *http.Request) {
	claims, err := auth.GetClaims(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	limit := 50
	maxStr := req.URL.Query().Get("max")
	if maxStr != "" {
		if maxValue, err := strconv.Atoi(maxStr); err == nil {
			limit = max(1, min(maxValue, 100))
		}
	}

	startId := uuid.NullUUID{}
	fromStr := req.URL.Query().Get("from")
	if fromStr != "" {
		if fromUUID, err := uuid.Parse(fromStr); err == nil {
			startId.Valid = true
			startId.UUID = fromUUID
		}
	}

	rows, err := s.q.GetTransactionHistory(req.Context(), queries.GetTransactionHistoryParams{
		TwitchUserID: claims.User.Id,
		NumRecords:   int32(limit + 1),
		StartID:      startId,
	})
	numItemsToReturn := min(limit, len(rows))
	items := make([]Transaction, 0, numItemsToReturn)
	for i := 0; i < numItemsToReturn; i++ {
		items = append(items, buildHistoryItem(&rows[i]))
	}
	nextCursor := ""
	if len(rows) > limit {
		nextCursor = rows[limit].ID.String()
	}

	history := &TransactionHistory{
		Items:      items,
		NextCursor: nextCursor,
	}
	if err := json.NewEncoder(res).Encode(history); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
}
