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

// Package solana defines the Anchor IDL and account schemas for the
// CharacterNFT Solana program — the Solana-side equivalent of the
// Ethereum CharacterNFT contract.
package solana

// CharacterIDL is the Anchor IDL for the character_nft Solana program.
// It describes the same logical operations as the Ethereum contract:
// mint, transfer_from, advance_stage, and platform fee management.
const CharacterIDL = `{
  "version": "0.1.0",
  "name": "character_nft",
  "instructions": [
    {
      "name": "initialize",
      "accounts": [
        {"name": "platform", "isMut": true, "isSigner": true},
        {"name": "state", "isMut": true, "isSigner": false},
        {"name": "systemProgram", "isMut": false, "isSigner": false}
      ],
      "args": [
        {"name": "mintFeeLamports", "type": "u64"},
        {"name": "transactionFeeBps", "type": "u16"}
      ]
    },
    {
      "name": "mint",
      "accounts": [
        {"name": "creator", "isMut": true, "isSigner": true},
        {"name": "state", "isMut": true, "isSigner": false},
        {"name": "character", "isMut": true, "isSigner": false},
        {"name": "platform", "isMut": true, "isSigner": false},
        {"name": "systemProgram", "isMut": false, "isSigner": false}
      ],
      "args": [
        {"name": "metadataUri", "type": "string"},
        {"name": "traitHash", "type": {"array": ["u8", 32]}}
      ]
    },
    {
      "name": "transferFrom",
      "accounts": [
        {"name": "owner", "isMut": true, "isSigner": true},
        {"name": "character", "isMut": true, "isSigner": false},
        {"name": "recipient", "isMut": true, "isSigner": false},
        {"name": "platform", "isMut": true, "isSigner": false},
        {"name": "state", "isMut": false, "isSigner": false},
        {"name": "systemProgram", "isMut": false, "isSigner": false}
      ],
      "args": [
        {"name": "salePriceLamports", "type": "u64"}
      ]
    },
    {
      "name": "advanceStage",
      "accounts": [
        {"name": "owner", "isMut": false, "isSigner": true},
        {"name": "character", "isMut": true, "isSigner": false}
      ],
      "args": [
        {"name": "newMetadataUri", "type": "string"}
      ]
    },
    {
      "name": "setMintFee",
      "accounts": [
        {"name": "platform", "isMut": false, "isSigner": true},
        {"name": "state", "isMut": true, "isSigner": false}
      ],
      "args": [
        {"name": "newFeeLamports", "type": "u64"}
      ]
    },
    {
      "name": "setTransactionFee",
      "accounts": [
        {"name": "platform", "isMut": false, "isSigner": true},
        {"name": "state", "isMut": true, "isSigner": false}
      ],
      "args": [
        {"name": "newFeeBps", "type": "u16"}
      ]
    }
  ],
  "accounts": [
    {
      "name": "ProgramState",
      "type": {
        "kind": "struct",
        "fields": [
          {"name": "platform", "type": "publicKey"},
          {"name": "mintFeeLamports", "type": "u64"},
          {"name": "transactionFeeBps", "type": "u16"},
          {"name": "nextTokenId", "type": "u64"}
        ]
      }
    },
    {
      "name": "Character",
      "type": {
        "kind": "struct",
        "fields": [
          {"name": "tokenId", "type": "u64"},
          {"name": "creator", "type": "publicKey"},
          {"name": "owner", "type": "publicKey"},
          {"name": "createdAt", "type": "i64"},
          {"name": "stage", "type": "u8"},
          {"name": "metadataUri", "type": "string"},
          {"name": "traitHash", "type": {"array": ["u8", 32]}}
        ]
      }
    }
  ],
  "errors": [
    {"code": 6000, "name": "AlreadyLicensed", "msg": "Character is already at the final stage"},
    {"code": 6001, "name": "NotOwner", "msg": "Only the owner can perform this action"},
    {"code": 6002, "name": "FeeTooHigh", "msg": "Transaction fee exceeds 10000 bps"},
    {"code": 6003, "name": "InsufficientFunds", "msg": "Insufficient lamports for mint fee"}
  ]
}`

// ProgramStateSize is the byte size of the on-chain ProgramState account.
// 8 (discriminator) + 32 (platform pubkey) + 8 (mint fee) + 2 (tx fee bps) + 8 (next token id)
const ProgramStateSize = 8 + 32 + 8 + 2 + 8

// CharacterBaseSize is the minimum byte size of a Character account
// (excluding the variable-length metadataUri string).
// 8 (discriminator) + 8 (tokenId) + 32 (creator) + 32 (owner) + 8 (createdAt) + 1 (stage) + 4 (string len prefix) + 32 (traitHash)
const CharacterBaseSize = 8 + 8 + 32 + 32 + 8 + 1 + 4 + 32
