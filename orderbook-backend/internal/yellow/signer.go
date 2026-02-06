package yellow

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
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
