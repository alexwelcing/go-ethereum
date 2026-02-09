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
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	charcontract "github.com/ethereum/go-ethereum/contracts/character"
	"github.com/ethereum/go-ethereum/core/types"
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

// Service orchestrates the full text-to-character lifecycle:
//  1. Accept text traits from the user
//  2. Build metadata & compute trait hash
//  3. Store metadata off-chain (IPFS, etc.)
//  4. Mint the NFT on-chain (collecting mint fee)
//  5. Let users advance their character through the pipeline
//  6. Facilitate secondary sales (collecting transaction fee)
//
// The Service does NOT perform media generation — it delegates to
// registered StageProcessors via the Pipeline.
type Service struct {
	nft      *charcontract.CharacterNFT
	pipeline *Pipeline
	store    MetadataStore
	fees     *FeeSchedule
	opts     *bind.TransactOpts

	mu    sync.RWMutex
	cache map[uint64]*CharacterMeta // tokenID → metadata (in-memory cache)
}

// NewService creates a new character service wired to an on-chain contract.
func NewService(
	nft *charcontract.CharacterNFT,
	pipeline *Pipeline,
	store MetadataStore,
	fees *FeeSchedule,
	opts *bind.TransactOpts,
) *Service {
	return &Service{
		nft:      nft,
		pipeline: pipeline,
		store:    store,
		fees:     fees,
		opts:     opts,
		cache:    make(map[uint64]*CharacterMeta),
	}
}

// ──────────────────────────────────────────────
//  Minting
// ──────────────────────────────────────────────

// MintRequest is the input from the consumer to create a new character.
type MintRequest struct {
	Name   string  `json:"name"`
	Traits []Trait `json:"traits"`
}

// MintResult is returned after a successful mint.
type MintResult struct {
	MetadataURI string      `json:"metadata_uri"`
	TraitHash   common.Hash `json:"trait_hash"`
	TxHash      common.Hash `json:"tx_hash"`
}

// Mint creates a new character from text traits, stores metadata, and mints
// the NFT on-chain.  Returns the transaction hash and metadata URI.
func (s *Service) Mint(creator common.Address, req *MintRequest) (*MintResult, error) {
	// 1. Build off-chain metadata
	meta, err := NewCharacterMeta(req.Name, creator, req.Traits)
	if err != nil {
		return nil, fmt.Errorf("character service: %v", err)
	}

	// 2. Store metadata (e.g. to IPFS)
	uri, err := s.store.Put(meta)
	if err != nil {
		return nil, fmt.Errorf("character service: failed to store metadata: %v", err)
	}

	// 3. Mint on-chain — caller must have attached >= mintFee in tx opts
	oldValue := s.opts.Value
	s.opts.Value = s.fees.QuoteMint()
	tx, err := s.nft.Mint(uri, meta.TraitHash)
	s.opts.Value = oldValue
	if err != nil {
		return nil, fmt.Errorf("character service: mint tx failed: %v", err)
	}

	log.Info("Character minted", "name", req.Name, "tx", tx.Hash().Hex(), "uri", uri)

	return &MintResult{
		MetadataURI: uri,
		TraitHash:   meta.TraitHash,
		TxHash:      tx.Hash(),
	}, nil
}

// ──────────────────────────────────────────────
//  Pipeline advancement
// ──────────────────────────────────────────────

// Advance moves a character to the next stage by running the registered
// processor, updating off-chain metadata, and recording the new URI on-chain.
func (s *Service) Advance(tokenID uint64) (*types.Transaction, error) {
	// Fetch current metadata
	meta, err := s.getOrFetchMeta(tokenID)
	if err != nil {
		return nil, err
	}

	// Run the pipeline processor for the next stage
	assetURI, err := s.pipeline.Advance(meta)
	if err != nil {
		return nil, fmt.Errorf("character service: pipeline advance failed: %v", err)
	}
	log.Info("Character stage advanced", "tokenID", tokenID, "stage", meta.Stage, "asset", assetURI)

	// Persist updated metadata
	newURI, err := s.store.Put(meta)
	if err != nil {
		return nil, fmt.Errorf("character service: failed to store updated metadata: %v", err)
	}

	// Record on-chain
	tx, err := s.nft.AdvanceStage(new(big.Int).SetUint64(tokenID), newURI)
	if err != nil {
		return nil, fmt.Errorf("character service: advanceStage tx failed: %v", err)
	}

	// Update cache
	s.mu.Lock()
	s.cache[tokenID] = meta
	s.mu.Unlock()

	return tx, nil
}

