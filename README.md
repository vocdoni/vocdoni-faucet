# vocdoni-faucet

[![GoDoc](https://godoc.org/go.vocdoni.io/vocdoni-faucet?status.svg)](https://godoc.org/go.vocdoni.io/vocdoni-faucet)
[![Go Report Card](https://goreportcard.com/badge/go.vocdoni.io/vocdoni-faucet)](https://goreportcard.com/report/go.vocdoni.io/vocdoni-faucet)
[![Coverage Status](https://coveralls.io/repos/github/vocdoni/vocdoni-faucet/badge.svg)](https://coveralls.io/github/vocdoni/vocdoni-faucet)


[![Join Discord](https://img.shields.io/badge/discord-join%20chat-blue.svg)](https://discord.gg/4hKeArDaU2)
[![Twitter Follow](https://img.shields.io/twitter/follow/vocdoni.svg?style=social&label=Follow)](https://twitter.com/vocdoni)

[![Contributor Covenant](https://img.shields.io/badge/Contributor%20Covenant-v1.4%20adopted-ff69b4.svg)](code-of-conduct.md) [![License: AGPL v3](https://img.shields.io/badge/License-AGPL%20v3-blue.svg)](https://www.gnu.org/licenses/agpl-3.0)

This repository contains the source code of the Vocdoni faucet.

The Vocdoni faucet is a service for distributing Vocdoni and EVM based blockchains tokens, it exposes a REST API
for requesting such tokens.

## Build

Compile from source in a golang environment (Go>1.18 required):

```
git clone https://go.vocdoni.io/vocdoni-faucet.git
cd vocdoni-faucet
go build ./cmd/main.go
```

## Usage

`go run ./cmd/main.go`

Options:
- `--apiListenHost` **string**         API endpoint listen address (default "0.0.0.0")
- `--apiListenPort` **int**            API endpoint http port (default 8000)
- `--apiRoute` **string**              dvote API route (default "/")
- `--apiTLSDomain` **string**          enable TLS secure API domain with LetsEncrypt auto-generated certificate
- `--dataDir` **string**               directory where data is stored (default "/home/me/.faucet")
- `--enableEVM`                        enable evm faucet (default true)
- `--enableMetrics`                    enable prometheus metrics (default true)
- `--enableVocdoni`                    enable vocdoni faucet (default true)
- `--evmEndpoints` **stringArray**     evm endpoints to connect with (requied for the evm faucet)
- `--evmNetwork` **string**            one of the available evm chains
- `--evmPrivKeys` **stringArray**      hexString privKeys for EVM faucet accounts
- `--faucetAmount` **uint**            faucet amount (default 100)
- `--faucetAmountThreshold` **uint**   minimum amount threshold for transfer (default 100)
- `--faucetEnableChallenge`            if true a faucet challenge must be solved
- `--logErrorFile` **string**          log errors and warnings to a file
- `--logLevel` **string**              log level (debug, info, warn, error, fatal) (default "info")
- `--logOutput` **string**             log output (stdout, stderr or filepath) (default "stdout")
- `--metricsRefreshInterval` **int**   metrics refresh interval in seconds (default 10)
- `--saveConfig`                       overwrites an existing config file with the CLI provided flags
- `--vocdoniNetwork` **string**        one of the available vocdoni networks
- `--vocdoniPrivKey` **string**        hexString privKeys for vocdoni faucet accounts

## API

- Request (Vocdoni)

    `curl -X GET https://foo.bar/faucet/vocdoni/<network>/<from>`

    - `<network>` one of `[dev, stage, azeno]`
    - `<from>` an EVM address (i.e `0xeD33259a056F4fb449FFB7B7E2eCB43a9B5685Bf`)

- Response (Vocdoni)

    HTTP 200

    ```json
    {
        "amount": 100,
        "faucetPackage": "0xabc" // faucet package bytes
    }
    ```

    HTTP 400

    ```json
    {
        "error": "Message goes here"
    }
    ```

- Request (EVM)

    `curl -X GET https://foo.bar/faucet/evm/<network>/<from>`

    - `<network>` one of `[mainnet, goerli, sepolia, matic, mumbai, gnosisChain, evmtest]`
    - `<from>` an EVM address (i.e `0xeD33259a056F4fb449FFB7B7E2eCB43a9B5685Bf`)

- Response (EVM)

    HTTP 200

    ```json
    {
        "amount": 100,
        "txHash": "0x123"
    }
    ```

    HTTP 400

    ```json
    {
        "error": "Message goes here"
    }
    ```
