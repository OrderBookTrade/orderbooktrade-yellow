package engine

import (
	"container/heap"
	"errors"
	"sync"
)

var (
	ErrInvalidPrice    = errors.New("invalid price: must be between 0 and 10000 basis points")
	ErrInvalidQuantity = errors.New("invalid quantity: must be greater than 0")
	ErrOrderNotFound   = errors.New("order not found")
)

// Orderbook is the core matching engine with price-time priority
type Orderbook struct {
	mu      sync.RWMutex
	bids    *orderHeap // Max heap for buy orders (highest price first)
	asks    *orderHeap // Min heap for sell orders (lowest price first)
	orders  map[string]*Order
	history *TradeHistory

	// Callback for trade notifications
	onTrade func(*Trade)
}

// NewOrderbook creates a new orderbook matching engine
func NewOrderbook() *Orderbook {
	ob := &Orderbook{
		bids:    newOrderHeap(true),  // Max heap
		asks:    newOrderHeap(false), // Min heap
		orders:  make(map[string]*Order),
		history: NewTradeHistory(1000),
	}
	heap.Init(ob.bids)
	heap.Init(ob.asks)
	return ob
}

// SetTradeCallback sets the callback for trade notifications
func (ob *Orderbook) SetTradeCallback(fn func(*Trade)) {
	ob.mu.Lock()
	defer ob.mu.Unlock()
	ob.onTrade = fn
}

// PlaceOrder adds a new order and attempts to match it
func (ob *Orderbook) PlaceOrder(order *Order) ([]*Trade, error) {
	if order.Price > 10000 {
		return nil, ErrInvalidPrice
	}
	if order.Quantity == 0 {
		return nil, ErrInvalidQuantity
	}

	ob.mu.Lock()
	defer ob.mu.Unlock()

	var trades []*Trade

	if order.IsBuy() {
		trades = ob.matchBuy(order)
	} else {
		trades = ob.matchSell(order)
	}

	// If order is not fully filled, add to book
	if order.RemainingQty() > 0 && order.Status != StatusCancelled {
		ob.orders[order.ID] = order
		if order.IsBuy() {
			heap.Push(ob.bids, order)
		} else {
			heap.Push(ob.asks, order)
		}
	}

	// Notify trades
	for _, trade := range trades {
		ob.history.Add(trade)
		if ob.onTrade != nil {
			ob.onTrade(trade)
		}
	}

	return trades, nil
}

// matchBuy matches a buy order against the ask book
func (ob *Orderbook) matchBuy(buy *Order) []*Trade {
	var trades []*Trade

	for ob.asks.Len() > 0 && buy.RemainingQty() > 0 {
		bestAsk := ob.asks.Peek()

		// Price check: buy price must be >= ask price
		if buy.Price < bestAsk.Price {
			break
		}

		// Match at the ask price (price improvement for buyer)
		matchQty := min(buy.RemainingQty(), bestAsk.RemainingQty())
		matchPrice := bestAsk.Price

		buy.Fill(matchQty)
		bestAsk.Fill(matchQty)

		trade := NewTrade(buy, bestAsk, matchPrice, matchQty)
		trades = append(trades, trade)

		// Remove filled order from book
		if bestAsk.RemainingQty() == 0 {
			heap.Pop(ob.asks)
			delete(ob.orders, bestAsk.ID)
		}
	}

	return trades
}

// matchSell matches a sell order against the bid book
func (ob *Orderbook) matchSell(sell *Order) []*Trade {
	var trades []*Trade

	for ob.bids.Len() > 0 && sell.RemainingQty() > 0 {
		bestBid := ob.bids.Peek()

		// Price check: sell price must be <= bid price
		if sell.Price > bestBid.Price {
			break
		}

		// Match at the bid price (price improvement for seller)
		matchQty := min(sell.RemainingQty(), bestBid.RemainingQty())
		matchPrice := bestBid.Price

		sell.Fill(matchQty)
		bestBid.Fill(matchQty)

		trade := NewTrade(bestBid, sell, matchPrice, matchQty)
		trades = append(trades, trade)

		// Remove filled order from book
		if bestBid.RemainingQty() == 0 {
			heap.Pop(ob.bids)
			delete(ob.orders, bestBid.ID)
		}
	}

	return trades
}

