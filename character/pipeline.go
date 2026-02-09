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
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// Errors returned by pipeline operations.
var (
	ErrNoTraits        = errors.New("character: at least one trait is required")
	ErrAlreadyLicensed = errors.New("character: character is already at the final stage")
	ErrInvalidStage    = errors.New("character: invalid stage transition")
)

// StageProcessor is the interface that external services must implement to
// handle a specific stage transition.  The platform does NOT perform any
// media generation itself — it delegates to pluggable processors.
//
// Example implementations (external):
//   TextToImage   — calls a diffusion model API
//   ImageTo3D     — calls a 3-D reconstruction API
//   ModelToVideo  — calls a video rendering API
//   VideoToLicensed — triggers licensing/model-training workflow
type StageProcessor interface {
	// Process takes the current character metadata, performs the stage
	// transformation, and returns the URI of the newly created asset.
	// The implementation is responsible for uploading the asset to IPFS
	// or any other storage backend.
	Process(meta *CharacterMeta) (assetURI string, err error)

	// Stage returns which stage this processor handles.
	Stage() Stage
}

// Pipeline orchestrates the character lifecycle from text to licensed model.
// It holds one StageProcessor per transition and validates ordering.
type Pipeline struct {
	processors map[Stage]StageProcessor
}

// NewPipeline creates an empty pipeline.  Register processors before use.
func NewPipeline() *Pipeline {
	return &Pipeline{
		processors: make(map[Stage]StageProcessor),
	}
}

// Register adds a stage processor.  It replaces any existing processor for
// the same stage.
func (p *Pipeline) Register(proc StageProcessor) {
	p.processors[proc.Stage()] = proc
}

// Advance moves a character to the next stage by delegating to the
// registered processor.  Returns the updated metadata and the asset URI
// produced by the processor.
func (p *Pipeline) Advance(meta *CharacterMeta) (assetURI string, err error) {
	if meta.Stage >= StageLicensed {
		return "", ErrAlreadyLicensed
	}

	nextStage := meta.Stage + 1
	proc, ok := p.processors[nextStage]
	if !ok {
		return "", fmt.Errorf("character: no processor registered for stage %s", nextStage)
	}

	assetURI, err = proc.Process(meta)
	if err != nil {
		return "", fmt.Errorf("character: stage %s processor failed: %v", nextStage, err)
	}

	// Update metadata
	if meta.Assets == nil {
		meta.Assets = make(map[string]string)
	}
	meta.Assets[nextStage.String()] = assetURI
	meta.Stage = nextStage

	return assetURI, nil
}

// HashTraits computes the keccak256 hash of the canonical JSON encoding of
// the provided traits.  Traits are sorted by (category, name) to ensure
// deterministic hashing regardless of input order.
func HashTraits(traits []Trait) common.Hash {
	// Sort for determinism
	sorted := make([]Trait, len(traits))
	copy(sorted, traits)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Category != sorted[j].Category {
			return sorted[i].Category < sorted[j].Category
		}
		return sorted[i].Name < sorted[j].Name
	})

	canonical, _ := json.Marshal(sorted)
	return common.BytesToHash(crypto.Keccak256(canonical))
}

// NewCharacterMeta creates an initial metadata object at StageText from a
// set of traits.  It computes the trait hash and validates inputs.
func NewCharacterMeta(name string, creator common.Address, traits []Trait) (*CharacterMeta, error) {
	if len(traits) == 0 {
		return nil, ErrNoTraits
	}
	return &CharacterMeta{
		Creator:   creator,
		Name:      name,
		Traits:    traits,
		Stage:     StageText,
		Assets:    make(map[string]string),
		TraitHash: HashTraits(traits),
	}, nil
}
