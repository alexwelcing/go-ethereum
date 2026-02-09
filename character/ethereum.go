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

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	charcontract "github.com/ethereum/go-ethereum/contracts/character"
)

// EthereumBackend implements ChainBackend for the Ethereum CharacterNFT contract.
type EthereumBackend struct {
	nft  *charcontract.CharacterNFT
	opts *bind.TransactOpts
	fees *FeeSchedule
}

// NewEthereumBackend creates an Ethereum chain backend wired to a deployed
// CharacterNFT contract.
func NewEthereumBackend(nft *charcontract.CharacterNFT, opts *bind.TransactOpts, fees *FeeSchedule) *EthereumBackend {
	return &EthereumBackend{nft: nft, opts: opts, fees: fees}
}

func (e *EthereumBackend) Chain() ChainID { return ChainEthereum }

func (e *EthereumBackend) Mint(metadataURI string, traitHash [32]byte) (string, error) {
	oldValue := e.opts.Value
	e.opts.Value = e.fees.QuoteMint()
	tx, err := e.nft.Mint(metadataURI, traitHash)
	e.opts.Value = oldValue
	if err != nil {
		return "", err
	}
	return tx.Hash().Hex(), nil
}

func (e *EthereumBackend) TransferFrom(tokenID uint64, to string, salePrice *big.Int) (string, error) {
	oldValue := e.opts.Value
	e.opts.Value = salePrice
	tx, err := e.nft.TransferFrom(new(big.Int).SetUint64(tokenID), common.HexToAddress(to))
	e.opts.Value = oldValue
	if err != nil {
		return "", err
	}
	return tx.Hash().Hex(), nil
}

func (e *EthereumBackend) AdvanceStage(tokenID uint64, newMetadataURI string) (string, error) {
	tx, err := e.nft.AdvanceStage(new(big.Int).SetUint64(tokenID), newMetadataURI)
	if err != nil {
		return "", err
	}
	return tx.Hash().Hex(), nil
}

func (e *EthereumBackend) GetCharacter(tokenID uint64) (*OnChainCharacter, error) {
	info, err := e.nft.GetCharacter(new(big.Int).SetUint64(tokenID))
	if err != nil {
		return nil, err
	}
	return &OnChainCharacter{
		Creator:     info.Creator.Hex(),
		CreatedAt:   info.CreatedAt.Uint64(),
		Stage:       info.Stage,
		MetadataURI: info.MetadataURI,
		TraitHash:   info.TraitHash,
	}, nil
}

func (e *EthereumBackend) OwnerOf(tokenID uint64) (string, error) {
	addr, err := e.nft.OwnerOf(new(big.Int).SetUint64(tokenID))
	if err != nil {
		return "", err
	}
	return addr.Hex(), nil
}

func (e *EthereumBackend) BalanceOf(owner string) (uint64, error) {
	bal, err := e.nft.BalanceOf(common.HexToAddress(owner))
	if err != nil {
		return 0, err
	}
	return bal.Uint64(), nil
}

func (e *EthereumBackend) TotalSupply() (uint64, error) {
	supply, err := e.nft.TotalSupply()
	if err != nil {
		return 0, err
	}
	return supply.Uint64(), nil
}

func (e *EthereumBackend) MintFee() (*big.Int, error) {
	return e.nft.MintFee()
}

func (e *EthereumBackend) TransactionFeeBps() (*big.Int, error) {
	return e.nft.TransactionFeeBps()
}

func (e *EthereumBackend) PlatformAddress() (string, error) {
	addr, err := e.nft.Platform()
	if err != nil {
		return "", err
	}
	return addr.Hex(), nil
}
