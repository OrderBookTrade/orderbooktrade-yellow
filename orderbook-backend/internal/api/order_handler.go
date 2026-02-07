package api

import (
	"encoding/json"
	"net/http"

	"orderbook-backend/internal/engine"
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
