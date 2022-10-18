package faucet

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	goethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	evmcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	evmtypes "github.com/ethereum/go-ethereum/core/types"
	evmClient "github.com/ethereum/go-ethereum/ethclient"
	"go.vocdoni.io/dvote/crypto/ethereum"
	"go.vocdoni.io/dvote/log"
	"go.vocdoni.io/vocdoni-faucet/config"
)

// EVM contains all components required for the EVM faucet
type EVM struct {
	// network one of the available EVM networks
	network string
	// chainID chainId/networkId of the network
	chainID int
	// amount of tokens to be transferred
	amount uint64
	// endpoints to connect with
	endpoints []string
	// client client instance connected to an endpoint
	client *evmClient.Client
	// signers pool of signers
	signers []*Signer
	// timeout timeout for EVM network operations
	timeout time.Duration
	// sendConditions conditions to meet before sending faucet tokens
	sendConditions *sendConditions
	lock           sync.RWMutex

	// for testing purposes
	forTest     bool
	testBackend *evmTestBackend
}

// NewEVM returns an EVM instance
func NewEVM() *EVM {
	return &EVM{}
}

// Amount returns the amount for the faucet
func (e *EVM) Amout() uint64 {
	e.lock.RLock()
	defer e.lock.RUnlock()
	return e.amount
}

// Signers returns the signers of the faucet
func (e *EVM) Signers() []*Signer {
	e.lock.RLock()
	defer e.lock.RUnlock()
	return e.signers
}

// Network returns the faucet EVM network
func (e *EVM) Network() string {
	e.lock.RLock()
	defer e.lock.RUnlock()
	return e.network
}

func (e *EVM) setSendConditions(balance uint64, challenge bool) {
	e.lock.Lock()
	defer e.lock.Unlock()
	e.sendConditions = &sendConditions{
		Balance:   balance,
		Challenge: challenge,
	}
}

// SetAmount sets the amount for the faucet
func (e *EVM) SetAmount(amount uint64) error {
	e.lock.Lock()
	defer e.lock.Unlock()
	if amount == 0 && amount == e.amount {
		return ErrInvalidAmount
	}
	e.amount = amount
	return nil
}

// SetEndpoints appends endpoints to the existing ones
func (e *EVM) SetEndpoints(endpoints []string) error {
	if len(endpoints) == 0 {
		return ErrInvalidEndpoint
	}
	e.lock.Lock()
	defer e.lock.Unlock()
	e.endpoints = make([]string, 0)
	for _, endpoint := range endpoints {
		if len(endpoint) == 0 {
			return ErrInvalidEndpoint
		}
		e.endpoints = append(e.endpoints, endpoint)
	}
	return nil
}

// SetSigners append new signers to the existing ones
func (e *EVM) SetSigners(signersPrivKeys []string) error {
	if len(signersPrivKeys) == 0 {
		return ErrInvalidSigner
	}
	e.lock.Lock()
	defer e.lock.Unlock()
	signers := make([]*Signer, 0)
	for _, key := range signersPrivKeys {
		s := new(ethereum.SignKeys)
		if err := s.AddHexKey(key); err != nil {
			return fmt.Errorf("cannot import key: %w", err)
		}
		signers = append(signers, &Signer{SignKeys: s, Taken: make(chan bool, 1)})
	}
	e.signers = signers
	return nil
}

// Init creates a new EVM faucet object initialized with the given config
func (e *EVM) Init(ctx context.Context, evmConfig *config.FaucetConfig) error {
	// get chain specs
	chainSpecs, err := EVMSpecsFor(evmConfig.EVMNetwork)
	if err != nil {
		return err
	}
	e.network = chainSpecs.Name
	e.chainID = chainSpecs.NetworkID

	// check endpoints
	if err := e.SetEndpoints(evmConfig.EVMEndpoints); err != nil {
		return fmt.Errorf("cannot set endpoints: %w", err)
	}

	// set amout to transfer
	if err := e.SetAmount(evmConfig.EVMAmount); err != nil {
		return ErrInvalidAmount
	}

	// set signers
	if err := e.SetSigners(evmConfig.EVMPrivKeys); err != nil {
		return ErrInvalidSigner
	}

	// set default timeout for endpoint calls
	if evmConfig.EVMTimeout == 0 {
		return ErrInvalidTimeout
	}
	e.timeout = evmConfig.EVMTimeout

	// set send conditions
	e.setSendConditions(evmConfig.EVMSendConditions.Balance, evmConfig.EVMSendConditions.Challenge)

	return nil
}

// NewClient returns a working ethereum client connected to one of the faucet provided endpoints,
// returns error any endpoint works as expected
func (e *EVM) NewClient(ctx context.Context) error {
	e.client = &evmClient.Client{}
	var err error
	for _, endpoint := range e.endpoints {
		tctx, cancel := context.WithTimeout(ctx, e.timeout)
		defer cancel()
		e.client, err = evmClient.DialContext(tctx, endpoint)
		if err != nil {
			log.Warnf("cannot connect to %s with error %s", endpoint, err)
			continue
		}
		chainID, err := e.ClientChainID(ctx)
		if err != nil {
			log.Warnf("cannot get info from endpoint %s with err %s", endpoint, err)
			continue
		}
		// check network
		if chainID.Int64() != int64(e.chainID) {
			log.Warnf("got networkID %s but %s is expected, skipping endpoint", chainID.String(), e.network)
			continue
		}
		return nil
	}
	return ErrInvalidEndpoint
}

