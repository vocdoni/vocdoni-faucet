package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.vocdoni.io/dvote/httprouter"
	"go.vocdoni.io/dvote/log"
	"go.vocdoni.io/vocdoni-faucet/api"
	"go.vocdoni.io/vocdoni-faucet/config"
	"go.vocdoni.io/vocdoni-faucet/faucet"
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

	// init proxy
	var httpRouter httprouter.HTTProuter
	httpRouter.TLSdomain = cfg.API.Ssl.Domain
	httpRouter.TLSdirCert = cfg.API.Ssl.DirCert
	if err := httpRouter.Init(cfg.API.ListenHost, cfg.API.ListenPort); err != nil {
		log.Fatal(err)
	}

	// init vocdoni faucet
	v := faucet.NewVocdoni()
	if cfg.Faucet.EnableVocdoni {
		if err := v.Init(context.Background(), cfg.Faucet); err != nil {
			log.Fatal(err)
		}
	}

	// init evm faucet
	e := faucet.NewEVM()
	if cfg.Faucet.EnableEVM {
		if err := e.Init(context.Background(), cfg.Faucet); err != nil {
			log.Fatal(err)
		}
	}

	// init api
	a := api.NewAPI()
	if err := a.Init(&httpRouter, cfg.API.Route, v, e); err != nil {
		log.Fatal(err)
	}
	log.Infof("API available at %s", cfg.API.Route)

	log.Info("startup complete")
	// close if interrupt received
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	log.Warnf("received SIGTERM, exiting at %s", time.Now().Format(time.RFC850))
	os.Exit(0)
}
