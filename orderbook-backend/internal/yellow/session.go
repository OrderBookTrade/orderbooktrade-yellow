package yellow

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// Session manages an app session lifecycle with Yellow Network
type Session struct {
	mu          sync.RWMutex
	client      *Client
	channelID   string
	version     uint64
	allocations []Allocation
	active      bool
}

// SessionManager manages multiple sessions
type SessionManager struct {
	mu       sync.RWMutex
	client   *Client
	sessions map[string]*Session
}

// NewSessionManager creates a new session manager
func NewSessionManager(client *Client) *SessionManager {
	return &SessionManager{
		client:   client,
		sessions: make(map[string]*Session),
	}
}

// CreateSession creates a new app session
func (m *SessionManager) CreateSession(
	ctx context.Context,
	participants []string,
	allocations []Allocation,
	adjudicatorAddr string,
) (*Session, error) {
	if !m.client.IsAuthenticated() {
		return nil, fmt.Errorf("client not authenticated")
	}

	// Build app definition
	weights := make([]int, len(participants))
	for i := range weights {
		weights[i] = 1
	}

	def := AppDefinition{
		Protocol:     "orderbook",
		Participants: participants,
		Weights:      weights,
		Quorum:       len(participants),
		Challenge:    3600, // 1 hour challenge period
		Nonce:        generateNonce(),
	}

	req, err := NewCreateAppSession(def, allocations)
	if err != nil {
		return nil, err
	}

	resp, err := m.client.SendRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("create session failed: %w", err)
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("create session error: %s", resp.Error.Message)
	}

	var result CreateAppSessionResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse result: %w", err)
	}

	session := &Session{
		client:      m.client,
		channelID:   result.ChannelID,
		version:     0,
		allocations: allocations,
		active:      true,
	}

	m.mu.Lock()
	m.sessions[result.ChannelID] = session
	m.mu.Unlock()

	return session, nil
}

// GetSession returns a session by channel ID
func (m *SessionManager) GetSession(channelID string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	session, ok := m.sessions[channelID]
	return session, ok
}

// CloseSession closes an app session
func (m *SessionManager) CloseSession(ctx context.Context, channelID string) error {
	m.mu.Lock()
	session, ok := m.sessions[channelID]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("session not found: %s", channelID)
	}
	delete(m.sessions, channelID)
	m.mu.Unlock()

	return session.Close(ctx)
}

// UpdateState updates the session state with new allocations
func (s *Session) UpdateState(ctx context.Context, allocations []Allocation, appData string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.active {
		return fmt.Errorf("session is not active")
	}

	s.version++

	state := StateUpdate{
		Version:     s.version,
		Allocations: allocations,
		AppData:     appData,
	}

	// Sign the state (simplified - in production, need proper EIP-712)
	// For now, we'll sign a simple hash
	sig := "" // TODO: Implement proper signing

	req, err := NewAppSessionMessage(s.channelID, state, sig)
	if err != nil {
		s.version-- // Rollback
		return err
	}

	resp, err := s.client.SendRequest(ctx, req)
	if err != nil {
		s.version--
		return fmt.Errorf("update state failed: %w", err)
	}

	if resp.Error != nil {
		s.version--
		return fmt.Errorf("update state error: %s", resp.Error.Message)
	}

	s.allocations = allocations
	return nil
}

// Close closes the session
func (s *Session) Close(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.active {
		return nil
	}

	req, err := NewCloseAppSession(s.channelID, s.allocations)
	if err != nil {
		return err
	}

	resp, err := s.client.SendRequest(ctx, req)
	if err != nil {
		return fmt.Errorf("close session failed: %w", err)
	}

	if resp.Error != nil {
		return fmt.Errorf("close session error: %s", resp.Error.Message)
	}

	s.active = false
	return nil
}

// GetChannelID returns the session's channel ID
func (s *Session) GetChannelID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.channelID
}

// GetAllocations returns the current allocations
func (s *Session) GetAllocations() []Allocation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Allocation, len(s.allocations))
	copy(result, s.allocations)
	return result
}

// IsActive returns whether the session is active
func (s *Session) IsActive() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.active
}

// generateNonce generates a unique nonce for session creation
func generateNonce() int64 {
	return nonce()
}

var nonceCounter int64

func nonce() int64 {
	nonceCounter++
	return nonceCounter
}
