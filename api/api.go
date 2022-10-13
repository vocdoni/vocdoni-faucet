package api

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"go.vocdoni.io/dvote/httprouter"
	"go.vocdoni.io/dvote/httprouter/bearerstdapi"
	"go.vocdoni.io/dvote/log"
	"go.vocdoni.io/dvote/types"
	"go.vocdoni.io/proto/build/go/models"
	"go.vocdoni.io/vocdoni-faucet/faucet"
)

const (
	EVM     = "evm"
	Vocdoni = "vocdoni"
)

// FaucetRequestData represents the data of a faucet request
type FaucetRequestData struct {
	// Network represents one of the supported faucet networks (evm or vocdoni)
	Network string `json:"network"`
	// From represents the address for the faucet to send tokens
	From types.HexBytes `json:"from"`
}

// FaucetResponse represents the message on the response of a faucet request
type FaucetResponse struct {
	// FaucetPackage is the Vocdoni faucet package
	FaucetPackage *models.FaucetPackage `json:"faucetPackage,omitempty"`
	// TxHash is the EVM tx hash
	TxHash types.HexBytes `json:"txHash,omitempty"`
}

// API is the URL based API supporting bearer authentication.
type API struct {
	baseRoute string

	router        *httprouter.HTTProuter
	api           *bearerstdapi.BearerStandardAPI
	evmFaucet     *faucet.EVM
	vocdoniFaucet *faucet.Vocdoni
}

// NewAPI returns a new instance of the API
func NewAPI() *API {
	return &API{}
}

// Init initianizes an API instance
func (a *API) Init(router *httprouter.HTTProuter, baseRoute string, vfaucet *faucet.Vocdoni, efaucet *faucet.EVM) error {
	if router == nil {
		return fmt.Errorf("httprouter is nil")
	}
	a.router = router
	if len(baseRoute) == 0 || baseRoute[0] != '/' {
		return fmt.Errorf("invalid base route (%s), it must start with /", baseRoute)
	}
	// remove trailing slash
	if len(baseRoute) > 1 {
		baseRoute = strings.TrimSuffix(baseRoute, "/")
	}
	a.baseRoute = baseRoute
	// bearer token api
	var err error
	if a.api, err = bearerstdapi.NewBearerStandardAPI(a.router, a.baseRoute); err != nil {
		return err
	}
	// attach faucet modules
	a.attach(vfaucet, efaucet)
	// enable handlers
	if err := a.enableFaucetHandlers(); err != nil {
		return fmt.Errorf("cannot enable handlers %w", err)
	}
	return nil
}

func (a *API) enableFaucetHandlers() error {
	if err := a.api.RegisterMethod(
		"/faucet",
		"POST",
		bearerstdapi.MethodAccessTypePublic,
		a.faucetHandler,
	); err != nil {
		return err
	}
	return nil
}

// attach takes a list of modules which are used by the handlers in order to interact with the system.
// Attach must be called before enableHandlers.
func (a *API) attach(vocdoniFaucet *faucet.Vocdoni, EVMFaucet *faucet.EVM) {
	a.vocdoniFaucet = vocdoniFaucet
	a.evmFaucet = EVMFaucet
}

// request funds to the faucet
func (a *API) faucetHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	req := &FaucetRequestData{}
	if err := json.Unmarshal(msg.Data, req); err != nil {
		return err
	}
	fromAddress := common.BytesToAddress(req.From)
	log.Infof(`request: { "network": %s, "from": %s }`, req.Network, fromAddress)

	switch req.Network {
	// evm
	case EVM:
		data, err := a.evmFaucet.SendTokens(context.Background(), fromAddress)
		if err != nil {
			return fmt.Errorf("error sending evm tokens: %s", err)
		}
		resp := &FaucetResponse{
			TxHash: types.HexBytes(data.Bytes()),
		}
		msg, err := json.Marshal(resp)
		if err != nil {
			return err
		}
		return ctx.Send(msg, bearerstdapi.HTTPstatusCodeOK)
	// vocdoni
	case Vocdoni:
		faucetPackage, err := a.vocdoniFaucet.GenerateFaucetPackage(fromAddress)
		if err != nil {
			return fmt.Errorf("error sending evm tokens: %s", err)
		}
		resp := &FaucetResponse{
			FaucetPackage: faucetPackage,
		}
		msg, err := json.Marshal(resp)
		if err != nil {
			return err
		}
		return ctx.Send(msg, bearerstdapi.HTTPstatusCodeOK)
	default:
		return fmt.Errorf("%s", "unsupported network")
	}
}
