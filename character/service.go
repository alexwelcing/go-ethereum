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
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/log"
)

// MetadataStore is the interface for persisting character metadata off-chain.
// Implementations could target IPFS, S3, a local database, etc.
type MetadataStore interface {
	// Put stores metadata and returns its content-addressed URI.
	Put(meta *CharacterMeta) (uri string, err error)

	// Get retrieves metadata by its URI.
	Get(uri string) (*CharacterMeta, error)
}

// Errors for the service layer.
var (
	ErrChainNotRegistered = errors.New("character service: requested chain has no registered backend")
)

// Service orchestrates the full text-to-character lifecycle across multiple
// chains.  It is the single entry point for the platform:
//  1. Accept text traits from the user
//  2. Build metadata & compute trait hash
//  3. Store metadata off-chain (IPFS, etc.)
//  4. Mint the NFT on whichever chain the user picks (collecting mint fee)
//  5. Let users advance their character through the pipeline
//  6. Facilitate secondary sales (collecting transaction fee)
//
// The Service does NOT perform media generation — it delegates to
// registered StageProcessors via the Pipeline.
type Service struct {
	chains   map[ChainID]ChainBackend
	pipeline *Pipeline
	store    MetadataStore
	fees     *FeeSchedule

	mu    sync.RWMutex
	cache map[cacheKey]*CharacterMeta
}

// cacheKey uniquely identifies a character across chains.
type cacheKey struct {
	chain   ChainID
	tokenID uint64
}

// NewService creates a new multi-chain character service.
func NewService(
	pipeline *Pipeline,
	store MetadataStore,
	fees *FeeSchedule,
) *Service {
	return &Service{
		chains:   make(map[ChainID]ChainBackend),
		pipeline: pipeline,
		store:    store,
		fees:     fees,
		cache:    make(map[cacheKey]*CharacterMeta),
	}
}

// RegisterChain adds a chain backend.  Call this for each chain you want
// to support (e.g. Ethereum, Solana).
func (s *Service) RegisterChain(backend ChainBackend) {
	s.chains[backend.Chain()] = backend
}

// backend returns the registered backend for a chain, or an error.
func (s *Service) backend(chain ChainID) (ChainBackend, error) {
	b, ok := s.chains[chain]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrChainNotRegistered, chain)
	}
	return b, nil
}

// ──────────────────────────────────────────────
//  Minting
// ──────────────────────────────────────────────

// MintRequest is the input from the consumer to create a new character.
type MintRequest struct {
	Name   string  `json:"name"`
	Traits []Trait `json:"traits"`
	Chain  ChainID `json:"chain"` // which chain to mint on
}

// MintResult is returned after a successful mint.
type MintResult struct {
	MetadataURI string   `json:"metadata_uri"`
	TraitHash   [32]byte `json:"trait_hash"`
	TxHash      string   `json:"tx_hash"`
	Chain       ChainID  `json:"chain"`
}

// Mint creates a new character from text traits, stores metadata, and mints
// the NFT on the requested chain.
func (s *Service) Mint(creator string, req *MintRequest) (*MintResult, error) {
	b, err := s.backend(req.Chain)
	if err != nil {
		return nil, err
	}

	// 1. Build off-chain metadata
	meta, err := NewCharacterMeta(req.Name, creator, req.Chain, req.Traits)
	if err != nil {
		return nil, fmt.Errorf("character service: %v", err)
	}

	// 2. Store metadata (e.g. to IPFS)
	uri, err := s.store.Put(meta)
	if err != nil {
		return nil, fmt.Errorf("character service: failed to store metadata: %v", err)
	}

	// 3. Mint on-chain
	txHash, err := b.Mint(uri, meta.TraitHash)
	if err != nil {
		return nil, fmt.Errorf("character service: mint tx failed on %s: %v", req.Chain, err)
	}

	log.Info("Character minted", "name", req.Name, "chain", req.Chain, "tx", txHash, "uri", uri)

	return &MintResult{
		MetadataURI: uri,
		TraitHash:   meta.TraitHash,
		TxHash:      txHash,
		Chain:       req.Chain,
	}, nil
}

// ──────────────────────────────────────────────
//  Pipeline advancement
// ──────────────────────────────────────────────

// Advance moves a character to the next stage by running the registered
// processor, updating off-chain metadata, and recording the new URI on-chain.
func (s *Service) Advance(chain ChainID, tokenID uint64) (string, error) {
	b, err := s.backend(chain)
	if err != nil {
		return "", err
	}

	meta, err := s.getOrFetchMeta(chain, tokenID)
	if err != nil {
		return "", err
	}

	// Run the pipeline processor for the next stage
	assetURI, err := s.pipeline.Advance(meta)
	if err != nil {
		return "", fmt.Errorf("character service: pipeline advance failed: %v", err)
	}
	log.Info("Character stage advanced", "chain", chain, "tokenID", tokenID, "stage", meta.Stage, "asset", assetURI)

	// Persist updated metadata
	newURI, err := s.store.Put(meta)
	if err != nil {
		return "", fmt.Errorf("character service: failed to store updated metadata: %v", err)
	}

	// Record on-chain
	txHash, err := b.AdvanceStage(tokenID, newURI)
	if err != nil {
		return "", fmt.Errorf("character service: advanceStage tx failed on %s: %v", chain, err)
	}

	s.mu.Lock()
	s.cache[cacheKey{chain, tokenID}] = meta
	s.mu.Unlock()

	return txHash, nil
}

