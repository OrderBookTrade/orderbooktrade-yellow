package engine

import (
	"errors"
	"sync"
)

var (
	ErrInsufficientBalance  = errors.New("insufficient USDC balance")
	ErrInsufficientPosition = errors.New("insufficient shares to sell")
)

// Position tracks a user's share holdings in a specific market
type Position struct {
	UserID    string `json:"user_id"`
	MarketID  string `json:"market_id"`
	YesShares uint64 `json:"yes_shares"`
	NoShares  uint64 `json:"no_shares"`
	Balance   uint64 `json:"balance"` // USDC balance in basis points (10000 = 1 USDC)
}

// PositionManager tracks all user positions
type PositionManager struct {
	mu        sync.RWMutex
	positions map[string]map[string]*Position // userID -> marketID -> Position
	balances  map[string]uint64               // userID -> USDC balance
}

// NewPositionManager creates a new position manager
func NewPositionManager() *PositionManager {
	return &PositionManager{
		positions: make(map[string]map[string]*Position),
		balances:  make(map[string]uint64),
	}
}

// Deposit adds USDC to a user's balance
func (pm *PositionManager) Deposit(userID string, amount uint64) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.balances[userID] += amount
}

// GetBalance returns a user's USDC balance
func (pm *PositionManager) GetBalance(userID string) uint64 {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.balances[userID]
}

// GetPosition returns a user's position in a specific market
func (pm *PositionManager) GetPosition(userID, marketID string) *Position {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	userPositions, ok := pm.positions[userID]
	if !ok {
		return &Position{UserID: userID, MarketID: marketID}
	}
	pos, ok := userPositions[marketID]
	if !ok {
		return &Position{UserID: userID, MarketID: marketID}
	}
	return pos
}

// getOrCreatePosition gets or creates a position (must hold lock)
func (pm *PositionManager) getOrCreatePosition(userID, marketID string) *Position {
	if _, ok := pm.positions[userID]; !ok {
		pm.positions[userID] = make(map[string]*Position)
	}
	if _, ok := pm.positions[userID][marketID]; !ok {
		pm.positions[userID][marketID] = &Position{
			UserID:   userID,
			MarketID: marketID,
		}
	}
	return pm.positions[userID][marketID]
}

// ValidateOrder checks if a user can place an order (has sufficient balance/shares)
func (pm *PositionManager) ValidateOrder(order *Order) error {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if order.Side == SideBuy {
		// Buy: need USDC = price * quantity
		cost := order.Price * order.Quantity / 10000 // Convert from basis points
		if pm.balances[order.UserID] < cost*10000 {  // Compare in basis points
			return ErrInsufficientBalance
		}
	} else {
		// Sell: need shares
		pos := pm.GetPosition(order.UserID, order.MarketID)
		var available uint64
		if order.OutcomeID == OutcomeYes {
			available = pos.YesShares
		} else {
			available = pos.NoShares
		}
		if available < order.Quantity {
			return ErrInsufficientPosition
		}
	}

	return nil
}

// ExecuteTrade updates positions after a trade is executed
// buyer pays USDC, receives shares
// seller pays shares, receives USDC
func (pm *PositionManager) ExecuteTrade(trade *Trade) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Trade has buyerID, sellerID, price, quantity
	// The order that matched determines which outcome was traded

	buyerPos := pm.getOrCreatePosition(trade.BuyerID, trade.MarketID)
	sellerPos := pm.getOrCreatePosition(trade.SellerID, trade.MarketID)

	// Cost = price * quantity (in basis points)
	cost := trade.Price * trade.Quantity

	// Buyer pays USDC
	pm.balances[trade.BuyerID] -= cost
	// Seller receives USDC
	pm.balances[trade.SellerID] += cost

	// Transfer shares based on outcome
	if trade.OutcomeID == OutcomeYes {
		buyerPos.YesShares += trade.Quantity
		sellerPos.YesShares -= trade.Quantity
	} else {
		buyerPos.NoShares += trade.Quantity
		sellerPos.NoShares -= trade.Quantity
	}
}

// MintShares mints new shares for a market (used when user deposits for first time)
// In prediction markets, you often mint 1 YES + 1 NO for 1 USDC
func (pm *PositionManager) MintShares(userID, marketID string, amount uint64) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Cost to mint = amount USDC (10000 basis points = 1 USDC)
	cost := amount * 10000
	if pm.balances[userID] < cost {
		return ErrInsufficientBalance
	}

	pos := pm.getOrCreatePosition(userID, marketID)

	// Deduct USDC
	pm.balances[userID] -= cost

	// Mint equal YES and NO shares
	pos.YesShares += amount
	pos.NoShares += amount

	return nil
}

// RedeemShares redeems YES+NO pairs back to USDC
func (pm *PositionManager) RedeemShares(userID, marketID string, amount uint64) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pos := pm.getOrCreatePosition(userID, marketID)

	if pos.YesShares < amount || pos.NoShares < amount {
		return ErrInsufficientPosition
	}

	// Burn shares
	pos.YesShares -= amount
	pos.NoShares -= amount

	// Credit USDC (1 pair = 1 USDC = 10000 basis points)
	pm.balances[userID] += amount * 10000

	return nil
}

// PayoutWinningShares pays out winning shares after market resolution
func (pm *PositionManager) PayoutWinningShares(userID, marketID string, winningOutcome OutcomeID) uint64 {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pos := pm.getOrCreatePosition(userID, marketID)

	var payout uint64
	if winningOutcome == OutcomeYes {
		payout = pos.YesShares * 10000 // Each share = 1 USDC = 10000 basis points
		pos.YesShares = 0
		pos.NoShares = 0 // Losing shares become worthless
	} else {
		payout = pos.NoShares * 10000
		pos.NoShares = 0
		pos.YesShares = 0
	}

	pm.balances[userID] += payout
	return payout
}

// GetAllPositions returns all positions for a market
func (pm *PositionManager) GetAllPositions(marketID string) []*Position {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var positions []*Position
	for _, userPositions := range pm.positions {
		if pos, ok := userPositions[marketID]; ok {
			if pos.YesShares > 0 || pos.NoShares > 0 {
				positions = append(positions, pos)
			}
		}
	}
	return positions
}
