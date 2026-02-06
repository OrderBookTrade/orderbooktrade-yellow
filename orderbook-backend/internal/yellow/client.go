package yellow

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Client manages the WebSocket connection to Yellow ClearNode
type Client struct {
	mu     sync.RWMutex
	conn   *websocket.Conn
	url    string
	signer *Signer

	sessionID     string
	authenticated bool

	// Pending requests waiting for response
	pending   map[int64]chan *Response
	pendingMu sync.Mutex

	// Callbacks
	onMessage func(*Response)
	onError   func(error)

	// Control
	done   chan struct{}
	closed bool
}

// NewClient creates a new Yellow Network client
func NewClient(url string, signer *Signer) *Client {
	return &Client{
		url:     url,
		signer:  signer,
		pending: make(map[int64]chan *Response),
		done:    make(chan struct{}),
	}
}

// Connect establishes the WebSocket connection
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		return nil // Already connected
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.DialContext(ctx, c.url, nil)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	c.conn = conn
	c.closed = false

	// Start message reader
	go c.readLoop()

	return nil
}

// Authenticate performs the auth flow with the ClearNode
func (c *Client) Authenticate(ctx context.Context) error {
	// Step 1: Send auth request
	authReq, err := NewAuthRequest(c.signer.AddressHex())
	if err != nil {
		return err
	}

	resp, err := c.SendRequest(ctx, authReq)
	if err != nil {
		return fmt.Errorf("auth request failed: %w", err)
	}

	if resp.Error != nil {
		return fmt.Errorf("auth request error: %s", resp.Error.Message)
	}

	var authResult AuthRequestResult
	if err := json.Unmarshal(resp.Result, &authResult); err != nil {
		return fmt.Errorf("failed to parse auth result: %w", err)
	}

	// Step 2: Sign the challenge
	signature, err := c.signer.SignMessageHex([]byte(authResult.Challenge))
	if err != nil {
		return fmt.Errorf("failed to sign challenge: %w", err)
	}

	// Step 3: Verify signature
	verifyReq, err := NewAuthVerify(c.signer.AddressHex(), signature, time.Now().Unix())
	if err != nil {
		return err
	}

	resp, err = c.SendRequest(ctx, verifyReq)
	if err != nil {
		return fmt.Errorf("auth verify failed: %w", err)
	}

	if resp.Error != nil {
		return fmt.Errorf("auth verify error: %s", resp.Error.Message)
	}

	var verifyResult AuthVerifyResult
	if err := json.Unmarshal(resp.Result, &verifyResult); err != nil {
		return fmt.Errorf("failed to parse verify result: %w", err)
	}

	c.mu.Lock()
	c.sessionID = verifyResult.SessionID
	c.authenticated = true
	c.mu.Unlock()

	log.Printf("Authenticated with Yellow Network, session: %s", verifyResult.SessionID)
	return nil
}

// SendRequest sends a JSON-RPC request and waits for response
func (c *Client) SendRequest(ctx context.Context, req *Request) (*Response, error) {
	c.mu.RLock()
	if c.conn == nil {
		c.mu.RUnlock()
		return nil, fmt.Errorf("not connected")
	}
	c.mu.RUnlock()

	// Create response channel
	respChan := make(chan *Response, 1)
	c.pendingMu.Lock()
	c.pending[req.ID] = respChan
	c.pendingMu.Unlock()

	defer func() {
		c.pendingMu.Lock()
		delete(c.pending, req.ID)
		c.pendingMu.Unlock()
	}()

	// Send request
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	err = c.conn.WriteMessage(websocket.TextMessage, data)
	c.mu.Unlock()

	if err != nil {
		return nil, fmt.Errorf("failed to send: %w", err)
	}

	// Wait for response
	select {
	case resp := <-respChan:
		return resp, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("request timeout")
	}
}

// readLoop reads messages from the WebSocket
func (c *Client) readLoop() {
	defer func() {
		c.mu.Lock()
		c.closed = true
		if c.conn != nil {
			c.conn.Close()
		}
		c.mu.Unlock()
	}()

	for {
		select {
		case <-c.done:
			return
		default:
		}

		c.mu.RLock()
		conn := c.conn
		c.mu.RUnlock()

		if conn == nil {
			return
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				if c.onError != nil {
					c.onError(err)
				}
			}
			return
		}

		resp, err := ParseResponse(message)
		if err != nil {
			log.Printf("Failed to parse response: %v", err)
			continue
		}

		// Check if this is a response to a pending request
		c.pendingMu.Lock()
		if ch, ok := c.pending[resp.ID]; ok {
			ch <- resp
			c.pendingMu.Unlock()
			continue
		}
		c.pendingMu.Unlock()

		// Otherwise, it's an unsolicited message (notification)
		if c.onMessage != nil {
			c.onMessage(resp)
		}
	}
}

// SetMessageHandler sets the callback for unsolicited messages
func (c *Client) SetMessageHandler(fn func(*Response)) {
	c.onMessage = fn
}

// SetErrorHandler sets the callback for connection errors
func (c *Client) SetErrorHandler(fn func(error)) {
	c.onError = fn
}

// IsAuthenticated returns whether the client is authenticated
func (c *Client) IsAuthenticated() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.authenticated
}

// Close closes the connection
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	close(c.done)
	c.closed = true

	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// Ping sends a ping request
func (c *Client) Ping(ctx context.Context) error {
	req, err := NewPingRequest()
	if err != nil {
		return err
	}

	resp, err := c.SendRequest(ctx, req)
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return fmt.Errorf("ping error: %s", resp.Error.Message)
	}

	return nil
}
