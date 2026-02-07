package engine

import "sync"

// OutcomeID represents a binary prediction outcome
type OutcomeID string

const (
	OutcomeYES OutcomeID = "YES"
	OutcomeNO  OutcomeID = "NO"
)

// MarketOrderbooks manages separate orderbooks for YES and NO outcomes
type MarketOrderbooks struct {
	mu         sync.RWMutex
	orderbooks map[string]*OutcomeOrderbooks // marketID -> outcome orderbooks
}

// OutcomeOrderbooks holds both YES and NO orderbooks for a single market
type OutcomeOrderbooks struct {
	YES *Orderbook
	NO  *Orderbook
}

// NewMarketOrderbooks creates a new market orderbooks manager
func NewMarketOrderbooks() *MarketOrderbooks {
	return &MarketOrderbooks{
		orderbooks: make(map[string]*OutcomeOrderbooks),
	}
}

// GetOrCreate returns the orderbooks for a market, creating them if needed
func (m *MarketOrderbooks) GetOrCreate(marketID string) *OutcomeOrderbooks {
	m.mu.Lock()
	defer m.mu.Unlock()

	if obs, exists := m.orderbooks[marketID]; exists {
		return obs
	}

	obs := &OutcomeOrderbooks{
		YES: NewOrderbook(),
		NO:  NewOrderbook(),
	}
	m.orderbooks[marketID] = obs
	return obs
}

// Get returns the orderbooks for a market, or nil if not found
func (m *MarketOrderbooks) Get(marketID string) *OutcomeOrderbooks {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.orderbooks[marketID]
}

// GetOrderbook returns a specific outcome's orderbook for a market
func (m *MarketOrderbooks) GetOrderbook(marketID string, outcome OutcomeID) *Orderbook {
	obs := m.GetOrCreate(marketID)
	if outcome == OutcomeYES {
		return obs.YES
	}
	return obs.NO
}

// SetTradeCallback sets trade callbacks for all orderbooks in a market
func (m *MarketOrderbooks) SetTradeCallback(marketID string, fn func(*Trade)) {
	obs := m.GetOrCreate(marketID)
	obs.YES.SetTradeCallback(fn)
	obs.NO.SetTradeCallback(fn)
}

// SetGlobalTradeCallback sets trade callback for all existing and future orderbooks
func (m *MarketOrderbooks) SetGlobalTradeCallback(fn func(*Trade)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, obs := range m.orderbooks {
		obs.YES.SetTradeCallback(fn)
		obs.NO.SetTradeCallback(fn)
	}
}
