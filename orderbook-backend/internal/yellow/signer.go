package yellow

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

// Signer handles EIP-712 typed data signing for state channel messages
type Signer struct {
	privateKey *ecdsa.PrivateKey
	address    common.Address
}

// NewSigner creates a signer from a hex-encoded private key
func NewSigner(hexKey string) (*Signer, error) {
	// Remove 0x prefix if present
	if len(hexKey) >= 2 && hexKey[:2] == "0x" {
		hexKey = hexKey[2:]
	}

	keyBytes, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("invalid hex key: %w", err)
	}

	privateKey, err := crypto.ToECDSA(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}

	address := crypto.PubkeyToAddress(privateKey.PublicKey)

	return &Signer{
		privateKey: privateKey,
		address:    address,
	}, nil
}

// Address returns the signer's Ethereum address
func (s *Signer) Address() common.Address {
	return s.address
}

// AddressHex returns the signer's address as a hex string
func (s *Signer) AddressHex() string {
	return s.address.Hex()
}

// SignMessage signs a message with EIP-191 personal sign prefix
func (s *Signer) SignMessage(message []byte) ([]byte, error) {
	hash := accounts.TextHash(message)
	sig, err := crypto.Sign(hash, s.privateKey)
	if err != nil {
		return nil, err
	}

	// Adjust v value for Ethereum (27 or 28)
	if sig[64] < 27 {
		sig[64] += 27
	}

	return sig, nil
}

// SignMessageHex signs a message and returns hex-encoded signature
func (s *Signer) SignMessageHex(message []byte) (string, error) {
	sig, err := s.SignMessage(message)
	if err != nil {
		return "", err
	}
	return "0x" + hex.EncodeToString(sig), nil
}

// SignEIP712Auth signs the Yellow Network auth challenge using EIP-712
func (s *Signer) SignEIP712Auth(
	challenge string,
	params AuthRequestParams,
	domainName string,
) (string, error) {
	// Build EIP-712 TypedData
	typedData := apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": []apitypes.Type{
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
			},
			"AuthVerify": []apitypes.Type{
				{Name: "address", Type: "address"},
				{Name: "session_key", Type: "address"},
				{Name: "challenge_message", Type: "string"},
				{Name: "allowances", Type: "Allowance[]"},
				{Name: "expires_at", Type: "uint256"},
				{Name: "scope", Type: "string"},
				{Name: "application", Type: "string"},
			},
			"Allowance": []apitypes.Type{
				{Name: "asset", Type: "string"},
				{Name: "amount", Type: "string"},
			},
		},
		PrimaryType: "AuthVerify",
		Domain: apitypes.TypedDataDomain{
			Name:    domainName,
			Version: "1",
		},
		Message: apitypes.TypedDataMessage{
			"address":           params.Address,
			"session_key":       params.SessionKey,
			"challenge_message": challenge,
			"allowances":        convertAllowancesToTypedData(params.Allowances),
			"expires_at":        fmt.Sprintf("%d", params.ExpiresAt),
			"scope":             params.Scope,
			"application":       params.Application,
		},
	}

	// Calculate the hash to sign
	domainSeparator, err := typedData.HashStruct("EIP712Domain", typedData.Domain.Map())
	if err != nil {
		return "", fmt.Errorf("failed to hash domain: %w", err)
	}

	typedDataHash, err := typedData.HashStruct(typedData.PrimaryType, typedData.Message)
	if err != nil {
		return "", fmt.Errorf("failed to hash message: %w", err)
	}

	// Final hash: keccak256("\x19\x01" + domainSeparator + typedDataHash)
	rawData := []byte(fmt.Sprintf("\x19\x01%s%s", string(domainSeparator), string(typedDataHash)))
	hash := crypto.Keccak256(rawData)

	// Sign the hash
	sig, err := crypto.Sign(hash, s.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign: %w", err)
	}

	// Adjust v value for Ethereum (27 or 28)
	if sig[64] < 27 {
		sig[64] += 27
	}

	return "0x" + hex.EncodeToString(sig), nil
}

// convertAllowancesToTypedData converts allowances to TypedData format
func convertAllowancesToTypedData(allowances []AuthAllowance) []map[string]interface{} {
	result := make([]map[string]interface{}, len(allowances))
	for i, a := range allowances {
		result[i] = map[string]interface{}{
			"asset":  a.Asset,
			"amount": a.Amount,
		}
	}
	return result
}

// GenerateSessionKey generates a new random session keypair
func GenerateSessionKey() (*ecdsa.PrivateKey, common.Address, error) {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, common.Address{}, fmt.Errorf("failed to generate key: %w", err)
	}

	address := crypto.PubkeyToAddress(privateKey.PublicKey)
	return privateKey, address, nil
}

// SignStateHash signs a state channel state hash (EIP-712 style)
func (s *Signer) SignStateHash(
	channelID [32]byte,
	version uint64,
	allocations []Allocation,
) ([]byte, error) {
	// Build the state hash according to Nitrolite protocol
	stateHash := buildStateHash(channelID, version, allocations)

	sig, err := crypto.Sign(stateHash, s.privateKey)
	if err != nil {
		return nil, err
	}

	if sig[64] < 27 {
		sig[64] += 27
	}

	return sig, nil
}

// SignStateHashHex signs and returns hex-encoded signature
func (s *Signer) SignStateHashHex(
	channelID [32]byte,
	version uint64,
	allocations []Allocation,
) (string, error) {
	sig, err := s.SignStateHash(channelID, version, allocations)
	if err != nil {
		return "", err
	}
	return "0x" + hex.EncodeToString(sig), nil
}

// buildStateHash constructs the hash to sign for a state update
func buildStateHash(channelID [32]byte, version uint64, allocations []Allocation) []byte {
	// Simplified state hash - in production should match Nitrolite's exact format
	// STATE_TYPEHASH = keccak256("AllowStateHash(bytes32 channelId,uint8 intent,uint256 version,bytes data,Allocation[] allocations)Allocation(address destination,address token,uint256 amount)")

	// For now, create a simple hash of the key fields
	data := append(channelID[:], big.NewInt(int64(version)).Bytes()...)

	for _, alloc := range allocations {
		data = append(data, common.HexToAddress(alloc.Participant).Bytes()...)
		data = append(data, common.HexToAddress(alloc.Token).Bytes()...)
		// Parse amount as big.Int
		amount := new(big.Int)
		amount.SetString(alloc.Amount, 10)
		data = append(data, common.LeftPadBytes(amount.Bytes(), 32)...)
	}

	return crypto.Keccak256(data)
}

// VerifySignature verifies a signature against a message and address
func VerifySignature(message []byte, sigHex string, expectedAddr common.Address) (bool, error) {
	if len(sigHex) >= 2 && sigHex[:2] == "0x" {
		sigHex = sigHex[2:]
	}

	sig, err := hex.DecodeString(sigHex)
	if err != nil {
		return false, err
	}

	if len(sig) != 65 {
		return false, fmt.Errorf("invalid signature length: %d", len(sig))
	}

	// Adjust v value back
	if sig[64] >= 27 {
		sig[64] -= 27
	}

	hash := accounts.TextHash(message)
	pubKey, err := crypto.SigToPub(hash, sig)
	if err != nil {
		return false, err
	}

	recoveredAddr := crypto.PubkeyToAddress(*pubKey)
	return recoveredAddr == expectedAddr, nil
}
