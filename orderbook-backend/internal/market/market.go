package market

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// MarketStatus represents the lifecycle stage of a prediction market
type MarketStatus int

const (
	StatusTrading  MarketStatus = iota // Accepting orders
	StatusLocked                       // No more orders, awaiting resolution
	StatusResolved                     // Outcome determined, payouts ready
)

func (s MarketStatus) String() string {
	switch s {
	case StatusTrading:
		return "trading"
	case StatusLocked:
		return "locked"
	case StatusResolved:
		return "resolved"
	default:
		return "unknown"
	}
}

// Outcome represents the possible outcomes of a binary market
type Outcome string

const (
	OutcomeYes Outcome = "YES"
	OutcomeNo  Outcome = "NO"
)

// Market represents a binary prediction market
type Market struct {
	ID          string       `json:"id"`
	Question    string       `json:"question"`
	Description string       `json:"description,omitempty"`
	Status      MarketStatus `json:"status"`
	Outcome     *Outcome     `json:"outcome,omitempty"` // nil until resolved
	CreatedAt   time.Time    `json:"created_at"`
	ResolvesAt  time.Time    `json:"resolves_at"` // When trading locks
	ResolvedAt  *time.Time   `json:"resolved_at,omitempty"`
	CreatorID   string       `json:"creator_id"`
}

// MarketJSON is the JSON representation of a market
type MarketJSON struct {
	ID          string  `json:"id"`
	Question    string  `json:"question"`
	Description string  `json:"description,omitempty"`
	Status      string  `json:"status"`
	Outcome     *string `json:"outcome,omitempty"`
	CreatedAt   string  `json:"created_at"`
	ResolvesAt  string  `json:"resolves_at"`
	ResolvedAt  *string `json:"resolved_at,omitempty"`
	CreatorID   string  `json:"creator_id"`
}

// ToJSON converts a Market to its JSON representation
func (m *Market) ToJSON() MarketJSON {
	mj := MarketJSON{
		ID:          m.ID,
		Question:    m.Question,
		Description: m.Description,
		Status:      m.Status.String(),
		CreatedAt:   m.CreatedAt.Format(time.RFC3339),
		ResolvesAt:  m.ResolvesAt.Format(time.RFC3339),
		CreatorID:   m.CreatorID,
	}
	if m.Outcome != nil {
		s := string(*m.Outcome)
		mj.Outcome = &s
	}
	if m.ResolvedAt != nil {
		s := m.ResolvedAt.Format(time.RFC3339)
		mj.ResolvedAt = &s
	}
	return mj
}

// Manager manages all prediction markets
type Manager struct {
	mu      sync.RWMutex
	markets map[string]*Market
}

// NewManager creates a new market manager
func NewManager() *Manager {
	return &Manager{
		markets: make(map[string]*Market),
	}
}

// CreateMarketRequest is the request to create a new market
type CreateMarketRequest struct {
	Question    string    `json:"question"`
	Description string    `json:"description,omitempty"`
	ResolvesAt  time.Time `json:"resolves_at"`
	CreatorID   string    `json:"creator_id"`
}

// Create creates a new prediction market
func (m *Manager) Create(req CreateMarketRequest) (*Market, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	market := &Market{
		ID:          uuid.New().String(),
		Question:    req.Question,
		Description: req.Description,
		Status:      StatusTrading,
		CreatedAt:   time.Now(),
		ResolvesAt:  req.ResolvesAt,
		CreatorID:   req.CreatorID,
	}

	m.markets[market.ID] = market
	return market, nil
}

// Get retrieves a market by ID
func (m *Manager) Get(id string) (*Market, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	market, ok := m.markets[id]
	return market, ok
}

// List returns all markets
func (m *Manager) List() []*Market {
	m.mu.RLock()
	defer m.mu.RUnlock()

	markets := make([]*Market, 0, len(m.markets))
	for _, market := range m.markets {
		markets = append(markets, market)
	}
	return markets
}

// Lock transitions a market to locked status
func (m *Manager) Lock(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	market, ok := m.markets[id]
	if !ok {
		return ErrMarketNotFound
	}
	if market.Status != StatusTrading {
		return ErrInvalidTransition
	}

	market.Status = StatusLocked
	return nil
}
