package api

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"go.vocdoni.io/dvote/httprouter"
	"go.vocdoni.io/dvote/httprouter/bearerstdapi"
	"go.vocdoni.io/dvote/types"
	"go.vocdoni.io/dvote/util"
	"go.vocdoni.io/vocdoni-faucet/faucet"
	"google.golang.org/protobuf/proto"
)

const (
	EVM     = "evm"
	Vocdoni = "vocdoni"
	// Maximum number of requests a whitelisted caller can do
	MaxRequest = 10000
)

var (
	ErrInvalidToken       = errors.New("invalid token")
	ErrInvalidFromAddress = errors.New("invalid from address")
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
	// Amount transferred
	Amount uint64 `json:"amount"`
	// FaucetPackage is the Vocdoni faucet package
	FaucetPackage []byte `json:"faucetPackage,omitempty"`
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
func (a *API) Init(router *httprouter.HTTProuter,
	baseRoute,
	whitelist string,
	vfaucet *faucet.Vocdoni,
	efaucet *faucet.EVM,
) error {
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
	// add whitelisted bearer tokens
	bearerWhitelist := strings.Split(whitelist, ",")
	for _, token := range bearerWhitelist {
		a.api.AddAuthToken(token, int64(MaxRequest))
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
		"/evm/{network}/{from}",
		"GET",
		bearerstdapi.MethodAccessTypePrivate,
		a.faucetHandler,
	); err != nil {
		return err
	}
	if err := a.api.RegisterMethod(
		"/vocdoni/{network}/{from}",
		"GET",
		bearerstdapi.MethodAccessTypePrivate,
		a.faucetHandler,
	); err != nil {
		return err
	}
	return nil
}

// attach takes a list of modules which are used
// by the handlers in order to interact with the system.
// Attach must be called before enableHandlers.
func (a *API) attach(vocdoniFaucet *faucet.Vocdoni, EVMFaucet *faucet.EVM) {
	a.vocdoniFaucet = vocdoniFaucet
	a.evmFaucet = EVMFaucet
}

func (a *API) networkParse(network, origin string) faucet.FaucetNetworks {
	switch origin {
	case EVM:
		return faucet.EVMSupportedFaucetNetworksMap[network]
	case Vocdoni:
		return faucet.VocdoniSupportedFaucetNetworksMap[network]
	default:
		return faucet.FaucetNetworksUndefined
	}
}

func (a *API) fromParse(from string) (*common.Address, error) {
	from = util.TrimHex(from)
	fromAddrBytes, err := hex.DecodeString(from)
	if err != nil {
		return nil, ErrInvalidFromAddress
	}
	fromAddr := common.BytesToAddress(fromAddrBytes)
	if fromAddr == types.EthereumZeroAddress {
		return nil, ErrInvalidFromAddress
	}
	return &fromAddr, err
}

func (a *API) faucetHandler(msg *bearerstdapi.BearerStandardAPIdata,
	ctx *httprouter.HTTPContext,
) error {
	// get auth token
	token, err := uuid.Parse(msg.AuthToken)
	if err != nil {
		return err
	}
	// authorize
	if a.api.GetAuthTokens(token.String()) == 0 {
		return ErrInvalidToken
	}
	// get network url param
	origin := strings.Split(ctx.Request.URL.Path, "/")
	network := a.networkParse(ctx.URLParam("network"), origin[1])
	// get from url param
	from, err := a.fromParse(ctx.URLParam("from"))
	if err != nil {
		return err
	}
	// handle
	switch network {
	case faucet.FaucetNetworksUndefined:
		return fmt.Errorf("%s", "unsupported network")
	case faucet.FaucetNetworksVocdoniDev,
		faucet.FaucetNetworksVocdoniStage,
		faucet.FaucetNetworksVocdoniAzeno:
		return a.vocdoniFaucetHandler(ctx, network, *from)
	case faucet.FaucetNetworksEthereum,
		faucet.FaucetNetworksGoerli,
		faucet.FaucetNetworksSepolia,
		faucet.FaucetNetworksMatic,
		faucet.FaucetNetworksMumbai,
		faucet.FaucetNetworksGnosisChain,
		faucet.FaucetNetworksEVMTest:
		return a.evmFaucetHandler(ctx, network, *from)
	}
	return fmt.Errorf("cannot handle request")
}

// request evm funds to the faucet
func (a *API) evmFaucetHandler(ctx *httprouter.HTTPContext,
	network faucet.FaucetNetworks,
	from common.Address,
) error {
	if faucet.EVMSupportedFaucetNetworksMap[a.evmFaucet.Network()] != network {
		return fmt.Errorf("unavailable network")
	}
	txHash, err := a.evmFaucet.SendTokens(context.Background(), from)
	if err != nil {
		return fmt.Errorf("error sending evm tokens: %s", err)
	}
	resp := &FaucetResponse{
		TxHash: types.HexBytes(txHash.Bytes()),
		Amount: a.evmFaucet.Amout(),
	}
	msg, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	return ctx.Send(msg, bearerstdapi.HTTPstatusCodeOK)
}

// request vocdoni funds to the faucet
func (a *API) vocdoniFaucetHandler(ctx *httprouter.HTTPContext,
	network faucet.FaucetNetworks,
	from common.Address,
) error {
	if faucet.VocdoniSupportedFaucetNetworksMap[a.vocdoniFaucet.Network()] != network {
		return fmt.Errorf("unavailable network")
	}
	faucetPackage, err := a.vocdoniFaucet.GenerateFaucetPackage(from)
	if err != nil {
		return fmt.Errorf("error sending evm tokens: %s", err)
	}
	faucetPackageBytes, err := proto.Marshal(faucetPackage)
	if err != nil {
		return err
	}
	resp := &FaucetResponse{
		FaucetPackage: faucetPackageBytes,
		Amount:        a.vocdoniFaucet.Amount(),
	}
	msg, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	return ctx.Send(msg, bearerstdapi.HTTPstatusCodeOK)
}
