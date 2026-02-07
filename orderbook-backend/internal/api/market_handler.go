package api

import (
	"encoding/json"
	"net/http"
	"time"

	"orderbook-backend/internal/engine"
	"orderbook-backend/internal/market"
)

// CreateMarketRequest is the request to create a new market
type CreateMarketRequest struct {
	Question    string `json:"question"`
	Description string `json:"description,omitempty"`
	ResolvesAt  string `json:"resolves_at"` // RFC3339 format
	CreatorID   string `json:"creator_id"`
}

// handleCreateMarket handles POST /api/market
func (s *Server) handleCreateMarket(w http.ResponseWriter, r *http.Request) {
	var req CreateMarketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Question == "" {
		writeError(w, http.StatusBadRequest, "question is required")
		return
	}

	resolvesAt, err := time.Parse(time.RFC3339, req.ResolvesAt)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid resolves_at format, use RFC3339")
		return
	}

	mkt, err := s.marketManager.Create(market.CreateMarketRequest{
		Question:    req.Question,
		Description: req.Description,
		ResolvesAt:  resolvesAt,
		CreatorID:   req.CreatorID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, mkt.ToJSON())
}

// handleListMarkets handles GET /api/markets
func (s *Server) handleListMarkets(w http.ResponseWriter, r *http.Request) {
	markets := s.marketManager.List()

	result := make([]market.MarketJSON, 0, len(markets))
	for _, m := range markets {
		result = append(result, m.ToJSON())
	}

	writeJSON(w, http.StatusOK, result)
}

// handleGetMarket handles GET /api/market/{id}
func (s *Server) handleGetMarket(w http.ResponseWriter, r *http.Request) {
	marketID := r.PathValue("id")
	if marketID == "" {
		writeError(w, http.StatusBadRequest, "market id required")
		return
	}

	mkt, ok := s.marketManager.Get(marketID)
	if !ok {
		writeError(w, http.StatusNotFound, "market not found")
		return
	}

	writeJSON(w, http.StatusOK, mkt.ToJSON())
}

// ResolveMarketRequest is the request to resolve a market
type ResolveMarketRequest struct {
	Outcome string `json:"outcome"` // "YES" or "NO"
}

// handleResolveMarket handles POST /api/market/{id}/resolve
func (s *Server) handleResolveMarket(w http.ResponseWriter, r *http.Request) {
	marketID := r.PathValue("id")
	if marketID == "" {
		writeError(w, http.StatusBadRequest, "market id required")
		return
	}

	var req ResolveMarketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var outcome market.Outcome
	switch req.Outcome {
	case "YES":
		outcome = market.OutcomeYes
	case "NO":
		outcome = market.OutcomeNo
	default:
		writeError(w, http.StatusBadRequest, "outcome must be 'YES' or 'NO'")
		return
	}

	// First lock the market
	if err := s.marketManager.Lock(marketID); err != nil {
		// Market might already be locked, which is fine
		if err != market.ErrInvalidTransition {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}

	// Resolve the market
	mkt, err := s.marketManager.Resolve(market.ResolveRequest{
		MarketID: marketID,
		Outcome:  outcome,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Payout winning shares to all position holders
	positions := s.positions.GetAllPositions(marketID)
	var totalPayout uint64
	for _, pos := range positions {
		var engineOutcome engine.OutcomeID
		if mkt.Outcome != nil && *mkt.Outcome == market.OutcomeYes {
			engineOutcome = engine.OutcomeYES
		} else {
			engineOutcome = engine.OutcomeNO
		}
		payout := s.positions.PayoutWinningShares(pos.UserID, marketID, engineOutcome)
		totalPayout += payout
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"market":       mkt.ToJSON(),
		"total_payout": totalPayout,
		"positions":    len(positions),
	})
}
