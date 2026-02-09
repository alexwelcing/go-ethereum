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

// Package character provides high-level Go bindings for the CharacterNFT contract.
// It facilitates minting text-to-character NFTs and collecting platform fees
// on mints and secondary-market transactions.
package character

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/character/contract"
	"github.com/ethereum/go-ethereum/core/types"
)

// CharacterNFT is a high-level wrapper around the on-chain CharacterNFT contract.
type CharacterNFT struct {
	abi             abi.ABI
	address         common.Address
	contract        *bind.BoundContract
	contractBackend bind.ContractBackend
	transactOpts    *bind.TransactOpts
}

// NewCharacterNFT connects to an already-deployed CharacterNFT contract.
func NewCharacterNFT(opts *bind.TransactOpts, addr common.Address, backend bind.ContractBackend) (*CharacterNFT, error) {
	parsed, err := abi.JSON(strings.NewReader(contract.CharacterNFTABI))
	if err != nil {
		return nil, err
	}
	bound := bind.NewBoundContract(addr, parsed, backend, backend)
	return &CharacterNFT{
		abi:             parsed,
		address:         addr,
		contract:        bound,
		contractBackend: backend,
		transactOpts:    opts,
	}, nil
}

// ──────────────────────────────────────────────
//  Write methods
// ──────────────────────────────────────────────

// Mint creates a new character NFT on-chain.
// The caller must attach at least `mintFee` wei.
func (c *CharacterNFT) Mint(metadataURI string, traitHash [32]byte) (*types.Transaction, error) {
	return c.contract.Transact(c.transactOpts, "mint", metadataURI, traitHash)
}

// TransferFrom transfers a character NFT.  If value is attached it is treated
// as a sale — the platform takes its percentage and the remainder goes to the seller.
func (c *CharacterNFT) TransferFrom(tokenId *big.Int, to common.Address) (*types.Transaction, error) {
	return c.contract.Transact(c.transactOpts, "transferFrom", tokenId, to)
}

// Approve grants another address permission to transfer a specific token.
func (c *CharacterNFT) Approve(tokenId *big.Int, approved common.Address) (*types.Transaction, error) {
	return c.contract.Transact(c.transactOpts, "approve", tokenId, approved)
}

// AdvanceStage moves a character to the next pipeline stage (Text→Image→3D→Video→Licensed).
func (c *CharacterNFT) AdvanceStage(tokenId *big.Int, newMetadataURI string) (*types.Transaction, error) {
	return c.contract.Transact(c.transactOpts, "advanceStage", tokenId, newMetadataURI)
}

// SetMintFee updates the flat mint fee (platform-only).
func (c *CharacterNFT) SetMintFee(newFee *big.Int) (*types.Transaction, error) {
	return c.contract.Transact(c.transactOpts, "setMintFee", newFee)
}

// SetTransactionFee updates the secondary-sale fee in basis points (platform-only).
func (c *CharacterNFT) SetTransactionFee(newFeeBps *big.Int) (*types.Transaction, error) {
	return c.contract.Transact(c.transactOpts, "setTransactionFee", newFeeBps)
}

// TransferPlatform hands platform ownership to a new address (platform-only).
func (c *CharacterNFT) TransferPlatform(newPlatform common.Address) (*types.Transaction, error) {
	return c.contract.Transact(c.transactOpts, "transferPlatform", newPlatform)
}

// ──────────────────────────────────────────────
//  Read methods
// ──────────────────────────────────────────────

// CharacterInfo holds the on-chain data for a single character.
type CharacterInfo struct {
	Creator     common.Address
	CreatedAt   *big.Int
	Stage       uint8
	MetadataURI string
	TraitHash   [32]byte
}

// GetCharacter reads full character data from the contract.
func (c *CharacterNFT) GetCharacter(tokenId *big.Int) (*CharacterInfo, error) {
	var out []interface{}
	err := c.contract.Call(&bind.CallOpts{}, &out, "getCharacter", tokenId)
	if err != nil {
		return nil, err
	}
	info := &CharacterInfo{
		Creator:     out[0].(common.Address),
		CreatedAt:   out[1].(*big.Int),
		Stage:       out[2].(uint8),
		MetadataURI: out[3].(string),
		TraitHash:   out[4].([32]byte),
	}
	return info, nil
}

// OwnerOf returns the current owner of a token.
func (c *CharacterNFT) OwnerOf(tokenId *big.Int) (common.Address, error) {
	var out []interface{}
	err := c.contract.Call(&bind.CallOpts{}, &out, "ownerOf", tokenId)
	if err != nil {
		return common.Address{}, err
	}
	return out[0].(common.Address), nil
}

// BalanceOf returns how many characters an address owns.
func (c *CharacterNFT) BalanceOf(owner common.Address) (*big.Int, error) {
	var out []interface{}
	err := c.contract.Call(&bind.CallOpts{}, &out, "balanceOf", owner)
	if err != nil {
		return nil, err
	}
	return out[0].(*big.Int), nil
}

// TotalSupply returns the total number of minted characters.
func (c *CharacterNFT) TotalSupply() (*big.Int, error) {
	var out []interface{}
	err := c.contract.Call(&bind.CallOpts{}, &out, "totalSupply")
	if err != nil {
		return nil, err
	}
	return out[0].(*big.Int), nil
}

// MintFee returns the current flat mint fee in wei.
func (c *CharacterNFT) MintFee() (*big.Int, error) {
	var out []interface{}
	err := c.contract.Call(&bind.CallOpts{}, &out, "mintFee")
	if err != nil {
		return nil, err
	}
	return out[0].(*big.Int), nil
}

// TransactionFeeBps returns the current secondary-sale fee in basis points.
func (c *CharacterNFT) TransactionFeeBps() (*big.Int, error) {
	var out []interface{}
	err := c.contract.Call(&bind.CallOpts{}, &out, "transactionFeeBps")
	if err != nil {
		return nil, err
	}
	return out[0].(*big.Int), nil
}

// Platform returns the current platform wallet address.
func (c *CharacterNFT) Platform() (common.Address, error) {
	var out []interface{}
	err := c.contract.Call(&bind.CallOpts{}, &out, "platform")
	if err != nil {
		return common.Address{}, err
	}
	return out[0].(common.Address), nil
}
