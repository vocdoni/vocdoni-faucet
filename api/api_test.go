package api_test

import (
	"bytes"
	"context"
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
	"go.vocdoni.io/dvote/types"
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
		SendConditions: config.SendConditionsConfig{
			Balance:   100,
			Challenge: false,
		},
	}
	vConfig = &config.FaucetConfig{
		EVMAmount:      100,
		VocdoniNetwork: "vocdoniDev",
		VocdoniPrivKey: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		SendConditions: config.SendConditionsConfig{
			Balance:   100,
			Challenge: false,
		},
	}
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
	router.Init("127.0.0.1", 0)
	addr, err := url.Parse("http://" + path.Join(router.Address().String(), "/faucet"))
	qt.Assert(t, err, qt.IsNil)

	t.Logf("address: %s", addr)

	api := faucetapi.NewAPI()
	qt.Assert(t, api.Init(&router, "/", v, e), qt.IsNil)

	c := newTestHTTPclient(t, addr, nil)

	// create vocdoni request
	body := &faucetapi.FaucetRequestData{
		Network: "vocdoni",
		From:    types.HexBytes(evmcommon.Address{}.Bytes()),
	}
	data, err := json.Marshal(body)
	qt.Assert(t, err, qt.IsNil)
	resp, code := c.request("POST", data)
	qt.Assert(t, code, qt.Equals, 200)
	respData := &faucetapi.FaucetResponse{}
	qt.Assert(t, json.Unmarshal(resp, &respData), qt.IsNil)
	qt.Assert(t, respData.FaucetPackage.Payload.Amount, qt.DeepEquals, uint64(100))
	qt.Assert(t, evmcommon.BytesToAddress(respData.FaucetPackage.Payload.To), qt.DeepEquals, evmcommon.BytesToAddress([]byte{}))
	payloadBytes, err := proto.Marshal(respData.FaucetPackage.Payload)
	qt.Assert(t, err, qt.IsNil)
	fromAddress, err := ethereum.AddrFromSignature(payloadBytes, respData.FaucetPackage.Signature)
	qt.Assert(t, err, qt.IsNil)
	qt.Assert(t, fromAddress, qt.DeepEquals, v.Signer().Address())
	t.Logf("%s", fmt.Sprintf(
		`"response": {
			"code": %d,
			"data": {
				"faucetPackage":{
					"payload": {
						"identifier": %d,
						"to": %s,
						"amount": %d
					},
				"signature": %s}
				}
			},
		recovered from address of the faucet is %s`,
		code,
		respData.FaucetPackage.Payload.Identifier,
		evmcommon.BytesToAddress(respData.FaucetPackage.Payload.To),
		respData.FaucetPackage.Payload.Amount,
		evmcommon.Bytes2Hex(respData.FaucetPackage.Signature),
		fromAddress,
	))

	// create ethereum request
	body = &faucetapi.FaucetRequestData{
		Network: "evm",
		From:    types.HexBytes(evmcommon.HexToAddress("0xAAafD269cf7F6C7a7afa92A32127fbc72593638e").Bytes()),
	}
	data, err = json.Marshal(body)
	qt.Assert(t, err, qt.IsNil)
	resp, code = c.request("POST", data)
	// make the test blockchain to mine a block
	qt.Assert(t, code, qt.Equals, 200)
	respData = &faucetapi.FaucetResponse{}
	qt.Assert(t, json.Unmarshal(resp, &respData), qt.IsNil)
	t.Logf("%s", fmt.Sprintf(
		`"response": {
				"code": %d,
				"data": {
					"txHash": "%s"
				},
			}`,
		code,
		evmcommon.BytesToHash(respData.TxHash).Hex(),
	))
	balance, err := e.ClientBalanceAt(context.Background(), evmcommon.HexToAddress("0xAAafD269cf7F6C7a7afa92A32127fbc72593638e"), nil)
	qt.Assert(t, err, qt.IsNil)
	// 0 balance as no commited block
	qt.Assert(t, balance.Cmp(big.NewInt(int64(0))), qt.Equals, 0)
	e.TestBackend().Commit()
	balance, err = e.ClientBalanceAt(context.Background(), evmcommon.HexToAddress("0xAAafD269cf7F6C7a7afa92A32127fbc72593638e"), nil)
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
