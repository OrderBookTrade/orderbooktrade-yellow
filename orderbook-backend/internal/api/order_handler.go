package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"orderbook-backend/internal/engine"
	"orderbook-backend/internal/yellow"
)

// PlaceOrderRequest is the request body for placing an order
type PlaceOrderRequest struct {
	UserID    string `json:"user_id"`
	MarketID  string `json:"market_id"`
	OutcomeID string `json:"outcome_id"` // "YES" or "NO"
	Side      string `json:"side"`       // "buy" or "sell"
	Price     uint64 `json:"price"`      // 0-10000 basis points (0-100% probability)
	Quantity  uint64 `json:"quantity"`   // Number of shares
}

// PlaceOrderResponse is the response for a placed order
type PlaceOrderResponse struct {
	Order  *engine.Order   `json:"order"`
	Trades []*engine.Trade `json:"trades"`
}

// handlePlaceOrder handles POST /api/order
func (s *Server) handlePlaceOrder(w http.ResponseWriter, r *http.Request) {
	var req PlaceOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate market exists and is trading
	market, ok := s.marketManager.Get(req.MarketID)
	if !ok {
		writeError(w, http.StatusNotFound, "market not found")
		return
	}
	if market.Status != 0 { // StatusTrading = 0
		writeError(w, http.StatusBadRequest, "market is not accepting orders")
		return
	}

	// Validate side
	var side engine.Side
	switch req.Side {
	case "buy":
		side = engine.SideBuy
	case "sell":
		side = engine.SideSell
	default:
		writeError(w, http.StatusBadRequest, "invalid side: must be 'buy' or 'sell'")
		return
	}

	// Validate outcome
	var outcome engine.OutcomeID
	switch req.OutcomeID {
	case "YES":
		outcome = engine.OutcomeYES
	case "NO":
		outcome = engine.OutcomeNO
	default:
		writeError(w, http.StatusBadRequest, "invalid outcome_id: must be 'YES' or 'NO'")
		return
	}

	// Create order
	order := engine.NewOrder(req.UserID, req.MarketID, outcome, side, req.Price, req.Quantity)

	// Validate user can place this order (has balance/shares)
	if err := s.positions.ValidateOrder(order); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Get the correct orderbook for this market and outcome
	orderbook := s.marketOrderbooks.GetOrderbook(req.MarketID, outcome)

	// Place order and get trades
	trades, err := orderbook.PlaceOrder(order)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Execute trades (update positions)
	for _, trade := range trades {
		s.positions.ExecuteTrade(trade)
		// Broadcast each trade to WebSocket clients
		s.wsHub.Broadcast(Message{
			Type: "trade",
			Data: trade,
		})
	}

	// Update Yellow Network state channel if connected
	if len(trades) > 0 {
		s.updateYellowSession(r.Context(), req.MarketID)
	}

	// Broadcast orderbook update for this market
	s.broadcastOrderbookForMarket(req.MarketID)

	writeJSON(w, http.StatusOK, PlaceOrderResponse{
		Order:  order,
		Trades: trades,
	})
}

// handleGetOrderbook handles GET /api/orderbook?market_id=xxx&outcome=YES
func (s *Server) handleGetOrderbook(w http.ResponseWriter, r *http.Request) {
	marketID := r.URL.Query().Get("market_id")
	outcomeStr := r.URL.Query().Get("outcome")

	// Default to YES if not specified
	outcome := engine.OutcomeYES
	if outcomeStr == "NO" {
		outcome = engine.OutcomeNO
	}

	// Get orderbook for specific market and outcome
	orderbook := s.marketOrderbooks.GetOrderbook(marketID, outcome)
	snapshot := orderbook.GetSnapshot()

	// Add outcome info to response
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"outcome": string(outcome),
		"bids":    snapshot.Bids,
		"asks":    snapshot.Asks,
	})
}

