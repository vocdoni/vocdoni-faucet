package api_test

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"net/http"
	"net/url"
	"path"
	"testing"
	"time"

	evmcommon "github.com/ethereum/go-ethereum/common"
	qt "github.com/frankban/quicktest"
	"github.com/google/uuid"
	"go.vocdoni.io/dvote/crypto/ethereum"
	"go.vocdoni.io/dvote/httprouter"
	"go.vocdoni.io/dvote/log"
	"go.vocdoni.io/proto/build/go/models"
	faucetapi "go.vocdoni.io/vocdoni-faucet/api"
	"go.vocdoni.io/vocdoni-faucet/config"
	"go.vocdoni.io/vocdoni-faucet/faucet"
	"google.golang.org/protobuf/proto"
)

var (
	eConfig = &config.FaucetConfig{
		EVMAmount:    100,
		EVMNetwork:   "evmtest",
		EVMPrivKeys:  []string{"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
		EVMEndpoints: []string{"localhost:8545"},
		EVMTimeout:   10,
		EVMSendConditions: config.SendConditionsConfig{
			Balance:   100,
			Challenge: false,
		},
	}
	vConfig = &config.FaucetConfig{
		VocdoniAmount:  100,
		VocdoniNetwork: "dev",
		VocdoniPrivKey: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		VocdoniSendConditions: config.SendConditionsConfig{
			Balance:   100,
			Challenge: false,
		},
	}
	randomEVMAddress = evmcommon.HexToAddress("0xAAafD269cf7F6C7a7afa92A32127fbc72593638e")
)

func TestAPI(t *testing.T) {
	log.Init("debug", "stdout")

	// create vocdoni faucet
	v := faucet.NewVocdoni()
	qt.Assert(t, v.Init(context.Background(), vConfig), qt.IsNil)

	// create ethereum faucet
	e := faucet.NewEVM()
	qt.Assert(t, e.InitForTest(context.Background(), eConfig), qt.IsNil)

	router := httprouter.HTTProuter{}
	qt.Assert(t, router.Init("127.0.0.1", 0), qt.IsNil)
	addr, err := url.Parse("http://" + path.Join(router.Address().String(), "/faucet"))
	qt.Assert(t, err, qt.IsNil)

	t.Logf("address: %s", addr)

	api := faucetapi.NewAPI()
	// api whitelist
	token, err := uuid.NewUUID()
	qt.Assert(t, err, qt.IsNil)
	qt.Assert(t, api.Init(&router, "/faucet", token.String(), true, true, v, e), qt.IsNil)
	c := newTestHTTPclient(t, addr, &token)

	// create vocdoni request
	resp, code := c.request("GET", nil, "vocdoni", "dev", randomEVMAddress.Hex())
	qt.Assert(t, code, qt.Equals, 200)
	respData := &faucetapi.FaucetResponse{}
	qt.Assert(t, json.Unmarshal(resp, &respData), qt.IsNil)
	faucetPackageData := &models.FaucetPackage{}
	qt.Assert(t, proto.Unmarshal(respData.FaucetPackage, faucetPackageData), qt.IsNil)
	qt.Assert(t, faucetPackageData.Payload.Amount, qt.DeepEquals, uint64(100))
	qt.Assert(t,
		evmcommon.BytesToAddress(faucetPackageData.Payload.To),
		qt.DeepEquals,
		randomEVMAddress,
	)
	payloadBytes, err := proto.Marshal(faucetPackageData.Payload)
	qt.Assert(t, err, qt.IsNil)
	fromAddress, err := ethereum.AddrFromSignature(payloadBytes, faucetPackageData.Signature)
	qt.Assert(t, err, qt.IsNil)
	qt.Assert(t, fromAddress, qt.DeepEquals, v.Signer().Address())
	t.Logf("%s", fmt.Sprintf(
		`"response": {
			"code": %d,
			"data": {
				"faucetPackage": %s,
				"amount": %d
			}
		recovered from address of the faucet is %s`,
		code,
		hex.EncodeToString(respData.FaucetPackage),
		respData.Amount,
		fromAddress,
	))

	// create ethereum request
	resp, code = c.request("GET", nil, "evm", "evmtest", randomEVMAddress.String())
	// make the test blockchain to mine a block
	qt.Assert(t, code, qt.Equals, 200)
	respData = &faucetapi.FaucetResponse{}
	qt.Assert(t, json.Unmarshal(resp, &respData), qt.IsNil)
	t.Logf("%s", fmt.Sprintf(
		`"response": {
				"code": %d,
				"data": {
					"txHash": "%s",
					"amount": "%d"
				},
			}`,
		code,
		evmcommon.BytesToHash(respData.TxHash).Hex(),
		respData.Amount,
	))
	balance, err := e.ClientBalanceAt(context.Background(), randomEVMAddress, nil)
	qt.Assert(t, err, qt.IsNil)
	// 0 balance as no committed block
	qt.Assert(t, balance.Cmp(big.NewInt(int64(0))), qt.Equals, 0)
	e.TestBackend().Commit()
	balance, err = e.ClientBalanceAt(context.Background(), randomEVMAddress, nil)
	qt.Assert(t, err, qt.IsNil)
	// balance updated
	qt.Assert(t, balance.Cmp(big.NewInt(int64(100))), qt.Equals, 0)
}

type testHTTPclient struct {
	c     *http.Client
	token *uuid.UUID
	addr  *url.URL
	t     *testing.T
}

func (c *testHTTPclient) request(method string, body []byte, urlPath ...string) ([]byte, int) {
	u, err := url.Parse(c.addr.String())
	qt.Assert(c.t, err, qt.IsNil)
	u.Path = path.Join(u.Path, path.Join(urlPath...))
	headers := http.Header{}
	if c.token != nil {
		headers = http.Header{"Authorization": []string{"Bearer " + c.token.String()}}
	}
	resp, err := c.c.Do(&http.Request{
		Method: method,
		URL:    u,
		Header: headers,
		Body:   io.NopCloser(bytes.NewBuffer(body)),
	})
	qt.Assert(c.t, err, qt.IsNil)
	data, err := ioutil.ReadAll(resp.Body)
	qt.Assert(c.t, err, qt.IsNil)
	return data, resp.StatusCode
}

func newTestHTTPclient(t *testing.T, addr *url.URL, bearerToken *uuid.UUID) *testHTTPclient {
	tr := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    5 * time.Second,
		DisableCompression: false,
	}
	return &testHTTPclient{
		c:     &http.Client{Transport: tr, Timeout: time.Second * 8},
		token: bearerToken,
		addr:  addr,
		t:     t,
	}
}
