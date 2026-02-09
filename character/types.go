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

// Package character defines the off-chain domain model for the text-to-character
// pipeline.  It is chain-agnostic — Ethereum and Solana (and future chains)
// plug in via the ChainBackend interface defined in chain.go.
package character

// Stage mirrors the on-chain character stage enum (same values on all chains).
type Stage uint8

const (
	StageText     Stage = iota // raw text attributes provided by user
	StageImage                 // 2-D image generated from text
	StageModel3D               // 3-D model derived from image
	StageVideo                 // animated / video character
	StageLicensed              // personality licensed into a model
)

// String returns a human-readable stage name.
func (s Stage) String() string {
	switch s {
	case StageText:
		return "text"
	case StageImage:
		return "image"
	case StageModel3D:
		return "3d_model"
	case StageVideo:
		return "video"
	case StageLicensed:
		return "licensed"
	default:
		return "unknown"
	}
}

// Trait is a single text-based attribute that feeds the character pipeline.
// Example: {Category: "personality", Name: "humor", Value: "dry wit"}
type Trait struct {
	Category string `json:"category"` // e.g. "appearance", "personality", "backstory"
	Name     string `json:"name"`     // attribute key
	Value    string `json:"value"`    // attribute value (free text)
}

// CharacterMeta is the off-chain metadata associated with a minted character.
// It is stored at the URI recorded on-chain and follows a simple JSON schema
// so third-party renderers and marketplaces can consume it.
//
// Addresses are stored as strings so the same structure works for Ethereum
// (0x-hex) and Solana (base58).
type CharacterMeta struct {
	// TokenID is the on-chain token identifier (set after minting).
	TokenID uint64 `json:"token_id"`

	// Chain identifies which blockchain this character lives on.
	Chain ChainID `json:"chain"`

	// Creator is the address of the original minter (hex or base58).
	Creator string `json:"creator"`

	// Name is the user-chosen display name.
	Name string `json:"name"`

	// Traits are the raw text attributes that define the character.
	Traits []Trait `json:"traits"`

	// Stage is the current pipeline stage.
	Stage Stage `json:"stage"`

	// Assets maps stage names to URIs (e.g. "image" → "ipfs://Qm…").
	// Populated progressively as the character advances through the pipeline.
	Assets map[string]string `json:"assets,omitempty"`

	// TraitHash is the keccak256 of the canonical trait encoding, matching
	// the on-chain traitHash field for provenance verification.
	// Both Ethereum and Solana programs can verify keccak256.
	TraitHash [32]byte `json:"trait_hash"`
}
