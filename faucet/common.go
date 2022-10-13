package faucet

import (
	"errors"

	"go.vocdoni.io/dvote/crypto/ethereum"
)

// FaucetNetworks represents an integer pointing to
// the supported faucet networks
type FaucetNetworks int

const (
	// FaucetNetworksEthereum represents the Ethereum main network
	FaucetNetworksEthereum FaucetNetworks = iota
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
)

var (
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
)

var (
	// SupportedFaucetNetworksMap have all the networks the faucet supports
	SupportedFaucetNetworksMap = map[string]FaucetNetworks{
		"mainnet":      FaucetNetworksEthereum,
		"vocdoniDev":   FaucetNetworksVocdoniDev,
		"vocdoniStage": FaucetNetworksVocdoniStage,
		"vocdoniAzeno": FaucetNetworksVocdoniAzeno,
		"goerli":       FaucetNetworksGoerli,
		"sepolia":      FaucetNetworksSepolia,
		"matic":        FaucetNetworksMatic,
		"mumbai":       FaucetNetworksMumbai,
		"gnosisChain":  FaucetNetworksGnosisChain,
		"evmtest":      FaucetNetworksEVMTest,
	}
)

// Signer representes a signer
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

func (sc *sendConditions) basicBalanceCheck(balance uint64) bool {
	return balance < sc.Balance
}
