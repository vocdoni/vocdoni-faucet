package faucet_test

import (
	"context"
	"testing"

	qt "github.com/frankban/quicktest"
	"go.vocdoni.io/dvote/crypto/ethereum"
	"go.vocdoni.io/vocdoni-faucet/config"
	"go.vocdoni.io/vocdoni-faucet/faucet"
	"google.golang.org/protobuf/proto"
)

var (
	eConfig = &config.FaucetConfig{
		Amount:       100,
		EVMNetwork:   "mainnet",
		EVMPrivKeys:  []string{"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
		EVMEndpoints: []string{"http://localhost:8545"},
		EVMTimeout:   10,
		SendConditions: config.SendConditionsConfig{
			Balance:   100,
			Challenge: false,
		},
	}
	vConfig = &config.FaucetConfig{
		Amount:         100,
		VocdoniNetwork: "vocdoniDev",
		VocdoniPrivKey: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		SendConditions: config.SendConditionsConfig{
			Balance:   100,
			Challenge: false,
		},
	}
)

func TestNewEVM(t *testing.T) {
	e := faucet.NewEVM()
	// should not accept an invalid network name
	eConfig1 := *eConfig
	eConfig1.EVMNetwork = "invalid"
	qt.Assert(t, e.Init(context.Background(), &eConfig1), qt.ErrorIs, faucet.ErrInvalidNetwork)
	// should not accept invalid endpoints
	eConfig1.EVMNetwork = "mainnet"
	eConfig1.EVMEndpoints = []string{}
	qt.Assert(t, e.Init(context.Background(), &eConfig1), qt.ErrorIs, faucet.ErrInvalidEndpoint)
	// should not accept an invalid amout
	eConfig1.EVMEndpoints = []string{"http://localhost:8545"}
	eConfig1.Amount = 0
	qt.Assert(t, e.Init(context.Background(), &eConfig1), qt.ErrorIs, faucet.ErrInvalidAmount)
	// should not accept invalid priv keys
	eConfig1.Amount = 100
	eConfig1.EVMPrivKeys = []string{"0x0"}
	qt.Assert(t, e.Init(context.Background(), &eConfig1), qt.ErrorMatches, "invalid signer")
	// should not accept an invalid timeout
	eConfig1.EVMPrivKeys = []string{"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"}
	eConfig1.EVMTimeout = 0
	qt.Assert(t, e.Init(context.Background(), &eConfig1), qt.ErrorIs, faucet.ErrInvalidTimeout)

}

func TestNewClient(t *testing.T) {
	e := faucet.NewEVM()
	qt.Assert(t, e.Init(context.Background(), eConfig), qt.IsNil)
	// should not work
	e.SetEndpoints([]string{})
	qt.Assert(t, e.NewClient(context.Background()), qt.IsNotNil)
}

func TestSendTokens(t *testing.T) {
	e := faucet.NewEVM()
	eConfig.EVMNetwork = "evmtest"
	qt.Assert(t, e.InitForTest(context.Background(), eConfig), qt.IsNil)
	toAddr := &ethereum.SignKeys{}
	qt.Assert(t, toAddr.Generate(), qt.IsNil)

	// should work
	_, err := e.SendTokens(context.Background(), toAddr.Address())
	qt.Assert(t, err, qt.IsNil)
	e.TestBackend().Backend.Commit() // save ethereum state
	newBalance, err := e.ClientBalanceAt(context.Background(), toAddr.Address(), nil)
	qt.Assert(t, err, qt.IsNil)
	qt.Assert(t, newBalance.Int64(), qt.DeepEquals, int64(100))

	// should not work if sendConditions are not met
	_, err = e.SendTokens(context.Background(), toAddr.Address())
	qt.Assert(t, err, qt.IsNotNil)
	e.TestBackend().Backend.Commit() // save ethereum state
	newBalance, err = e.ClientBalanceAt(context.Background(), toAddr.Address(), nil)
	qt.Assert(t, err, qt.IsNil)
	qt.Assert(t, newBalance.Int64(), qt.DeepEquals, int64(100))

	/*  example added for development purposes

	// should hang forever if all signers are taken
	// waiting to have a signer available for
	// sending the transaction
	toAddr2 := &ethereum.SignKeys{}
	qt.Assert(t, toAddr2.Generate(), qt.IsNil)
	toAddr3 := &ethereum.SignKeys{}
	qt.Assert(t, toAddr3.Generate(), qt.IsNil)
	_, err = e.SendTokens(context.Background(), toAddr2.Address(), true)
	qt.Assert(t, err, qt.IsNil)

	// in this case the sendTokens loop will hang forever because
	// no signer is available since the previous transaction
	// using the same signer is not yet mined
	// notice that no commmit is executed here, if uncommented
	// this test must timeout

	_, err = e.SendTokens(context.Background(), toAddr3.Address(), true)
	qt.Assert(t, err, qt.IsNotNil)

	*/

	// add another signer and check that if signers[0] is taken
	// signers[1] is used
	// this case is the opposite from the case commented above,
	// as a new signer is added two send tokens can be executed
	// without waiting the tx to be mined because two diferent
	// signers are used.

	newSigner := &ethereum.SignKeys{}
	qt.Assert(t, newSigner.AddHexKey(
		"f3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"), qt.IsNil)
	qt.Assert(t, e.SetSigners([]string{
		"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		"f3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
	}), qt.IsNil)
	// send tokens to new signer
	qt.Assert(t, e.SetAmount(1079853350110000), qt.IsNil)
	_, err = e.SendTokens(context.Background(), newSigner.Address())
	qt.Assert(t, e.SetAmount(100), qt.IsNil)
	qt.Assert(t, err, qt.IsNil)
	e.TestBackend().Backend.Commit() // save ethereum state
	// expected to use two different signers
	toAddr2 := &ethereum.SignKeys{}
	qt.Assert(t, toAddr2.Generate(), qt.IsNil)
	toAddr3 := &ethereum.SignKeys{}
	qt.Assert(t, toAddr3.Generate(), qt.IsNil)
	_, err = e.SendTokens(context.Background(), toAddr2.Address())
	qt.Assert(t, err, qt.IsNil)
	_, err = e.SendTokens(context.Background(), toAddr3.Address())
	qt.Assert(t, err, qt.IsNil)
	e.TestBackend().Backend.Commit() // save ethereum state
	newBalance, err = e.ClientBalanceAt(context.Background(), toAddr2.Address(), nil)
	qt.Assert(t, err, qt.IsNil)
	qt.Assert(t, newBalance.Int64(), qt.DeepEquals, int64(100))
	newBalance, err = e.ClientBalanceAt(context.Background(), toAddr3.Address(), nil)
	qt.Assert(t, err, qt.IsNil)
	qt.Assert(t, newBalance.Int64(), qt.DeepEquals, int64(100))

}

func TestNewVocdoni(t *testing.T) {
	v := faucet.NewVocdoni()
	// should not accept an invalid network name
	vConfig1 := *vConfig
	vConfig1.VocdoniNetwork = "invalid"
	qt.Assert(t, v.Init(context.Background(), &vConfig1), qt.ErrorIs, faucet.ErrInvalidNetwork)
	// should not accept an invalid amout
	vConfig1.VocdoniNetwork = "vocdoniDev"
	vConfig1.Amount = 0
	qt.Assert(t, v.Init(context.Background(), &vConfig1), qt.ErrorIs, faucet.ErrInvalidAmount)
	// should not accept an invalid priv key
	vConfig1.Amount = 100
	vConfig1.VocdoniPrivKey = "0x0"
	qt.Assert(t, v.Init(context.Background(), &vConfig1), qt.ErrorMatches, "cannot import key: invalid hex data for private key")
	// should work
	vConfig1.VocdoniPrivKey = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	qt.Assert(t, v.Init(context.Background(), &vConfig1), qt.IsNil)
}

func TestGenerateFaucet(t *testing.T) {
	v := &faucet.Vocdoni{}
	qt.Assert(t, v.Init(context.Background(), vConfig), qt.IsNil)
	toAddr := &ethereum.SignKeys{}
	qt.Assert(t, toAddr.Generate(), qt.IsNil)
	faucetPackage, err := v.GenerateFaucetPackage(toAddr.Address())
	qt.Assert(t, err, qt.IsNil)
	qt.Assert(t, faucetPackage.Payload.Amount, qt.DeepEquals, v.Amount())
	qt.Assert(t, faucetPackage.Payload.To, qt.DeepEquals, toAddr.Address().Bytes())
	protoBytes, err := proto.Marshal(faucetPackage.Payload)
	qt.Assert(t, err, qt.IsNil)
	fromAddr, err := ethereum.AddrFromSignature(protoBytes, faucetPackage.Signature)
	qt.Assert(t, err, qt.IsNil)
	qt.Assert(t, fromAddr, qt.DeepEquals, v.Signer().Address())
}
