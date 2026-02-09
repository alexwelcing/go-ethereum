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
	"math/big"
)

// ChainID identifies which blockchain a backend targets.
type ChainID string

const (
	ChainEthereum ChainID = "ethereum"
	ChainSolana   ChainID = "solana"
)

// ChainBackend is the interface that each blockchain adapter must implement.
// It abstracts the on-chain operations so the service layer is chain-agnostic.
// Addresses are represented as strings because Ethereum uses 0x-prefixed hex
// while Solana uses base58.
type ChainBackend interface {
	// Chain returns which blockchain this backend targets.
	Chain() ChainID

	// Mint creates a new character NFT on-chain.
	// The implementation is responsible for attaching the correct mint fee.
	// Returns the transaction signature/hash as a hex or base58 string.
	Mint(metadataURI string, traitHash [32]byte) (txHash string, err error)

	// TransferFrom transfers a character, optionally as a sale.
	// If salePrice > 0, the contract/program takes the platform cut.
	TransferFrom(tokenID uint64, to string, salePrice *big.Int) (txHash string, err error)

	// AdvanceStage moves a character to the next pipeline stage on-chain.
	AdvanceStage(tokenID uint64, newMetadataURI string) (txHash string, err error)

	// GetCharacter reads full on-chain character data.
	GetCharacter(tokenID uint64) (*OnChainCharacter, error)

	// OwnerOf returns the current owner address as a string.
	OwnerOf(tokenID uint64) (string, error)

	// BalanceOf returns how many characters an address owns.
	BalanceOf(owner string) (uint64, error)

	// TotalSupply returns total minted characters.
	TotalSupply() (uint64, error)

	// MintFee returns the current mint fee in the chain's smallest unit
	// (wei for Ethereum, lamports for Solana).
	MintFee() (*big.Int, error)

	// TransactionFeeBps returns the secondary-sale fee in basis points.
	TransactionFeeBps() (*big.Int, error)

	// PlatformAddress returns the platform fee receiver address.
	PlatformAddress() (string, error)
}

// OnChainCharacter holds the data read from any chain's character record.
type OnChainCharacter struct {
	Creator     string   `json:"creator"`
	CreatedAt   uint64   `json:"created_at"`
	Stage       uint8    `json:"stage"`
	MetadataURI string   `json:"metadata_uri"`
	TraitHash   [32]byte `json:"trait_hash"`
}
