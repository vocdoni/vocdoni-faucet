package faucet

import (
	"errors"

	"go.vocdoni.io/dvote/crypto/ethereum"
)

// FaucetNetworks represents an integer pointing to
// the supported faucet networks
type FaucetNetworks int

const (
	FaucetNetworksUndefined FaucetNetworks = iota
	// FaucetNetworksEthereum represents the Ethereum main network
	FaucetNetworksEthereum
	// FaucetNetworksVocdoniDev represents the Vocdoni development network
	FaucetNetworksVocdoniDev
	// FaucetNetworksVocdoniStage represents the Vocdoni stagging network
	FaucetNetworksVocdoniStage
	// FaucetNetworksVocdoniAzeno represents the Vocdoni azeno network
	FaucetNetworksVocdoniAzeno
	// FaucetNetworksGoerli represents the Ethereum Goerli test network
	FaucetNetworksGoerli
	// FaucetNetworksSepolia represents the Ethereum Sepolia test network
	FaucetNetworksSepolia
	// FaucetNetworksMatic represents the Polygon main network
	FaucetNetworksMatic
	// FaucetNetworksMumbai represents the Polygon test network
	FaucetNetworksMumbai
	// FaucetNetworksGnosisChain represents the Gnosis main network
	FaucetNetworksGnosisChain
	// FaucetNetworksEVMTest represents the simulated local evm network
	FaucetNetworksEVMTest

	FaucetNetworksVocdoniLTS
	FaucetNetworksVocdoniProd
)

var (
	MAXUINT64 = uint64(9223372036854775807)
	// ErrInvalidEndpoint error wrapping invalid endpoint errors
	ErrInvalidEndpoint error = errors.New("invalid endpoint")
	// ErrInvalidAmount error wrapping invalid amount errors
	ErrInvalidAmount error = errors.New("invalid amount")
	// ErrInvalidNetwork error wrapping invalid network errors
	ErrInvalidNetwork error = errors.New("invalid network")
	// ErrInvalidTimeout error wrapping invalid timeout errors
	ErrInvalidTimeout error = errors.New("invalid timeout")
	// ErrInvalidSigner error wrapping invalid signer errors
	ErrInvalidSigner error = errors.New("invalid signer")

	// EVMSupportedFaucetNetworksMap have all the networks the faucet supports
	EVMSupportedFaucetNetworksMap = map[string]FaucetNetworks{
		"undefined":   FaucetNetworksUndefined,
		"mainnet":     FaucetNetworksEthereum,
		"goerli":      FaucetNetworksGoerli,
		"sepolia":     FaucetNetworksSepolia,
		"matic":       FaucetNetworksMatic,
		"mumbai":      FaucetNetworksMumbai,
		"gnosisChain": FaucetNetworksGnosisChain,
		"evmtest":     FaucetNetworksEVMTest,
	}

	// VocdoniSupportedFaucetNetworksMap have all the networks the faucet supports
	VocdoniSupportedFaucetNetworksMap = map[string]FaucetNetworks{
		"undefined": FaucetNetworksUndefined,
		"dev":       FaucetNetworksVocdoniDev,
		"stage":     FaucetNetworksVocdoniStage,
		"azeno":     FaucetNetworksVocdoniAzeno,
		"lts":       FaucetNetworksVocdoniLTS,
		"prod":      FaucetNetworksVocdoniLTS,
	}
)

// Signer represents a signer
type Signer struct {
	// SignKeys ECDSA keypair
	SignKeys *ethereum.SignKeys
	// Taken semaphore
	Taken chan bool
}

type sendConditions struct {
	// Balance balance threshold
	Balance uint64
	// Challenge true if challenge enabled
	Challenge bool
}

func (sc *sendConditions) balanceCheck(balance uint64) bool {
	return balance < sc.Balance
}
