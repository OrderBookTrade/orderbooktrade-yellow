package market

import (
	"time"
)

// ResolveRequest is the request to resolve a market
type ResolveRequest struct {
	MarketID string  `json:"market_id"`
	Outcome  Outcome `json:"outcome"` // YES or NO
}

// Payout represents the payout for a user after resolution
type Payout struct {
	UserID    string `json:"user_id"`
	MarketID  string `json:"market_id"`
	Shares    uint64 `json:"shares"`     // Number of winning shares
	AmountUSD uint64 `json:"amount_usd"` // Payout in USDC (6 decimals)
}

// Resolve resolves a market with the given outcome
func (m *Manager) Resolve(req ResolveRequest) (*Market, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	market, ok := m.markets[req.MarketID]
	if !ok {
		return nil, ErrMarketNotFound
	}

	if market.Status != StatusLocked {
		return nil, ErrMarketNotLocked
	}

	if market.Outcome != nil {
		return nil, ErrAlreadyResolved
	}

	if req.Outcome != OutcomeYes && req.Outcome != OutcomeNo {
		return nil, ErrInvalidOutcome
	}

	now := time.Now()
	market.Outcome = &req.Outcome
	market.ResolvedAt = &now
	market.Status = StatusResolved

	return market, nil
}

// CalculatePayouts calculates payouts for all users with positions in a resolved market
// positions: map[userID]Position where Position has YesShares and NoShares
func CalculatePayouts(market *Market, positions map[string]*Position) ([]Payout, error) {
	if market.Status != StatusResolved || market.Outcome == nil {
		return nil, ErrMarketNotLocked
	}

	var payouts []Payout

	for userID, pos := range positions {
		var winningShares uint64

		if *market.Outcome == OutcomeYes {
			winningShares = pos.YesShares
		} else {
			winningShares = pos.NoShares
		}

		if winningShares > 0 {
			// Each winning share pays out 1 USDC (1_000_000 in 6 decimal representation)
			// But we use basis points internally: 10000 = 1 USDC
			payout := Payout{
				UserID:    userID,
				MarketID:  market.ID,
				Shares:    winningShares,
				AmountUSD: winningShares * 10000, // 10000 basis points = 1 USDC
			}
			payouts = append(payouts, payout)
		}
	}

	return payouts, nil
}

// Position tracks a user's share holdings in a market
type Position struct {
	UserID    string `json:"user_id"`
	MarketID  string `json:"market_id"`
	YesShares uint64 `json:"yes_shares"`
	NoShares  uint64 `json:"no_shares"`
}
