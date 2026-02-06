package engine

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// Trade represents a completed trade between two orders
type Trade struct {
	ID          string    `json:"id"`
	MarketID    string    `json:"market_id"`
	OutcomeID   OutcomeID `json:"outcome_id"` // YES or NO
	BuyOrderID  string    `json:"buy_order_id"`
	SellOrderID string    `json:"sell_order_id"`
	BuyerID     string    `json:"buyer_id"`
	SellerID    string    `json:"seller_id"`
	Price       uint64    `json:"price"`
	Quantity    uint64    `json:"quantity"`
	Timestamp   time.Time `json:"timestamp"`
}

// NewTrade creates a new trade record
func NewTrade(buyOrder, sellOrder *Order, price, quantity uint64) *Trade {
	return &Trade{
		ID:          uuid.New().String(),
		MarketID:    buyOrder.MarketID,
		OutcomeID:   buyOrder.OutcomeID,
		BuyOrderID:  buyOrder.ID,
		SellOrderID: sellOrder.ID,
		BuyerID:     buyOrder.UserID,
		SellerID:    sellOrder.UserID,
		Price:       price,
		Quantity:    quantity,
		Timestamp:   time.Now(),
	}
}

// TradeHistory stores all completed trades
type TradeHistory struct {
	mu     sync.RWMutex
	trades []*Trade
	maxLen int
}

// NewTradeHistory creates a new trade history with max capacity
func NewTradeHistory(maxLen int) *TradeHistory {
	return &TradeHistory{
		trades: make([]*Trade, 0, maxLen),
		maxLen: maxLen,
	}
}

// Add records a new trade
func (h *TradeHistory) Add(trade *Trade) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.trades = append(h.trades, trade)

	// Trim if exceeds max length
	if len(h.trades) > h.maxLen {
		h.trades = h.trades[len(h.trades)-h.maxLen:]
	}
}

// Recent returns the most recent n trades
func (h *TradeHistory) Recent(n int) []*Trade {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if n > len(h.trades) {
		n = len(h.trades)
	}

	result := make([]*Trade, n)
	copy(result, h.trades[len(h.trades)-n:])
	return result
}

// All returns all trades
func (h *TradeHistory) All() []*Trade {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make([]*Trade, len(h.trades))
	copy(result, h.trades)
	return result
}