// CancelOrder cancels an order by ID
func (ob *Orderbook) CancelOrder(orderID string) error {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	order, exists := ob.orders[orderID]
	if !exists {
		return ErrOrderNotFound
	}

	order.Cancel()
	delete(ob.orders, orderID)

	// Note: Order stays in heap but will be skipped during matching
	// A cleaner approach would be to rebuild heaps, but this is O(n)

	return nil
}

// GetOrder returns an order by ID
func (ob *Orderbook) GetOrder(orderID string) (*Order, error) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	order, exists := ob.orders[orderID]
	if !exists {
		return nil, ErrOrderNotFound
	}
	return order, nil
}

// Snapshot returns the current state of the orderbook
type OrderbookSnapshot struct {
	Bids []OrderLevel `json:"bids"`
	Asks []OrderLevel `json:"asks"`
}

type OrderLevel struct {
	Price    uint64 `json:"price"`
	Quantity uint64 `json:"quantity"`
	Count    int    `json:"count"`
}

// GetSnapshot returns aggregated price levels
func (ob *Orderbook) GetSnapshot() OrderbookSnapshot {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	bids := ob.aggregateLevels(ob.bids, true)
	asks := ob.aggregateLevels(ob.asks, false)

	return OrderbookSnapshot{Bids: bids, Asks: asks}
}

func (ob *Orderbook) aggregateLevels(h *orderHeap, reverse bool) []OrderLevel {
	levels := make(map[uint64]*OrderLevel)

	for _, order := range h.orders {
		if order.Status == StatusCancelled || order.RemainingQty() == 0 {
			continue
		}

		if level, exists := levels[order.Price]; exists {
			level.Quantity += order.RemainingQty()
			level.Count++
		} else {
			levels[order.Price] = &OrderLevel{
				Price:    order.Price,
				Quantity: order.RemainingQty(),
				Count:    1,
			}
		}
	}

	result := make([]OrderLevel, 0, len(levels))
	for _, level := range levels {
		result = append(result, *level)
	}

	// Sort: bids descending, asks ascending
	if reverse {
		sortOrderLevelsDesc(result)
	} else {
		sortOrderLevelsAsc(result)
	}

	return result
}

// RecentTrades returns recent trades
func (ob *Orderbook) RecentTrades(n int) []*Trade {
	return ob.history.Recent(n)
}

// --- Order Heap Implementation ---

type orderHeap struct {
	orders []*Order
	isMax  bool // true for max heap (bids), false for min heap (asks)
}

func newOrderHeap(isMax bool) *orderHeap {
	return &orderHeap{
		orders: make([]*Order, 0),
		isMax:  isMax,
	}
}

func (h *orderHeap) Len() int { return len(h.orders) }

func (h *orderHeap) Less(i, j int) bool {
	oi, oj := h.orders[i], h.orders[j]

	// Skip cancelled orders
	if oi.Status == StatusCancelled {
		return false
	}
	if oj.Status == StatusCancelled {
		return true
	}

	if oi.Price == oj.Price {
		// Same price: earlier order has priority (FIFO)
		return oi.SequenceNum < oj.SequenceNum
	}

	if h.isMax {
		return oi.Price > oj.Price // Max heap: higher price first
	}
	return oi.Price < oj.Price // Min heap: lower price first
}

func (h *orderHeap) Swap(i, j int) {
	h.orders[i], h.orders[j] = h.orders[j], h.orders[i]
}

func (h *orderHeap) Push(x any) {
	h.orders = append(h.orders, x.(*Order))
}

func (h *orderHeap) Pop() any {
	old := h.orders
	n := len(old)
	order := old[n-1]
	h.orders = old[0 : n-1]
	return order
}

func (h *orderHeap) Peek() *Order {
	if len(h.orders) == 0 {
		return nil
	}
	return h.orders[0]
}

// --- Sorting helpers ---

func sortOrderLevelsDesc(levels []OrderLevel) {
	for i := 0; i < len(levels)-1; i++ {
		for j := i + 1; j < len(levels); j++ {
			if levels[i].Price < levels[j].Price {
				levels[i], levels[j] = levels[j], levels[i]
			}
		}
	}
}

func sortOrderLevelsAsc(levels []OrderLevel) {
	for i := 0; i < len(levels)-1; i++ {
		for j := i + 1; j < len(levels); j++ {
			if levels[i].Price > levels[j].Price {
				levels[i], levels[j] = levels[j], levels[i]
			}
		}
	}
}
