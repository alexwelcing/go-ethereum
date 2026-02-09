// Copyright 2018 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package character

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
)

// Solana RPC method constants.
const (
	solMethodGetAccountInfo    = "getAccountInfo"
	solMethodSendTransaction   = "sendTransaction"
	solMethodGetProgramAccounts = "getProgramAccounts"
)

// Errors specific to the Solana backend.
var (
	ErrSolanaNotConfigured = errors.New("solana: program ID or RPC endpoint not configured")
	ErrSolanaKeyNotSet     = errors.New("solana: signer keypair not configured")
	ErrSolanaRPCFailed     = errors.New("solana: RPC call failed")
)

// SolanaConfig holds the configuration needed to connect to a Solana cluster
// and interact with the deployed character_nft program.
type SolanaConfig struct {
	// RPCEndpoint is the Solana JSON-RPC URL (e.g. "https://api.mainnet-beta.solana.com").
	RPCEndpoint string `json:"rpc_endpoint"`

	// ProgramID is the base58-encoded address of the deployed character_nft program.
	ProgramID string `json:"program_id"`

	// StateAccount is the base58-encoded address of the ProgramState account
	// (created during `initialize`).
	StateAccount string `json:"state_account"`

	// PlatformKeypair is the path to the platform wallet keypair JSON file.
	PlatformKeypair string `json:"platform_keypair"`
}

// SolanaBackend implements ChainBackend for the Solana character_nft program.
//
// Transaction construction and signing use the Solana JSON-RPC directly.
// For production, this would use a full Solana Go SDK (e.g. gagliardetto/solana-go),
// but this implementation provides the structural foundation and RPC scaffolding.
type SolanaBackend struct {
	config SolanaConfig
	client *http.Client
	fees   *FeeSchedule
}

// NewSolanaBackend creates a Solana chain backend.
func NewSolanaBackend(config SolanaConfig, fees *FeeSchedule) (*SolanaBackend, error) {
	if config.RPCEndpoint == "" || config.ProgramID == "" {
		return nil, ErrSolanaNotConfigured
	}
	return &SolanaBackend{
		config: config,
		client: &http.Client{},
		fees:   fees,
	}, nil
}

func (s *SolanaBackend) Chain() ChainID { return ChainSolana }

func (s *SolanaBackend) Mint(metadataURI string, traitHash [32]byte) (string, error) {
	// Build the mint instruction data:
	// [8-byte discriminator] [4-byte string len] [string bytes] [32-byte trait hash]
	discriminator := anchorDiscriminator("global", "mint")

	uriBytes := []byte(metadataURI)
	data := make([]byte, 8+4+len(uriBytes)+32)
	copy(data[0:8], discriminator[:])
	binary.LittleEndian.PutUint32(data[8:12], uint32(len(uriBytes)))
	copy(data[12:12+len(uriBytes)], uriBytes)
	copy(data[12+len(uriBytes):], traitHash[:])

	return s.sendInstruction("mint", data)
}

func (s *SolanaBackend) TransferFrom(tokenID uint64, to string, salePrice *big.Int) (string, error) {
	discriminator := anchorDiscriminator("global", "transfer_from")

	price := uint64(0)
	if salePrice != nil {
		price = salePrice.Uint64()
	}

	data := make([]byte, 8+8)
	copy(data[0:8], discriminator[:])
	binary.LittleEndian.PutUint64(data[8:16], price)

	return s.sendInstruction("transfer_from", data)
}

func (s *SolanaBackend) AdvanceStage(tokenID uint64, newMetadataURI string) (string, error) {
	discriminator := anchorDiscriminator("global", "advance_stage")

	uriBytes := []byte(newMetadataURI)
	data := make([]byte, 8+4+len(uriBytes))
	copy(data[0:8], discriminator[:])
	binary.LittleEndian.PutUint32(data[8:12], uint32(len(uriBytes)))
	copy(data[12:], uriBytes)

	return s.sendInstruction("advance_stage", data)
}

