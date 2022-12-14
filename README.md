# vocdoni-faucet

[![GoDoc](https://godoc.org/go.vocdoni.io/faucet?status.svg)](https://godoc.org/go.vocdoni.io/faucet)
[![Go Report Card](https://goreportcard.com/badge/go.vocdoni.io/faucet)](https://goreportcard.com/report/go.vocdoni.io/faucet)
[![Coverage Status](https://coveralls.io/repos/github/vocdoni/vocdoni-faucet/badge.svg?branch=master)](https://coveralls.io/github/vocdoni/vocdoni-faucet?branch=master)

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

- `--apiListenHost` **string**                API endpoint listen address (default "0.0.0.0")
- `--apiListenPort` **int**                   API endpoint http port (default 8000)
- `--apiRoute` **string**                     dvote API route (default "/")
- `--apiTLSDomain` **string**                 enaapiLle TLS secure API domain with LetsEncrypt auto-generated certificate
- `--apiWhitelist` **string**                 bearer token whitelist for accepting requests (comma separated string)
- `--dataDir` **string**                      directory where data is stored (default "/home/me/.faucet")
- `--enableEVM` **bool**                      enable evm faucet (default true)
- `--enableVocdoni` **bool**                  enable vocdoni faucet (default true)
- `--evmEndpoints` **StringSlice**            evm endpoints to connect with (requied for the evm faucet)
- `--evmNetwork` **string**                   one of the available evm chains
- `--evmPrivKeys` **StringSlice**             hexString privKeys for EVM faucet accounts
- `--faucetEVMAmount` **uint**                evm faucet amount in wei (1000000000000000000 == 1 ETH) (default 1)
- `--faucetEVMAmountThreshold` **uint**       minimum EVM amount threshold for transfer (default 1)
- `--faucetEVMEnableChallenge` **bool**       if true a EVM faucet challenge must be solved
- `--faucetVocdoniAmount` **uint**            vocdoni faucet amount (default 100)
- `--faucetVocdoniAmountThreshold` **uint**   minimum vocdoni amount threshold for transfer (default 100)
- `--faucetVocdoniEnableChallenge` **bool**   if true a vocdoni faucet challenge must be solved
- `--logErrorFile` **string**                 log errors and warnings to a file
- `--logLevel` **string**                     log level (debug, info, warn, error, fatal) (default "info")
- `--logOutput` **string**                    log output (stdout, stderr or filepath) (default "stdout")
- `--vocdoniNetworks` **StringSlice**         one or more of the available vocdoni networks
- `--vocdoniPrivKey` **string**               hexString privKeys for vocdoni faucet accounts

## API

- Request (Vocdoni)

    `curl -X GET https://foo.bar/faucet/vocdoni/<network>/<from>`

    - `<network>` one or more of `[dev, stage, azeno]`
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