// ClientChainID returns the chainID that the client reports
// from the connected node
func (e *EVM) ClientBalanceAt(ctx context.Context,
	address evmcommon.Address,
	blockNumber *big.Int,
) (*big.Int, error) {
	if e == nil {
		if err := e.NewClient(ctx); err != nil {
			return nil, err
		}
	}
	tctx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()
	return e.balanceAt(tctx, address, blockNumber)
}

// ClientChainID returns the chainID that the client reports
// from the connected node
func (e *EVM) ClientChainID(ctx context.Context) (*big.Int, error) {
	if e.forTest {
		return e.testBackend.Backend.Blockchain().Config().ChainID, nil
	}
	if e.client == nil {
		if err := e.NewClient(ctx); err != nil {
			return nil, err
		}
	}
	tctx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()
	return e.client.ChainID(tctx)
}

func (e *EVM) checkTxStatus(ctx context.Context, txHash *evmcommon.Hash) (uint64, error) {
	var receipt *evmtypes.Receipt
	var err error
	if e.forTest {
		receipt, err = e.testBackend.Backend.TransactionReceipt(ctx, *txHash)
	} else {
		if e.client == nil {
			if err := e.NewClient(ctx); err != nil {
				return 0, err
			}
		}
		tctx, cancel := context.WithTimeout(ctx, e.timeout)
		defer cancel()
		receipt, err = e.client.TransactionReceipt(tctx, *txHash)
	}
	if err != nil {
		return 0, err
	}
	if receipt == nil {
		return 0, fmt.Errorf("cannot get staus, nil receipt")
	}
	return receipt.Status, nil
}

