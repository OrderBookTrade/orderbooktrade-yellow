package engine

import (
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

// Side represents the order side (buy or sell)
type Side string

const (
	SideBuy  Side = "buy"
	SideSell Side = "sell"
)

// OutcomeID represents the outcome being traded
type OutcomeID string

const (
	OutcomeYes OutcomeID = "YES"
	OutcomeNo  OutcomeID = "NO"
)

// OrderStatus represents the current status of an order
type OrderStatus string

const (
	StatusOpen      OrderStatus = "open"
	StatusPartial   OrderStatus = "partial"
	StatusFilled    OrderStatus = "filled"
	StatusCancelled OrderStatus = "cancelled"
)

// Order represents a limit order in the orderbook
type Order struct {
	ID          string      `json:"id"`
	UserID      string      `json:"user_id"`
	MarketID    string      `json:"market_id"`  // Prediction market ID
	OutcomeID   OutcomeID   `json:"outcome_id"` // YES or NO
	Side        Side        `json:"side"`
	Price       uint64      `json:"price"`      // Price in basis points (0-10000 for 0.00-1.00 probability)
	Quantity    uint64      `json:"quantity"`   // Total quantity (shares)
	FilledQty   uint64      `json:"filled_qty"` // Already filled quantity
	Status      OrderStatus `json:"status"`
	Timestamp   time.Time   `json:"timestamp"`
	SequenceNum uint64      `json:"sequence_num"` // For FIFO ordering at same price
}

var orderSequence uint64

// NewOrder creates a new order with auto-generated ID and timestamp
func NewOrder(userID, marketID string, outcomeID OutcomeID, side Side, price, quantity uint64) *Order {
	return &Order{
		ID:          uuid.New().String(),
		UserID:      userID,
		MarketID:    marketID,
		OutcomeID:   outcomeID,
		Side:        side,
		Price:       price,
		Quantity:    quantity,
		FilledQty:   0,
		Status:      StatusOpen,
		Timestamp:   time.Now(),
		SequenceNum: atomic.AddUint64(&orderSequence, 1),
	}
}

// RemainingQty returns the unfilled quantity
func (o *Order) RemainingQty() uint64 {
	return o.Quantity - o.FilledQty
}

// Fill adds to the filled quantity and updates status
func (o *Order) Fill(qty uint64) {
	o.FilledQty += qty
	if o.FilledQty >= o.Quantity {
		o.Status = StatusFilled
	} else if o.FilledQty > 0 {
		o.Status = StatusPartial
	}
}

// Cancel marks the order as cancelled
func (o *Order) Cancel() {
	o.Status = StatusCancelled
}

// IsBuy returns true if this is a buy order
func (o *Order) IsBuy() bool {
	return o.Side == SideBuy
}
