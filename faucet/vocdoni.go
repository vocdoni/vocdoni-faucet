package faucet

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"

	evmcommon "github.com/ethereum/go-ethereum/common"
	"go.vocdoni.io/dvote/crypto/ethereum"
	"go.vocdoni.io/proto/build/go/models"
	"go.vocdoni.io/vocdoni-faucet/config"
	"google.golang.org/protobuf/proto"
)

var (
	azeno = vocdoniSpecs{network: "azeno", networkID: "azeno"}
	stage = vocdoniSpecs{network: "stage", networkID: "stage"}
	dev   = vocdoniSpecs{network: "dev", networkID: "dev"}
	lts   = vocdoniSpecs{network: "lts", networkID: "lts"}
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
	case "lts":
		return &lts, nil
	default:
		return nil, ErrInvalidNetwork
	}
}

// Vocdoni contains all components required for the Vocdoni faucet
type Vocdoni struct {
	// network name of the network
	network,
	// networkID one of the Vocdoni networks
	networkID []string
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
func (v *Vocdoni) Network() []string {
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

	for _, network := range vocdoniConfig.VocdoniNetworks {
		chainSpecs, err := vocdoniSpecsFor(network)
		if err != nil {
			return err
		}
		v.network = append(v.network, chainSpecs.network)
		v.networkID = append(v.networkID, chainSpecs.networkID)

	}

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

// GenerateFaucetPackage generates a faucet package
func (v *Vocdoni) GenerateFaucetPackage(address evmcommon.Address) (*models.FaucetPackage, error) {
	identifier, err := rand.Int(rand.Reader, big.NewInt(int64(MAXUINT64)))
	if err != nil {
		return nil, fmt.Errorf("cannot generate faucet package identifier")
	}

	payload := &models.FaucetPayload{
		Identifier: identifier.Uint64(),
		To:         address.Bytes(),
		Amount:     v.amount,
	}
	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		return nil, err
	}
	payloadSignature, err := v.signer.SignEthereum(payloadBytes)
	if err != nil {
		return nil, err
	}
	return &models.FaucetPackage{
		Payload:   payloadBytes,
		Signature: payloadSignature,
	}, nil
}
