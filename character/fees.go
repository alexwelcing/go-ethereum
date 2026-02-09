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
	"errors"
	"math/big"
)

// Fee-related constants.
var (
	// DefaultMintFee is 0.01 ETH expressed in wei.
	DefaultMintFee = new(big.Int).Mul(big.NewInt(10), new(big.Int).Exp(big.NewInt(10), big.NewInt(15), nil)) // 0.01 ETH

	// DefaultTransactionFeeBps is 250 basis points (2.5%).
	DefaultTransactionFeeBps = big.NewInt(250)

	// MaxTransactionFeeBps caps the fee at 100% (10000 bps) to match the contract.
	MaxTransactionFeeBps = big.NewInt(10000)

	// BpsBase is the denominator for basis-point math.
	BpsBase = big.NewInt(10000)
)

// Errors for fee validation.
var (
	ErrFeeTooHigh    = errors.New("character: transaction fee exceeds 10000 bps")
	ErrNegativeFee   = errors.New("character: fee cannot be negative")
	ErrNegativePrice = errors.New("character: sale price cannot be negative")
)

// FeeSchedule holds the platform's current fee parameters.
// It is used off-chain for quoting and validation before submitting transactions.
type FeeSchedule struct {
	MintFee           *big.Int // flat fee in wei charged on every mint
	TransactionFeeBps *big.Int // basis points taken on secondary sales
}

// NewDefaultFeeSchedule returns a FeeSchedule with sensible defaults
// (0.01 ETH mint fee, 2.5% transaction fee).
func NewDefaultFeeSchedule() *FeeSchedule {
	return &FeeSchedule{
		MintFee:           new(big.Int).Set(DefaultMintFee),
		TransactionFeeBps: new(big.Int).Set(DefaultTransactionFeeBps),
	}
}

// NewFeeSchedule creates a FeeSchedule with the given parameters after validation.
func NewFeeSchedule(mintFee, txFeeBps *big.Int) (*FeeSchedule, error) {
	if mintFee.Sign() < 0 {
		return nil, ErrNegativeFee
	}
	if txFeeBps.Sign() < 0 {
		return nil, ErrNegativeFee
	}
	if txFeeBps.Cmp(MaxTransactionFeeBps) > 0 {
		return nil, ErrFeeTooHigh
	}
	return &FeeSchedule{
		MintFee:           new(big.Int).Set(mintFee),
		TransactionFeeBps: new(big.Int).Set(txFeeBps),
	}, nil
}

// PlatformCut calculates the platform's cut of a secondary sale given the
// total sale price.  Returns (platformCut, sellerProceeds).
func (fs *FeeSchedule) PlatformCut(salePrice *big.Int) (platformCut, sellerProceeds *big.Int, err error) {
	if salePrice.Sign() < 0 {
		return nil, nil, ErrNegativePrice
	}
	// platformCut = salePrice * transactionFeeBps / 10000
	platformCut = new(big.Int).Mul(salePrice, fs.TransactionFeeBps)
	platformCut.Div(platformCut, BpsBase)

	sellerProceeds = new(big.Int).Sub(salePrice, platformCut)
	return platformCut, sellerProceeds, nil
}

// QuoteMint returns the total cost a user must send to mint a character.
// Currently this equals the flat mint fee, but this method exists to
// accommodate future dynamic pricing.
func (fs *FeeSchedule) QuoteMint() *big.Int {
	return new(big.Int).Set(fs.MintFee)
}