// handleCancelOrder handles DELETE /api/order/{id}?market_id=xxx&outcome=YES
func (s *Server) handleCancelOrder(w http.ResponseWriter, r *http.Request) {
	orderID := r.PathValue("id")
	if orderID == "" {
		writeError(w, http.StatusBadRequest, "order id required")
		return
	}

	marketID := r.URL.Query().Get("market_id")
	outcomeStr := r.URL.Query().Get("outcome")

	outcome := engine.OutcomeYES
	if outcomeStr == "NO" {
		outcome = engine.OutcomeNO
	}

	orderbook := s.marketOrderbooks.GetOrderbook(marketID, outcome)
	if err := orderbook.CancelOrder(orderID); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	// Broadcast orderbook update
	s.broadcastOrderbookForMarket(marketID)

	writeJSON(w, http.StatusOK, map[string]string{
		"status":   "cancelled",
		"order_id": orderID,
	})
}

// handleGetTrades handles GET /api/trades?market_id=xxx&outcome=YES
func (s *Server) handleGetTrades(w http.ResponseWriter, r *http.Request) {
	marketID := r.URL.Query().Get("market_id")
	outcomeStr := r.URL.Query().Get("outcome")

	outcome := engine.OutcomeYES
	if outcomeStr == "NO" {
		outcome = engine.OutcomeNO
	}

	orderbook := s.marketOrderbooks.GetOrderbook(marketID, outcome)
	trades := orderbook.RecentTrades(100)
	writeJSON(w, http.StatusOK, trades)
}

// broadcastOrderbookForMarket sends both YES and NO orderbooks for a market
func (s *Server) broadcastOrderbookForMarket(marketID string) {
	obs := s.marketOrderbooks.Get(marketID)
	if obs == nil {
		return
	}

	yesSnapshot := obs.YES.GetSnapshot()
	noSnapshot := obs.NO.GetSnapshot()

	s.wsHub.Broadcast(Message{
		Type: "orderbook",
		Data: map[string]interface{}{
			"market_id": marketID,
			"YES": map[string]interface{}{
				"bids": yesSnapshot.Bids,
				"asks": yesSnapshot.Asks,
			},
			"NO": map[string]interface{}{
				"bids": noSnapshot.Bids,
				"asks": noSnapshot.Asks,
			},
		},
	})
}

// updateYellowSession updates the Yellow Network state channel after trades
func (s *Server) updateYellowSession(ctx context.Context, marketID string) {
	// Skip if Yellow Network is not connected
	if s.sessions == nil || s.yellowClient == nil {
		return
	}

	if !s.yellowClient.IsAuthenticated() {
		log.Printf("Yellow Network not authenticated, skipping state update")
		return
	}

	// Get all positions for this market and build allocations
	positions := s.positions.GetAllPositions(marketID)
	if len(positions) == 0 {
		return
	}

	// Build allocations from current positions
	allocations := make([]yellow.Allocation, 0)
	for _, pos := range positions {
		// Convert position to allocation
		// In real implementation, this would track actual token balances
		totalValue := pos.YesShares + pos.NoShares
		if totalValue > 0 {
			allocations = append(allocations, yellow.Allocation{
				Participant: pos.UserID,
				Token:       s.cfg.DefaultToken,
				Amount:      fmt.Sprintf("%d", totalValue),
			})
		}
	}

	// Get or create session for this market
	session, exists := s.sessions.GetSession(marketID)
	if !exists {
		// Create new session for this market
		participants := make([]string, 0, len(allocations))
		for _, alloc := range allocations {
			participants = append(participants, alloc.Participant)
		}

		var err error
		session, err = s.sessions.CreateSession(ctx, participants, allocations, s.cfg.AdjudicatorAddr)
		if err != nil {
			log.Printf("Failed to create Yellow session for market %s: %v", marketID, err)
			return
		}
		log.Printf("Created Yellow session for market %s: %s", marketID, session.GetChannelID())
	}

	// Build orderbook snapshot as appData
	obs := s.marketOrderbooks.Get(marketID)
	appData := ""
	if obs != nil {
		yesSnapshot := obs.YES.GetSnapshot()
		noSnapshot := obs.NO.GetSnapshot()
		appDataBytes, _ := json.Marshal(map[string]interface{}{
			"market_id": marketID,
			"YES":       yesSnapshot,
			"NO":        noSnapshot,
		})
		appData = string(appDataBytes)
	}

	// Update state channel
	if err := session.UpdateState(ctx, allocations, appData); err != nil {
		log.Printf("Failed to update Yellow session state for market %s: %v", marketID, err)
		return
	}

	log.Printf("Updated Yellow session state for market %s (version %d)", marketID, session.GetChannelID())
}
