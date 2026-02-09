// Copyright 2018 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

// charservice boots the text-to-character platform service.
//
// It connects to Ethereum and/or Solana, wires up the CharacterNFT
// contracts/programs, and exposes a JSON-RPC API for minting and
// managing characters across chains.
//
// Usage:
//   charservice [flags]
//
// Ethereum:
//   charservice --eth.rpc <endpoint> --eth.contract <address> --eth.keyfile <path>
//
// Solana:
//   charservice --sol.rpc <endpoint> --sol.program <address> --sol.state <address>
//
// Both:
//   charservice --eth.rpc ... --eth.contract ... --sol.rpc ... --sol.program ...
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/log"
	cli "gopkg.in/urfave/cli.v1"
)

var (
	app = cli.NewApp()

	// ── Global flags ──────────────────────────────────────────
	mintFeeFlag = cli.StringFlag{
		Name:  "mintfee",
		Usage: "Mint fee in smallest unit (wei/lamports) (default: 10000000000000000 = 0.01 ETH)",
		Value: "10000000000000000",
	}
	txFeeFlag = cli.Int64Flag{
		Name:  "txfee",
		Usage: "Transaction fee in basis points (default: 250 = 2.5%)",
		Value: 250,
	}
	listenFlag = cli.StringFlag{
		Name:  "listen",
		Usage: "HTTP listen address for JSON-RPC API",
		Value: ":8550",
	}

	// ── Ethereum flags ────────────────────────────────────────
	ethRPCFlag = cli.StringFlag{
		Name:  "eth.rpc",
		Usage: "Ethereum JSON-RPC endpoint (e.g. http://localhost:8545)",
	}
	ethContractFlag = cli.StringFlag{
		Name:  "eth.contract",
		Usage: "Deployed CharacterNFT contract address on Ethereum",
	}
	ethKeyfileFlag = cli.StringFlag{
		Name:  "eth.keyfile",
		Usage: "Path to the JSON keyfile for the Ethereum platform wallet",
	}

	// ── Solana flags ──────────────────────────────────────────
	solRPCFlag = cli.StringFlag{
		Name:  "sol.rpc",
		Usage: "Solana JSON-RPC endpoint (e.g. https://api.mainnet-beta.solana.com)",
	}
	solProgramFlag = cli.StringFlag{
		Name:  "sol.program",
		Usage: "Deployed character_nft program ID on Solana (base58)",
	}
	solStateFlag = cli.StringFlag{
		Name:  "sol.state",
		Usage: "ProgramState account address on Solana (base58)",
	}
	solKeypairFlag = cli.StringFlag{
		Name:  "sol.keypair",
		Usage: "Path to the Solana platform wallet keypair JSON",
	}
)

func init() {
	app.Name = "charservice"
	app.Usage = "Multi-chain text-to-character NFT platform service (Ethereum + Solana)"
	app.Version = "0.2.0"
	app.Action = run
	app.Flags = []cli.Flag{
		mintFeeFlag,
		txFeeFlag,
		listenFlag,
		// Ethereum
		ethRPCFlag,
		ethContractFlag,
		ethKeyfileFlag,
		// Solana
		solRPCFlag,
		solProgramFlag,
		solStateFlag,
		solKeypairFlag,
	}
	app.Commands = []cli.Command{
		{
			Name:   "info",
			Usage:  "Print contract/program and fee information",
			Action: infoCmd,
			Flags: []cli.Flag{
				ethRPCFlag, ethContractFlag,
				solRPCFlag, solProgramFlag, solStateFlag,
			},
		},
	}
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx *cli.Context) error {
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlInfo, log.StreamHandler(os.Stderr, log.TerminalFormat(true))))

	ethEnabled := ctx.IsSet("eth.rpc") && ctx.IsSet("eth.contract")
	solEnabled := ctx.IsSet("sol.rpc") && ctx.IsSet("sol.program")

	if !ethEnabled && !solEnabled {
		utils.Fatalf("At least one chain must be configured. Use --eth.rpc + --eth.contract for Ethereum, or --sol.rpc + --sol.program for Solana.")
	}

	var chains []string
	if ethEnabled {
		chains = append(chains, "ethereum")
	}
	if solEnabled {
		chains = append(chains, "solana")
	}

	log.Info("Character service starting",
		"chains", strings.Join(chains, ","),
		"listen", ctx.String("listen"),
		"mintFee", ctx.String("mintfee"),
		"txFee", ctx.Int64("txfee"),
	)

	if ethEnabled {
		log.Info("Ethereum backend configured",
			"rpc", ctx.String("eth.rpc"),
			"contract", ctx.String("eth.contract"),
		)
		// TODO: Wire up ethclient.Dial, contract binding, EthereumBackend
	}

	if solEnabled {
		log.Info("Solana backend configured",
			"rpc", ctx.String("sol.rpc"),
			"program", ctx.String("sol.program"),
			"state", ctx.String("sol.state"),
		)
		// TODO: Wire up SolanaBackend with config
	}

	log.Info("Character service ready",
		"listen", ctx.String("listen"),
		"chains", strings.Join(chains, ","),
	)
	log.Info("Revenue model: upfront mint fee + percentage of all secondary transactions on every chain")

	// Block forever (in production, start HTTP server here)
	select {}
}

func infoCmd(ctx *cli.Context) error {
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlInfo, log.StreamHandler(os.Stderr, log.TerminalFormat(true))))

	if ctx.IsSet("eth.contract") {
		log.Info("Ethereum CharacterNFT",
			"contract", ctx.String("eth.contract"),
			"rpc", ctx.String("eth.rpc"),
		)
	}
	if ctx.IsSet("sol.program") {
		log.Info("Solana character_nft",
			"program", ctx.String("sol.program"),
			"state", ctx.String("sol.state"),
			"rpc", ctx.String("sol.rpc"),
		)
	}

	// TODO: Connect to nodes and read contract/program state

	return nil
}
