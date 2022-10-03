package restAPI

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"go.vocdoni.io/dvote/db"
	"go.vocdoni.io/dvote/db/metadb"
	"go.vocdoni.io/dvote/httprouter"
	"go.vocdoni.io/dvote/httprouter/bearerstdapi"
	"go.vocdoni.io/dvote/metrics"
	"go.vocdoni.io/dvote/types"
	"go.vocdoni.io/vocdoni-faucet/faucet"
)

type FaucetNetworks int

const (
	VocdoniFaucetHandlerName                = "vocdoni"
	EVMFaucetHandlerName                    = "evm"
	MaxPageSize                             = 10
	FaucetNetworksEthereum   FaucetNetworks = iota
	FaucetNetworksVocdoniDev
	FaucetNetworksVocdoniAzeno
	FaucetNetworksGoerli
	FaucetNetworksSepolia
	FaucetNetworksMatic
	FaucetNetworksMumbai
	FaucetNetworksGnosisChain
)

var (
	SupportedFaucetNetworksMap = map[string]FaucetNetworks{
		"ethereum":     FaucetNetworksEthereum,
		"vocdoniDev":   FaucetNetworksVocdoniDev,
		"vocdoniAzeno": FaucetNetworksVocdoniAzeno,
		"goerli":       FaucetNetworksGoerli,
		"sepolia":      FaucetNetworksSepolia,
		"matic":        FaucetNetworksMatic,
		"mumbai":       FaucetNetworksMumbai,
		"gnosisChain":  FaucetNetworksGnosisChain,
	}
)

// RestAPI is the URL based REST API supporting bearer authentication.
type RestAPI struct {
	PrivateCalls uint64
	PublicCalls  uint64
	BaseRoute    string

	db            db.Database
	router        *httprouter.HTTProuter
	api           *bearerstdapi.BearerStandardAPI
	evmFaucet     *faucet.EVM
	vocdoniFaucet *faucet.Vocdoni
	//lint:ignore U1000 unused
	metricsagent *metrics.Agent
}

// NewRestAPI creates a new instance of the RestAPI.  Attach must be called next.
func NewRestAPI(router *httprouter.HTTProuter, baseRoute, dataDir string) (*RestAPI, error) {
	if router == nil {
		return nil, fmt.Errorf("httprouter is nil")
	}
	if len(baseRoute) == 0 || baseRoute[0] != '/' {
		return nil, fmt.Errorf("invalid base route (%s), it must start with /", baseRoute)
	}
	// remove trailing slash
	if len(baseRoute) > 1 {
		baseRoute = strings.TrimSuffix(baseRoute, "/")
	}
	restAPI := RestAPI{
		BaseRoute: baseRoute,
		router:    router,
	}
	var err error
	restAPI.api, err = bearerstdapi.NewBearerStandardAPI(router, baseRoute)
	if err != nil {
		return nil, err
	}
	// create local key value database
	restAPI.db, err = metadb.New(db.TypePebble, filepath.Join(dataDir, "restAPI"))
	if err != nil {
		return nil, err
	}
	// register endpoint
	if err := restAPI.api.RegisterMethod(
		"/faucet",
		"POST",
		bearerstdapi.MethodAccessTypePublic,
		restAPI.faucetHandler,
	); err != nil {
		return nil, err
	}

	return &restAPI, nil
}

// Attach takes a list of modules which are used by the handlers in order to interact with the system.
// Attach must be called before EnableHandlers.
func (u *RestAPI) Attach(vocdoniFaucet *faucet.Vocdoni, EVMFaucet *faucet.EVM, metricsAgent *metrics.Agent) {
	u.vocdoniFaucet = vocdoniFaucet
	u.evmFaucet = EVMFaucet
	u.metricsagent = metricsAgent
}

type FaucetRequest struct {
	Network string         `json:"network"`
	From    types.HexBytes `json:"from"`
}

// request funds to the faucet
func (u *RestAPI) faucetHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	req := &FaucetRequest{}
	if err := json.Unmarshal(msg.Data, req); err != nil {
		return err
	}

	if _, ok := SupportedFaucetNetworksMap[req.Network]; !ok {
		return fmt.Errorf("%s", "unsupported network")
	}

	switch SupportedFaucetNetworksMap[req.Network] {
	// executar evm
	case FaucetNetworksEthereum,
		FaucetNetworksGoerli,
		FaucetNetworksSepolia,
		FaucetNetworksMatic,
		FaucetNetworksMumbai,
		FaucetNetworksGnosisChain:
		txhash, err := u.evmFaucet.Send()
		if err != nil {
			return fmt.Errorf("error sending evm tokens: %s", err)
		}
		return ctx.Send(txhash, bearerstdapi.HTTPstatusCodeOK)
	case FaucetNetworksVocdoniDev, FaucetNetworksVocdoniAzeno:
		faucetPackage, err := u.vocdoniFaucet.Send()
		if err != nil {
			return fmt.Errorf("error sending evm tokens: %s", err)
		}
		return ctx.Send(faucetPackage, bearerstdapi.HTTPstatusCodeOK)
	// crear faucet pkg
	default:
		return fmt.Errorf("%s", "unsupported network")
	}

	return ctx.Send(data, bearerstdapi.HTTPstatusCodeOK)
}
