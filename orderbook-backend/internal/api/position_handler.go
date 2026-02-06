package api

import (
	"encoding/json"
	"net/http"

	"orderbook-backend/internal/engine"
)

// DepositRequest is the request to deposit USDC
type DepositRequest struct {
	UserID string `json:"user_id"`
	Amount uint64 `json:"amount"` // In basis points (10000 = 1 USDC)
}

// handleDeposit handles POST /api/deposit
func (s *Server) handleDeposit(w http.ResponseWriter, r *http.Request) {
	var req DepositRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.UserID == "" {
		writeError(w, http.StatusBadRequest, "user_id is required")
		return
	}
	if req.Amount == 0 {
		writeError(w, http.StatusBadRequest, "amount must be greater than 0")
		return
	}

	s.positions.Deposit(req.UserID, req.Amount)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"user_id": req.UserID,
		"balance": s.positions.GetBalance(req.UserID),
	})
}

// MintSharesRequest is the request to mint YES+NO shares
type MintSharesRequest struct {
	UserID   string `json:"user_id"`
	MarketID string `json:"market_id"`
	Amount   uint64 `json:"amount"` // Number of share pairs to mint (costs amount * 1 USDC)
}

// handleMintShares handles POST /api/mint
func (s *Server) handleMintShares(w http.ResponseWriter, r *http.Request) {
	var req MintSharesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate market exists
	if _, ok := s.marketManager.Get(req.MarketID); !ok {
		writeError(w, http.StatusNotFound, "market not found")
		return
	}

	if err := s.positions.MintShares(req.UserID, req.MarketID, req.Amount); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	pos := s.positions.GetPosition(req.UserID, req.MarketID)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"user_id":    req.UserID,
		"market_id":  req.MarketID,
		"yes_shares": pos.YesShares,
		"no_shares":  pos.NoShares,
		"balance":    s.positions.GetBalance(req.UserID),
	})
}

// handleGetPosition handles GET /api/position/{userId}
func (s *Server) handleGetPosition(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("userId")
	if userID == "" {
		writeError(w, http.StatusBadRequest, "userId required")
		return
	}

	marketID := r.URL.Query().Get("market_id")

	// Get balance
	balance := s.positions.GetBalance(userID)

	response := map[string]interface{}{
		"user_id": userID,
		"balance": balance,
	}

	// If market_id specified, get position for that market
	if marketID != "" {
		pos := s.positions.GetPosition(userID, marketID)
		response["position"] = &engine.Position{
			UserID:    userID,
			MarketID:  marketID,
			YesShares: pos.YesShares,
			NoShares:  pos.NoShares,
		}
	}

	writeJSON(w, http.StatusOK, response)
}
