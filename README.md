# vocdoni-faucet

[![GoDoc](https://godoc.org/go.vocdoni.io/vocdoni-faucet?status.svg)](https://godoc.org/go.vocdoni.io/vocdoni-faucet)
[![Go Report Card](https://goreportcard.com/badge/go.vocdoni.io/vocdoni-faucet)](https://goreportcard.com/report/go.vocdoni.io/vocdoni-faucet)

[![Join Discord](https://img.shields.io/badge/discord-join%20chat-blue.svg)](https://discord.gg/4hKeArDaU2)
[![Twitter Follow](https://img.shields.io/twitter/follow/vocdoni.svg?style=social&label=Follow)](https://twitter.com/vocdoni)

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

### Docker

You can run go-vocdoni-faucet as a standalone container with a docker script (configuration options can be changed in file `config/env`):

```
dockerfiles/faucet.sh
```

All data will be stored in the shared volume `run`.



[![Contributor Covenant](https://img.shields.io/badge/Contributor%20Covenant-v1.4%20adopted-ff69b4.svg)](code-of-conduct.md) [![License: AGPL v3](https://img.shields.io/badge/License-AGPL%20v3-blue.svg)](https://www.gnu.org/licenses/agpl-3.0)

## Usage

`go run ./cmd/main.go`

Options:
- `--apiListenHost` **string**         API endpoint listen address (default "0.0.0.0")
- `--apiListenPort` **int**            API endpoint http port (default 8000)
- `--apiRoute` **string**              dvote API route (default "/")
- `--apiSSLDomain` **string**          enable TLS secure API domain with LetsEncrypt auto-generated certificate
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

- Request

    `curl -X POST https://foo.bar/faucet`

- Request body (Vocdoni)

    ```json
    {
        "network": "vocdoni",
        "from": "0x0000000000000000000000000000000000000000",
    }
    ```
- Response (Vocdoni)

    HTTP 200

    ```json
    {
        "faucetPackage":{
            "payload": {
                "identifier": 7499402340572247695,
                "to": "0x0000000000000000000000000000000000000000",
                "amount": 100
            },
            "signature": "0x123"
        }
    }
    ```

    HTTP 400

    ```json
    {
        "error": "Message goes here"
    }
    ```

- Request body (EVM)

    ```json
    {
        "network": "evm",
        "from": "0x0000000000000000000000000000000000000000",
    }
    ```

- Response (EVM)

    HTTP 200

    ```json
    {
        "txHash": "0x123"
    }
    ```

    HTTP 400

    ```json
    {
        "error": "Message goes here"
    }
    ```
