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
		outcome = engine.OutcomeYes
	case "NO":
		outcome = engine.OutcomeNo
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

	// Place order and get trades
	trades, err := s.orderbook.PlaceOrder(order)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Execute trades (update positions)
	for _, trade := range trades {
		s.positions.ExecuteTrade(trade)
	}

	// Broadcast orderbook update
	s.broadcastOrderbook()

	writeJSON(w, http.StatusOK, PlaceOrderResponse{
		Order:  order,
		Trades: trades,
	})
}

// handleGetOrderbook handles GET /api/orderbook
func (s *Server) handleGetOrderbook(w http.ResponseWriter, r *http.Request) {
	snapshot := s.orderbook.GetSnapshot()
	writeJSON(w, http.StatusOK, snapshot)
}

// handleCancelOrder handles DELETE /api/order/{id}
func (s *Server) handleCancelOrder(w http.ResponseWriter, r *http.Request) {
	orderID := r.PathValue("id")
	if orderID == "" {
		writeError(w, http.StatusBadRequest, "order id required")
		return
	}

	if err := s.orderbook.CancelOrder(orderID); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	// Broadcast orderbook update
	s.broadcastOrderbook()

	writeJSON(w, http.StatusOK, map[string]string{
		"status":   "cancelled",
		"order_id": orderID,
	})
}

// handleGetTrades handles GET /api/trades
func (s *Server) handleGetTrades(w http.ResponseWriter, r *http.Request) {
	trades := s.orderbook.RecentTrades(100)
	writeJSON(w, http.StatusOK, trades)
}

// broadcastOrderbook sends the current orderbook state to all WebSocket clients
func (s *Server) broadcastOrderbook() {
	snapshot := s.orderbook.GetSnapshot()
	s.wsHub.Broadcast(Message{
		Type: "orderbook",
		Data: snapshot,
	})
}
