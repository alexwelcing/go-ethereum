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
// It connects to an Ethereum node, wires up the CharacterNFT contract,
// and exposes a JSON-RPC API for minting and managing characters.
//
// Usage:
//   charservice --rpc <endpoint> --contract <address> --keyfile <path> [--mintfee <wei>] [--txfee <bps>]
package main

import (
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/log"
	cli "gopkg.in/urfave/cli.v1"
)

var (
	app = cli.NewApp()

	// Flags
	rpcFlag = cli.StringFlag{
		Name:  "rpc",
		Usage: "Ethereum JSON-RPC endpoint (e.g. http://localhost:8545)",
		Value: "http://localhost:8545",
	}
	contractFlag = cli.StringFlag{
		Name:  "contract",
		Usage: "Deployed CharacterNFT contract address",
	}
	keyfileFlag = cli.StringFlag{
		Name:  "keyfile",
		Usage: "Path to the JSON keyfile for the platform wallet",
	}
	mintFeeFlag = cli.StringFlag{
		Name:  "mintfee",
		Usage: "Mint fee in wei (default: 10000000000000000 = 0.01 ETH)",
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
)

func init() {
	app.Name = "charservice"
	app.Usage = "Text-to-Character NFT platform service"
	app.Version = "0.1.0"
	app.Action = run
	app.Flags = []cli.Flag{
		rpcFlag,
		contractFlag,
		keyfileFlag,
		mintFeeFlag,
		txFeeFlag,
		listenFlag,
	}
	app.Commands = []cli.Command{
		{
			Name:   "info",
			Usage:  "Print contract and fee information",
			Action: infoCmd,
			Flags: []cli.Flag{
				rpcFlag,
				contractFlag,
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

	if !ctx.IsSet("contract") {
		utils.Fatalf("--contract flag is required")
	}

	log.Info("Character service starting",
		"rpc", ctx.String("rpc"),
		"contract", ctx.String("contract"),
		"listen", ctx.String("listen"),
		"mintFee", ctx.String("mintfee"),
		"txFee", ctx.Int64("txfee"),
	)

	// TODO: Wire up ethclient, contract binding, metadata store, and HTTP server.
	// This is the bootstrap skeleton — the full wiring depends on
	// deployment choices (IPFS vs S3 for metadata, keystore location, etc.)

	log.Info("Character service ready", "listen", ctx.String("listen"))
	log.Info("Revenue model: upfront mint fee + percentage of all secondary transactions")

	// Block forever (in production, start HTTP server here)
	select {}
}

func infoCmd(ctx *cli.Context) error {
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlInfo, log.StreamHandler(os.Stderr, log.TerminalFormat(true))))

	if !ctx.IsSet("contract") {
		utils.Fatalf("--contract flag is required")
	}

	log.Info("CharacterNFT contract info",
		"address", ctx.String("contract"),
		"rpc", ctx.String("rpc"),
	)

	// TODO: Connect to node and read contract state (totalSupply, fees, etc.)

	return nil
}
