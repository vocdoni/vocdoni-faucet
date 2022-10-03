package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"go.vocdoni.io/dvote/crypto/ethereum"
	"go.vocdoni.io/dvote/httprouter"
	"go.vocdoni.io/dvote/log"
	"go.vocdoni.io/vocdoni-faucet/config"
	"go.vocdoni.io/vocdoni-faucet/internal"
)

func main() {
	// Don't use the log package here, because we want to report the version
	// before loading the config. This is because something could go wrong
	// while loading the config, and because the logger isn't set up yet.
	// For the sake of including the version in the log, it's also included
	// in a log line later on.
	fmt.Fprintf(os.Stderr, "vocdoni-faucet version %q\n", internal.Version)

	// setup config
	cfg := config.NewConfig()
	if err := cfg.InitConfig(); err != nil {
		panic(fmt.Sprintf("error creating configuration: %s", err))
	}

	// init logger
	log.Init(cfg.Log.Level, cfg.Log.Output)
	if path := cfg.Log.ErrorFile; path != "" {
		if err := log.SetFileErrorLog(path); err != nil {
			log.Fatal(err)
		}
	}
	log.Debugf("starting vocdoni-faucet version %s with config %s", internal.Version, cfg.String())

	// setup signing key
	var signer *ethereum.SignKeys
	signer = ethereum.NewSignKeys()
	// add signing private key if exist in configuration or flags
	if len(cfg.SigningKey) != 32 {
		log.Infof("adding custom signing key")
		err := signer.AddHexKey(cfg.SigningKey)
		if err != nil {
			log.Fatalf("error adding hex key: (%s)", err)
		}
		pub, _ := signer.HexString()
		log.Infof("using custom pubKey %s", pub)
	} else {
		log.Fatal("no private key or wrong key (size != 16 bytes)")
	}
	// add authorized keys for private methods
	if cfg.API.AllowPrivate && cfg.API.AllowedAddrs != "" {
		keys := strings.Split(cfg.API.AllowedAddrs, ",")
		for _, key := range keys {
			signer.AddAuthKey(ethcommon.HexToAddress(key))
		}
	}

	// init proxy
	var httpRouter httprouter.HTTProuter
	httpRouter.TLSdomain = cfg.API.Ssl.Domain
	httpRouter.TLSdirCert = cfg.API.Ssl.DirCert
	if err := httpRouter.Init(cfg.API.ListenHost, cfg.API.ListenPort); err != nil {
		log.Fatal(err)
	}

	// init evm faucet service if enabled

	// init vocdoni faucet service if enabled

	// init REST API

	log.Info("startup complete")
	// close if interrupt received
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	log.Warnf("received SIGTERM, exiting at %s", time.Now().Format(time.RFC850))
	os.Exit(0)
}
