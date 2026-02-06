package market

import (
	"context"
	"log"
	"sync"
	"time"
)

// LifecycleManager handles automatic market status transitions
type LifecycleManager struct {
	marketManager *Manager
	stopCh        chan struct{}
	wg            sync.WaitGroup
}

// NewLifecycleManager creates a new lifecycle manager
func NewLifecycleManager(mm *Manager) *LifecycleManager {
	return &LifecycleManager{
		marketManager: mm,
		stopCh:        make(chan struct{}),
	}
}

// Start begins the lifecycle management goroutine
func (lm *LifecycleManager) Start(ctx context.Context) {
	lm.wg.Add(1)
	go lm.run(ctx)
}

// Stop stops the lifecycle manager
func (lm *LifecycleManager) Stop() {
	close(lm.stopCh)
	lm.wg.Wait()
}

// run is the main loop that checks for markets to lock
func (lm *LifecycleManager) run(ctx context.Context) {
	defer lm.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-lm.stopCh:
			return
		case <-ticker.C:
			lm.checkAndLockMarkets()
		}
	}
}

// checkAndLockMarkets locks any markets that have passed their resolution time
func (lm *LifecycleManager) checkAndLockMarkets() {
	now := time.Now()
	markets := lm.marketManager.List()

	for _, market := range markets {
		if market.Status == StatusTrading && now.After(market.ResolvesAt) {
			if err := lm.marketManager.Lock(market.ID); err != nil {
				log.Printf("Failed to lock market %s: %v", market.ID, err)
			} else {
				log.Printf("Market %s auto-locked (resolution time passed)", market.ID)
			}
		}
	}
}

// ForceTransition allows manual status transition (for admin/testing)
func (lm *LifecycleManager) ForceTransition(marketID string, targetStatus MarketStatus) error {
	lm.marketManager.mu.Lock()
	defer lm.marketManager.mu.Unlock()

	market, ok := lm.marketManager.markets[marketID]
	if !ok {
		return ErrMarketNotFound
	}

	// Validate transition
	switch targetStatus {
	case StatusLocked:
		if market.Status != StatusTrading {
			return ErrInvalidTransition
		}
	case StatusResolved:
		if market.Status != StatusLocked {
			return ErrMarketNotLocked
		}
	default:
		return ErrInvalidTransition
	}

	market.Status = targetStatus
	return nil
}