// ──────────────────────────────────────────────
//  Secondary sales
// ──────────────────────────────────────────────

// QuoteSale returns the platform cut and seller proceeds for a given price.
func (s *Service) QuoteSale(salePrice *big.Int) (platformCut, sellerProceeds *big.Int, err error) {
	return s.fees.PlatformCut(salePrice)
}

// Transfer facilitates a secondary sale of a character NFT on any chain.
func (s *Service) Transfer(chain ChainID, tokenID uint64, to string, salePrice *big.Int) (string, error) {
	b, err := s.backend(chain)
	if err != nil {
		return "", err
	}

	txHash, err := b.TransferFrom(tokenID, to, salePrice)
	if err != nil {
		return "", fmt.Errorf("character service: transfer tx failed on %s: %v", chain, err)
	}

	platformCut, _, _ := s.fees.PlatformCut(salePrice)
	log.Info("Character transferred", "chain", chain, "tokenID", tokenID, "to", to, "price", salePrice, "platformCut", platformCut)

	s.mu.Lock()
	delete(s.cache, cacheKey{chain, tokenID})
	s.mu.Unlock()

	return txHash, nil
}

// ──────────────────────────────────────────────
//  Reads
// ──────────────────────────────────────────────

// GetCharacter returns the off-chain metadata for a character.
func (s *Service) GetCharacter(chain ChainID, tokenID uint64) (*CharacterMeta, error) {
	return s.getOrFetchMeta(chain, tokenID)
}

// GetFeeSchedule returns the current fee schedule.
func (s *Service) GetFeeSchedule() *FeeSchedule {
	return s.fees
}

// SupportedChains returns the list of registered chain backends.
func (s *Service) SupportedChains() []ChainID {
	chains := make([]ChainID, 0, len(s.chains))
	for id := range s.chains {
		chains = append(chains, id)
	}
	return chains
}

// getOrFetchMeta checks cache, then falls back to on-chain → metadata store.
func (s *Service) getOrFetchMeta(chain ChainID, tokenID uint64) (*CharacterMeta, error) {
	key := cacheKey{chain, tokenID}

	s.mu.RLock()
	if meta, ok := s.cache[key]; ok {
		s.mu.RUnlock()
		return meta, nil
	}
	s.mu.RUnlock()

	b, err := s.backend(chain)
	if err != nil {
		return nil, err
	}

	info, err := b.GetCharacter(tokenID)
	if err != nil {
		return nil, fmt.Errorf("character service: on-chain read failed on %s: %v", chain, err)
	}

	meta, err := s.store.Get(info.MetadataURI)
	if err != nil {
		return nil, fmt.Errorf("character service: metadata fetch failed for %s: %v", info.MetadataURI, err)
	}
	meta.TokenID = tokenID
	meta.Chain = chain

	s.mu.Lock()
	s.cache[key] = meta
	s.mu.Unlock()

	return meta, nil
}

// ──────────────────────────────────────────────
//  JSON-RPC API (for node integration)
// ──────────────────────────────────────────────

// API exposes the character service over JSON-RPC.
// Method namespace: "character".
type API struct {
	service *Service
}

// NewAPI creates a JSON-RPC API backed by the given service.
func NewAPI(service *Service) *API {
	return &API{service: service}
}

// Mint handles "character_mint" RPC calls.
func (api *API) Mint(creator string, reqJSON json.RawMessage) (*MintResult, error) {
	var req MintRequest
	if err := json.Unmarshal(reqJSON, &req); err != nil {
		return nil, fmt.Errorf("invalid mint request: %v", err)
	}
	return api.service.Mint(creator, &req)
}

// GetCharacter handles "character_getCharacter" RPC calls.
func (api *API) GetCharacter(chain string, tokenID uint64) (*CharacterMeta, error) {
	return api.service.GetCharacter(ChainID(chain), tokenID)
}

// QuoteMint handles "character_quoteMint" RPC calls.
func (api *API) QuoteMint() string {
	return api.service.GetFeeSchedule().QuoteMint().String()
}

// QuoteSale handles "character_quoteSale" RPC calls.
func (api *API) QuoteSale(salePriceWei string) (map[string]string, error) {
	price, ok := new(big.Int).SetString(salePriceWei, 10)
	if !ok {
		return nil, fmt.Errorf("invalid price: %s", salePriceWei)
	}
	cut, proceeds, err := api.service.QuoteSale(price)
	if err != nil {
		return nil, err
	}
	return map[string]string{
		"platformCut":    cut.String(),
		"sellerProceeds": proceeds.String(),
	}, nil
}

// SupportedChains handles "character_supportedChains" RPC calls.
func (api *API) SupportedChains() []ChainID {
	return api.service.SupportedChains()
}
