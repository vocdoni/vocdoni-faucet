package faucet

import (
	"context"
	"fmt"

	"go.vocdoni.io/dvote/crypto/ethereum"
	"go.vocdoni.io/vocdoni-faucet/config"
)

// Vocdoni contains all components required for the Vocdoni faucet
type Vocdoni struct {
	// network name of the network
	network,
	// networkID one of the Vocdoni networks
	networkID string
	// amount of tokens to include
	amount uint64
	// signer account that will be used for signing
	signer *ethereum.SignKeys
	// sendConditions conditions to meet before executing an action
	sendConditions *sendConditions
}

// NewVocdoni returns a new instance of a Vocdoni faucet
func NewVocdoni() *Vocdoni {
	return &Vocdoni{}
}

// Amount returns amount
func (v *Vocdoni) Amount() uint64 {
	return v.amount
}

// Signer returns the signer
func (v *Vocdoni) Signer() *ethereum.SignKeys {
	return v.signer
}

// Network returns the faucet vocdoni network
func (v *Vocdoni) Network() string {
	return v.network
}

func (v *Vocdoni) setAmount(amount uint64) error {
	if amount == 0 {
		return ErrInvalidAmount
	}
	v.amount = amount
	return nil
}

func (v *Vocdoni) setSendConditions(balance uint64, challenge bool) {
	v.sendConditions = &sendConditions{
		Balance:   balance,
		Challenge: challenge,
	}
}

// Init initializes a Vocdoni instance with the given config
func (v *Vocdoni) Init(ctx context.Context, vocdoniConfig *config.FaucetConfig) error {
	// get chain specs
	chainSpecs, err := vocdoniSpecsFor(vocdoniConfig.VocdoniNetwork)
	if err != nil {
		return err
	}
	v.network = chainSpecs.network
	v.networkID = chainSpecs.networkID

	// set amout to transfer
	if err := v.setAmount(vocdoniConfig.VocdoniAmount); err != nil {
		return err
	}

	// set signer
	v.signer = new(ethereum.SignKeys)
	if err := v.signer.AddHexKey(vocdoniConfig.VocdoniPrivKey); err != nil {
		return fmt.Errorf("cannot import key: %w", err)
	}

	// set send conditions
	v.setSendConditions(
		vocdoniConfig.VocdoniSendConditions.Balance,
		vocdoniConfig.VocdoniSendConditions.Challenge,
	)

	return nil
}

var (
	azeno = vocdoniSpecs{network: "azeno", networkID: "azeno"}
	stage = vocdoniSpecs{network: "stage", networkID: "stage"}
	dev   = vocdoniSpecs{network: "dev", networkID: "dev"}
)

// vocdoniSpecs defines a set of Vocdoni blockchain network specifications
type vocdoniSpecs struct {
	network   string
	networkID string
}

// VocdoniSpecsFor returns the specs for the given Vocdoni blockchain network name
func vocdoniSpecsFor(name string) (*vocdoniSpecs, error) {
	switch name {
	case "azeno":
		return &azeno, nil
	case "stage":
		return &stage, nil
	case "dev":
		return &dev, nil
	default:
		return nil, ErrInvalidNetwork
	}
}
