package yellow

import (
	"encoding/json"
)

// JSON-RPC 2.0 request/response structures for ERC-7824

// Request is a JSON-RPC 2.0 request
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response is a JSON-RPC 2.0 response
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// RPCError represents a JSON-RPC error
type RPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// --- Method-specific params and results ---

// PingParams for the ping method
type PingParams struct{}

// PingResult for the ping response
type PingResult struct {
	Pong string `json:"pong"`
}

// AuthAllowance represents an asset allowance for the session
type AuthAllowance struct {
	Asset  string `json:"asset"`
	Amount string `json:"amount"`
}

// AuthRequestParams for authentication (EIP-712 compatible)
type AuthRequestParams struct {
	Address     string          `json:"address"`
	SessionKey  string          `json:"session_key"`
	Allowances  []AuthAllowance `json:"allowances"`
	ExpiresAt   int64           `json:"expires_at"`
	Scope       string          `json:"scope"`
	Application string          `json:"application"`
}

// AuthRequestResult with the challenge to sign
type AuthRequestResult struct {
	ChallengeMessage string `json:"challenge_message"`
}

// AuthVerifyParams to verify the signed challenge (EIP-712 signature)
type AuthVerifyParams struct {
	Address          string          `json:"address"`
	SessionKey       string          `json:"session_key"`
	Signature        string          `json:"signature"`
	ChallengeMessage string          `json:"challenge_message"`
	Allowances       []AuthAllowance `json:"allowances"`
	ExpiresAt        int64           `json:"expires_at"`
	Scope            string          `json:"scope"`
	Application      string          `json:"application"`
}

// AuthVerifyResult on successful auth
type AuthVerifyResult struct {
	SessionKey string `json:"session_key"`
	JWTToken   string `json:"jwt_token,omitempty"`
	ExpiresAt  int64  `json:"expires_at"`
}

// CreateAppSessionParams for creating an app session
type CreateAppSessionParams struct {
	Definition  AppDefinition `json:"definition"`
	Allocations []Allocation  `json:"allocations"`
}

// AppDefinition defines the app session configuration
type AppDefinition struct {
	Protocol     string   `json:"protocol"`
	Participants []string `json:"participants"`
	Weights      []int    `json:"weights"`
	Quorum       int      `json:"quorum"`
	Challenge    int64    `json:"challenge"`
	Nonce        int64    `json:"nonce"`
	AppData      string   `json:"app_data,omitempty"`
}

// Allocation represents a participant's fund allocation
type Allocation struct {
	Participant string `json:"participant"`
	Token       string `json:"token"`
	Amount      string `json:"amount"`
}

// CreateAppSessionResult on successful session creation
type CreateAppSessionResult struct {
	ChannelID string `json:"channel_id"`
	Status    string `json:"status"`
}

// CloseAppSessionParams for closing a session
type CloseAppSessionParams struct {
	ChannelID   string       `json:"channel_id"`
	Allocations []Allocation `json:"allocations"`
}

// CloseAppSessionResult on successful close
type CloseAppSessionResult struct {
	ChannelID string `json:"channel_id"`
	Status    string `json:"status"`
}

// AppSessionMessageParams for sending state updates
type AppSessionMessageParams struct {
	ChannelID string      `json:"channel_id"`
	StateData StateUpdate `json:"state_data"`
	Signature string      `json:"signature"`
}

// StateUpdate represents a state channel state update
type StateUpdate struct {
	Version     uint64       `json:"version"`
	Allocations []Allocation `json:"allocations"`
	AppData     string       `json:"app_data"`
}

// --- Message builders ---

var requestID int64

// NewRequest creates a new JSON-RPC request
func NewRequest(method string, params interface{}) (*Request, error) {
	requestID++

	paramsBytes, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	return &Request{
		JSONRPC: "2.0",
		ID:      requestID,
		Method:  method,
		Params:  paramsBytes,
	}, nil
}

// NewPingRequest creates a ping request
func NewPingRequest() (*Request, error) {
	return NewRequest("ping", PingParams{})
}

// NewAuthRequest creates an auth request with full EIP-712 parameters
func NewAuthRequest(params AuthRequestParams) (*Request, error) {
	return NewRequest("auth_request", params)
}

// NewAuthVerify creates an auth verify request with EIP-712 signature
func NewAuthVerify(params AuthVerifyParams) (*Request, error) {
	return NewRequest("auth_verify", params)
}

// NewCreateAppSession creates an app session request
func NewCreateAppSession(def AppDefinition, allocs []Allocation) (*Request, error) {
	return NewRequest("create_app_session", CreateAppSessionParams{
		Definition:  def,
		Allocations: allocs,
	})
}

// NewCloseAppSession creates a close session request
func NewCloseAppSession(channelID string, allocs []Allocation) (*Request, error) {
	return NewRequest("close_app_session", CloseAppSessionParams{
		ChannelID:   channelID,
		Allocations: allocs,
	})
}

// NewAppSessionMessage creates a state update message
func NewAppSessionMessage(channelID string, state StateUpdate, sig string) (*Request, error) {
	return NewRequest("app_session_message", AppSessionMessageParams{
		ChannelID: channelID,
		StateData: state,
		Signature: sig,
	})
}

// ParseResponse parses a JSON-RPC response
func ParseResponse(data []byte) (*Response, error) {
	var resp Response
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