func (s *SolanaBackend) GetCharacter(tokenID uint64) (*OnChainCharacter, error) {
	// In production, derive the character PDA from tokenID + programID
	// and call getAccountInfo, then deserialize the account data.
	//
	// Placeholder: returns structured error indicating the account lookup
	// path for the integrator to complete with their Solana SDK of choice.
	return nil, fmt.Errorf("solana: GetCharacter requires PDA derivation for token %d — wire up with solana-go SDK", tokenID)
}

func (s *SolanaBackend) OwnerOf(tokenID uint64) (string, error) {
	char, err := s.GetCharacter(tokenID)
	if err != nil {
		return "", err
	}
	return char.Creator, nil
}

func (s *SolanaBackend) BalanceOf(owner string) (uint64, error) {
	// Requires getProgramAccounts with owner filter
	return 0, fmt.Errorf("solana: BalanceOf requires getProgramAccounts filter — wire up with solana-go SDK")
}

func (s *SolanaBackend) TotalSupply() (uint64, error) {
	// Read from the ProgramState account's next_token_id field
	return 0, fmt.Errorf("solana: TotalSupply requires ProgramState deserialization — wire up with solana-go SDK")
}

func (s *SolanaBackend) MintFee() (*big.Int, error) {
	return new(big.Int).Set(s.fees.MintFee), nil
}

func (s *SolanaBackend) TransactionFeeBps() (*big.Int, error) {
	return new(big.Int).Set(s.fees.TransactionFeeBps), nil
}

func (s *SolanaBackend) PlatformAddress() (string, error) {
	// Read from ProgramState account
	return "", fmt.Errorf("solana: PlatformAddress requires ProgramState deserialization — wire up with solana-go SDK")
}

// ──────────────────────────────────────────────
//  Internal helpers
// ──────────────────────────────────────────────

// solanaRPCRequest is a JSON-RPC 2.0 request body for Solana.
type solanaRPCRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      int           `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

// solanaRPCResponse is a minimal JSON-RPC 2.0 response.
type solanaRPCResponse struct {
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// rpcCall makes a JSON-RPC call to the Solana cluster.
func (s *SolanaBackend) rpcCall(method string, params ...interface{}) (*solanaRPCResponse, error) {
	reqBody := solanaRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  method,
		Params:  params,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Post(s.config.RPCEndpoint, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSolanaRPCFailed, err)
	}
	defer resp.Body.Close()

	var rpcResp solanaRPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return nil, fmt.Errorf("%w: failed to decode response: %v", ErrSolanaRPCFailed, err)
	}
	if rpcResp.Error != nil {
		return nil, fmt.Errorf("%w: code=%d msg=%s", ErrSolanaRPCFailed, rpcResp.Error.Code, rpcResp.Error.Message)
	}
	return &rpcResp, nil
}

// sendInstruction is a placeholder that builds and sends a transaction.
// In production this would construct the full transaction with proper
// account metas, recent blockhash, and signing.
func (s *SolanaBackend) sendInstruction(name string, data []byte) (string, error) {
	// This is the integration point where a full Solana Go SDK (e.g.
	// gagliardetto/solana-go) would:
	// 1. Fetch recent blockhash via getLatestBlockhash
	// 2. Build the transaction with the instruction data + account metas
	// 3. Sign with the platform keypair
	// 4. Call sendTransaction
	//
	// For now, return an actionable error so integrators know exactly
	// what to wire up.
	return "", fmt.Errorf("solana: %s instruction built (%d bytes) — requires solana-go SDK for signing and submission", name, len(data))
}

// anchorDiscriminator computes the 8-byte Anchor instruction discriminator:
// sha256("namespace:name")[:8].
func anchorDiscriminator(namespace, name string) [8]byte {
	hash := sha256.Sum256([]byte(namespace + ":" + name))
	var disc [8]byte
	copy(disc[:], hash[:8])
	return disc
}