// ──────────────────────────────────────────────
//  Secondary sales
// ──────────────────────────────────────────────

// QuoteSale returns the platform cut and seller proceeds for a given price.
func (s *Service) QuoteSale(salePrice *big.Int) (platformCut, sellerProceeds *big.Int, err error) {
	return s.fees.PlatformCut(salePrice)
}

// Transfer facilitates a secondary sale of a character NFT.
func (s *Service) Transfer(tokenID uint64, to common.Address, salePrice *big.Int) (*types.Transaction, error) {
	oldValue := s.opts.Value
	s.opts.Value = salePrice
	tx, err := s.nft.TransferFrom(new(big.Int).SetUint64(tokenID), to)
	s.opts.Value = oldValue
	if err != nil {
		return nil, fmt.Errorf("character service: transfer tx failed: %v", err)
	}

	platformCut, _, _ := s.fees.PlatformCut(salePrice)
	log.Info("Character transferred", "tokenID", tokenID, "to", to.Hex(), "price", salePrice, "platformCut", platformCut)

	// Invalidate cache for this token
	s.mu.Lock()
	delete(s.cache, tokenID)
	s.mu.Unlock()

	return tx, nil
}

// ──────────────────────────────────────────────
//  Reads
// ──────────────────────────────────────────────

// GetCharacter returns the off-chain metadata for a character, fetching
// from the metadata store if not cached.
func (s *Service) GetCharacter(tokenID uint64) (*CharacterMeta, error) {
	return s.getOrFetchMeta(tokenID)
}

// GetFeeSchedule returns the current fee schedule.
func (s *Service) GetFeeSchedule() *FeeSchedule {
	return s.fees
}

// getOrFetchMeta checks cache, then falls back to on-chain → metadata store.
func (s *Service) getOrFetchMeta(tokenID uint64) (*CharacterMeta, error) {
	s.mu.RLock()
	if meta, ok := s.cache[tokenID]; ok {
		s.mu.RUnlock()
		return meta, nil
	}
	s.mu.RUnlock()

	// Read from chain to get metadata URI
	info, err := s.nft.GetCharacter(new(big.Int).SetUint64(tokenID))
	if err != nil {
		return nil, fmt.Errorf("character service: on-chain read failed: %v", err)
	}

	// Fetch from off-chain store
	meta, err := s.store.Get(info.MetadataURI)
	if err != nil {
		return nil, fmt.Errorf("character service: metadata fetch failed for %s: %v", info.MetadataURI, err)
	}
	meta.TokenID = tokenID

	s.mu.Lock()
	s.cache[tokenID] = meta
	s.mu.Unlock()

	return meta, nil
}

// ──────────────────────────────────────────────
//  JSON-RPC API (for node integration)
// ──────────────────────────────────────────────

// API exposes the character service over JSON-RPC when registered with a
// go-ethereum node.  Method namespace: "character".
type API struct {
	service *Service
}

// NewAPI creates a JSON-RPC API backed by the given service.
func NewAPI(service *Service) *API {
	return &API{service: service}
}

// Mint handles "character_mint" RPC calls.
func (api *API) Mint(creator common.Address, reqJSON json.RawMessage) (*MintResult, error) {
	var req MintRequest
	if err := json.Unmarshal(reqJSON, &req); err != nil {
		return nil, fmt.Errorf("invalid mint request: %v", err)
	}
	return api.service.Mint(creator, &req)
}

// GetCharacter handles "character_getCharacter" RPC calls.
func (api *API) GetCharacter(tokenID uint64) (*CharacterMeta, error) {
	return api.service.GetCharacter(tokenID)
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