// sendTokens send tokens and returns the hash of the tx
func (e *EVM) sendTokens(ctx context.Context,
	to evmcommon.Address,
	signerIndex int,
) (*evmcommon.Hash, error) {
	// get nonce for the signer
	tctx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()
	var nonce uint64
	var err error
	var gasPrice, maxPriorityFeePerGas *big.Int
	if e.forTest {
		nonce, err = e.testBackend.Backend.PendingNonceAt(tctx, e.signers[signerIndex].SignKeys.Address())
		if err != nil {
			return nil, fmt.Errorf("error creating tx: %s", err)
		}
		gasPrice, err = e.testBackend.Backend.SuggestGasPrice(tctx)
		if err != nil {
			return nil, fmt.Errorf("error creating tx: %s", err)
		}
		maxPriorityFeePerGas, err = e.testBackend.Backend.SuggestGasTipCap(tctx)
		if err != nil {
			return nil, fmt.Errorf("error creating tx: %s", err)
		}
	} else {
		if e.client == nil {
			if err := e.NewClient(ctx); err != nil {
				return nil, err
			}
		}
		nonce, err = e.client.PendingNonceAt(tctx, e.signers[signerIndex].SignKeys.Address())
		if err != nil {
			return nil, fmt.Errorf("error creating tx: %s", err)
		}
		gasPrice, err = e.client.SuggestGasPrice(tctx)
		if err != nil {
			return nil, fmt.Errorf("error creating tx: %s", err)
		}
		maxPriorityFeePerGas, err = e.client.SuggestGasTipCap(tctx)
		if err != nil {
			return nil, fmt.Errorf("error creating tx: %s", err)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("cannot get signer account nonce: %s", err)
	}
	// create tx
	tx := evmtypes.NewTx(&evmtypes.DynamicFeeTx{
		ChainID:   big.NewInt(int64(e.chainID)),
		Nonce:     nonce,
		GasFeeCap: gasPrice,
		GasTipCap: maxPriorityFeePerGas,
		Gas:       uint64(21000), // enough for standard eth transfers
		To:        &to,
		Value:     big.NewInt(int64(e.amount)), // in wei
	})
	// sign tx
	signedTx, err := evmtypes.SignTx(tx,
		evmtypes.NewLondonSigner(big.NewInt(int64(e.chainID))),
		&e.signers[signerIndex].SignKeys.Private,
	)
	if err != nil {
		return nil, fmt.Errorf("cannot sign transaction: %s", err)
	}
	// send tx
	tctx2, cancel2 := context.WithTimeout(ctx, e.timeout)
	defer cancel2()
	if e.forTest {
		err = e.testBackend.Backend.SendTransaction(tctx2, signedTx)
	} else {
		err = e.client.SendTransaction(tctx2, signedTx)
	}

	if err != nil {
		return nil, fmt.Errorf("cannot send signed tx: %s", err)
	}
	log.Infof("sending %d tokens to newly created entity %s from signer: %s. TxHash: %s and Nonce: %d",
		e.amount,
		to.String(),
		e.signers[signerIndex].SignKeys.AddressString(),
		signedTx.Hash().Hex(),
		signedTx.Nonce(),
	)
	nHash := new(evmcommon.Hash)
	*nHash = signedTx.Hash()
	return nHash, nil
}

// SendTokens sends an amount to an address if the address meets the send conditions
func (e *EVM) SendTokens(ctx context.Context, to evmcommon.Address) (*evmcommon.Hash, error) {
	if e.client == nil && !e.forTest {
		if err := e.NewClient(ctx); err != nil {
			return nil, err
		}
	}

	// check to address meet sendConditions
	tctx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()
	toBalance, err := e.balanceAt(tctx, to, nil) // nil means latest block
	if err != nil {
		return nil, fmt.Errorf("cannot check entity balance")
	}
	if !e.sendConditions.basicBalanceCheck(toBalance.Uint64()) {
		return nil, fmt.Errorf("%s has already a balance of: %d, greater than the sendConditions",
			to.String(),
			toBalance.Int64(),
		)
	}

	var finished bool
	var txHash *evmcommon.Hash
	var nonce uint64
	for {
		// run until signer available
		for signerIndex, signer := range e.signers {
			select {
			case signer.Taken <- true:
			default:
				// if signer is waiting for a tx select the next one
				log.Debugf("signer %s has a pending tx",
					signer.SignKeys.AddressString())
				continue
			}
			// send tokens
			log.Debugf("using signer %s", signer.SignKeys.AddressString())
			tctx2, cancel2 := context.WithTimeout(ctx, e.timeout)
			defer cancel2()
			txHash, err = e.sendTokens(tctx2, to, signerIndex)
			if err != nil {
				log.Warnf("cannot send tx %s", txHash.Hex())
				<-signer.Taken
				finished = true
				break
			}
			// add pending tx
			log.Infof("signer %s tx: %s with nonce: %d successfully sent",
				signer.SignKeys.Address().Hex(),
				txHash.String(),
				nonce,
			)
			tctx3, cancel3 := context.WithTimeout(ctx, e.timeout)
			defer cancel3()
			go e.waitForTx(tctx3, txHash, signerIndex)
			finished = true
			break
		}
		// wait for signers
		if finished {
			break
		}
		time.Sleep(time.Second * 5)
	}
	return txHash, nil
}

func (e *EVM) waitForTx(ctx context.Context, txHash *evmcommon.Hash, signerIndex int) {
	// wait until tx status is available, means tx is already mined
	for {
		status, err := e.checkTxStatus(ctx, txHash)
		if err != nil {
			if err == goethereum.NotFound {
				// wait and check again
				time.Sleep(time.Second * 10)
				continue
			}
			log.Warnf("cannot checktx hash %s status with err: %s", txHash.Hex(), err)
			break
		}
		log.Debugf("tx %s status is: %d", txHash.Hex(), status)
		if status == 0 {
			log.Warnf("tx %s failed", txHash.Hex())
			break
		}
		log.Infof("tx %s mined", txHash.Hex())
		break
	}
	e.lock.Lock()
	defer e.lock.Unlock()
	<-e.signers[signerIndex].Taken
}

func (e *EVM) balanceAt(ctx context.Context,
	address evmcommon.Address,
	blockNumber *big.Int,
) (*big.Int, error) {
	tctx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()
	if e.forTest {
		return e.testBackend.Backend.BalanceAt(tctx, address, blockNumber) // nil means latest block
	}
	return e.client.BalanceAt(tctx, address, blockNumber) // nil means latest block
}

// FOR TESTING PURPOSES

// InitForTest inits an EVM instance with a simulated evm backend
func (e *EVM) InitForTest(ctx context.Context, evmConfig *config.FaucetConfig) error {
	if err := e.Init(ctx, evmConfig); err != nil {
		return err
	}
	e.testBackend = &evmTestBackend{
		PrivKey: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
	}
	if err := e.testBackend.new(); err != nil {
		return err
	}
	e.forTest = true
	return nil
}

// TestBackend returns the simulated evm backend
func (e *EVM) TestBackend() *evmTestBackend {
	return e.testBackend
}

type evmTestBackend struct {
	PrivKey string
	Backend *backends.SimulatedBackend
}

func (eb *evmTestBackend) new() error {
	signKey := ethereum.NewSignKeys()
	if err := signKey.AddHexKey(eb.PrivKey); err != nil {
		// ignore error and generate a random one
		fmt.Printf("private key not found or err %s, generating a random one", err)
		if err := signKey.Generate(); err != nil {
			return err
		}
	}
	balance := new(big.Int)
	balance.SetString("10000000000000000000", 10) // 10 eth in wei
	genesisAlloc := map[evmcommon.Address]core.GenesisAccount{
		signKey.Address(): {
			Balance: balance,
		},
	}
	blockGasLimit := uint64(4712388)
	eb.Backend = backends.NewSimulatedBackend(genesisAlloc, blockGasLimit)
	eb.Commit()
	return nil
}

// Commit saves the simulated backend state (new block)
func (eb *evmTestBackend) Commit() {
	eb.Backend.Commit()
}
