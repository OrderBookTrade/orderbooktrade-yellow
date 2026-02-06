package market

import "errors"

var (
	ErrMarketNotFound    = errors.New("market not found")
	ErrInvalidTransition = errors.New("invalid market status transition")
	ErrMarketNotLocked   = errors.New("market must be locked before resolution")
	ErrAlreadyResolved   = errors.New("market already resolved")
	ErrInvalidOutcome    = errors.New("outcome must be YES or NO")
)
