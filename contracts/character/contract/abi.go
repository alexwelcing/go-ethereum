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

// Package contract contains the ABI and binding stubs for CharacterNFT.
// Once solc is available, regenerate with:
//   abigen --sol contract/character.sol --pkg contract --out contract/character_gen.go
package contract

// CharacterNFTABI is the ABI of the CharacterNFT contract.
const CharacterNFTABI = `[
	{
		"constant": false,
		"inputs": [
			{"name": "_metadataURI", "type": "string"},
			{"name": "_traitHash",   "type": "bytes32"}
		],
		"name": "mint",
		"outputs": [{"name": "tokenId", "type": "uint256"}],
		"payable": true,
		"type": "function"
	},
	{
		"constant": false,
		"inputs": [
			{"name": "_tokenId", "type": "uint256"},
			{"name": "_to",      "type": "address"}
		],
		"name": "transferFrom",
		"outputs": [],
		"payable": true,
		"type": "function"
	},
	{
		"constant": false,
		"inputs": [
			{"name": "_tokenId",  "type": "uint256"},
			{"name": "_approved", "type": "address"}
		],
		"name": "approve",
		"outputs": [],
		"payable": false,
		"type": "function"
	},
	{
		"constant": false,
		"inputs": [
			{"name": "_tokenId",       "type": "uint256"},
			{"name": "_newMetadataURI", "type": "string"}
		],
		"name": "advanceStage",
		"outputs": [],
		"payable": false,
		"type": "function"
	},
	{
		"constant": false,
		"inputs": [{"name": "_newFee", "type": "uint256"}],
		"name": "setMintFee",
		"outputs": [],
		"payable": false,
		"type": "function"
	},
	{
		"constant": false,
		"inputs": [{"name": "_newFeeBps", "type": "uint256"}],
		"name": "setTransactionFee",
		"outputs": [],
		"payable": false,
		"type": "function"
	},
	{
		"constant": false,
		"inputs": [{"name": "_newPlatform", "type": "address"}],
		"name": "transferPlatform",
		"outputs": [],
		"payable": false,
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [{"name": "_tokenId", "type": "uint256"}],
		"name": "getCharacter",
		"outputs": [
			{"name": "creator",     "type": "address"},
			{"name": "createdAt",   "type": "uint256"},
			{"name": "stage",       "type": "uint8"},
			{"name": "metadataURI", "type": "string"},
			{"name": "traitHash",   "type": "bytes32"}
		],
		"payable": false,
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [{"name": "_tokenId", "type": "uint256"}],
		"name": "ownerOf",
		"outputs": [{"name": "", "type": "address"}],
		"payable": false,
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [{"name": "_owner", "type": "address"}],
		"name": "balanceOf",
		"outputs": [{"name": "", "type": "uint256"}],
		"payable": false,
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [],
		"name": "totalSupply",
		"outputs": [{"name": "", "type": "uint256"}],
		"payable": false,
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [],
		"name": "platform",
		"outputs": [{"name": "", "type": "address"}],
		"payable": false,
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [],
		"name": "mintFee",
		"outputs": [{"name": "", "type": "uint256"}],
		"payable": false,
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [],
		"name": "transactionFeeBps",
		"outputs": [{"name": "", "type": "uint256"}],
		"payable": false,
		"type": "function"
	},
	{
		"anonymous": false,
		"inputs": [
			{"indexed": true,  "name": "tokenId",     "type": "uint256"},
			{"indexed": true,  "name": "creator",     "type": "address"},
			{"indexed": false, "name": "traitHash",   "type": "bytes32"},
			{"indexed": false, "name": "metadataURI", "type": "string"}
		],
		"name": "CharacterMinted",
		"type": "event"
	},
	{
		"anonymous": false,
		"inputs": [
			{"indexed": true,  "name": "tokenId",     "type": "uint256"},
			{"indexed": true,  "name": "from",         "type": "address"},
			{"indexed": true,  "name": "to",           "type": "address"},
			{"indexed": false, "name": "price",        "type": "uint256"},
			{"indexed": false, "name": "platformCut",  "type": "uint256"}
		],
		"name": "Transfer",
		"type": "event"
	},
	{
		"anonymous": false,
		"inputs": [
			{"indexed": true,  "name": "tokenId",        "type": "uint256"},
			{"indexed": false, "name": "newStage",        "type": "uint8"},
			{"indexed": false, "name": "newMetadataURI",  "type": "string"}
		],
		"name": "StageAdvanced",
		"type": "event"
	}
]`
