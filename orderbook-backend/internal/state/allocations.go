package state

import (
	"encoding/json"
	"sync"

	"orderbook-backend/internal/yellow"
)

// Allocations tracks the fund allocations within a state channel
type Allocations struct {
	mu        sync.RWMutex
	channelID string
	token     string
	balances  map[string]uint64 // participant address -> balance
	version   uint64
}

// NewAllocations creates a new allocations tracker
func NewAllocations(channelID string, token string, initial map[string]uint64) *Allocations {
	balances := make(map[string]uint64)
	for k, v := range initial {
		balances[k] = v
	}
	return &Allocations{
		channelID: channelID,
		token:     token,
		balances:  balances,
		version:   0,
	}
}

// GetBalance returns the balance for a participant
func (a *Allocations) GetBalance(participant string) uint64 {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.balances[participant]
}

// GetBalances returns all balances
func (a *Allocations) GetBalances() map[string]uint64 {
	a.mu.RLock()
	defer a.mu.RUnlock()
	result := make(map[string]uint64)
	for k, v := range a.balances {
		result[k] = v
	}
	return result
}

// Transfer moves funds from one participant to another
func (a *Allocations) Transfer(from, to string, amount uint64) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.balances[from] < amount {
		return ErrInsufficientBalance
	}

	a.balances[from] -= amount
	a.balances[to] += amount
	a.version++

	return nil
}

// ApplyTrade updates allocations based on a trade
// buyer pays seller `price * quantity`
func (a *Allocations) ApplyTrade(buyerAddr, sellerAddr string, price, quantity uint64) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Calculate total cost (price is in basis points, quantity is units)
	// cost = price * quantity / 10000 (if using basis points for 0-1 range)
	cost := (price * quantity) / 10000

	if a.balances[buyerAddr] < cost {
		return ErrInsufficientBalance
	}

	a.balances[buyerAddr] -= cost
	a.balances[sellerAddr] += cost
	a.version++

	return nil
}

// ToYellowAllocations converts to Yellow Network allocation format
func (a *Allocations) ToYellowAllocations() []yellow.Allocation {
	a.mu.RLock()
	defer a.mu.RUnlock()

	allocs := make([]yellow.Allocation, 0, len(a.balances))
	for participant, amount := range a.balances {
		allocs = append(allocs, yellow.Allocation{
			Participant: participant,
			Token:       a.token,
			Amount:      formatAmount(amount),
		})
	}
	return allocs
}

// GetVersion returns the current version
func (a *Allocations) GetVersion() uint64 {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.version
}

// Snapshot returns a JSON-serializable snapshot of the allocations
type AllocationSnapshot struct {
	ChannelID string            `json:"channel_id"`
	Token     string            `json:"token"`
	Balances  map[string]uint64 `json:"balances"`
	Version   uint64            `json:"version"`
}

func (a *Allocations) Snapshot() AllocationSnapshot {
	a.mu.RLock()
	defer a.mu.RUnlock()

	balances := make(map[string]uint64)
	for k, v := range a.balances {
		balances[k] = v
	}

	return AllocationSnapshot{
		ChannelID: a.channelID,
		Token:     a.token,
		Balances:  balances,
		Version:   a.version,
	}
}

// ToJSON returns the snapshot as JSON
func (a *Allocations) ToJSON() ([]byte, error) {
	return json.Marshal(a.Snapshot())
}

func formatAmount(amount uint64) string {
	return json.Number(string(rune(amount))).String()
}

// Errors
type AllocationError string

func (e AllocationError) Error() string {
	return string(e)
}

const (
	ErrInsufficientBalance AllocationError = "insufficient balance"
)
