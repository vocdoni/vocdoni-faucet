package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	vocdoniConfig "go.vocdoni.io/dvote/config"
)

var ErrBindPFlag = errors.New("viper error binding flag")

// LogConfig logging configuration
type LogConfig struct {
	Level,
	Output,
	ErrorFile string
}

// FaucetConfig faucet configuration
type FaucetConfig struct {
	EnableEVM,
	EnableVocdoni bool
	// EVMAmount evm amount to send by the faucet
	EVMAmount,
	// VocdoniAmount vocdoni amount to send by the faucet
	VocdoniAmount uint64
	// EVM network name to connect with.
	// Accepted one of SupportedFaucetNetworksMap
	EVMNetwork,
	// VocdoniPrivKey Vocdoni faucet signer key
	VocdoniPrivKey string
	// EVMPrivKeys EVM faucet signers keys
	EVMPrivKeys,
	// EVMEndpoints endpoints to connect the EVM faucet with
	EVMEndpoints,
	// Vocdoni network name to connect with.
	// Accepted one of SupportedFaucetNetworksMap
	VocdoniNetworks []string
	// EVMTimeout faucet global timeout for EVM operations in seconds
	EVMTimeout time.Duration
	// SendConditions config for sendConditions
	EVMSendConditions     SendConditionsConfig
	VocdoniSendConditions SendConditionsConfig
}

// SendConditionsConfig represents the send conditions of the faucet configuration
type SendConditionsConfig struct {
	Balance   uint64
	Challenge bool
}

// Config the global configuration of the faucet
type Config struct {
	// DataDir base directory to store data
	DataDir string
	Log     *LogConfig
	Faucet  *FaucetConfig
	API     *vocdoniConfig.API
}

// NewConfig returns a pointer to an initialized Config
func NewConfig() *Config {
	return &Config{
		Log:    new(LogConfig),
		Faucet: new(FaucetConfig),
		API:    new(vocdoniConfig.API),
	}
}

// Strings returns the configuration as a string
func (cfg *Config) String() string {
	return fmt.Sprintf("DataDir: %s, Log: %+v, Faucet: %+v, API: %+v",
		cfg.DataDir, cfg.Log, cfg.Faucet, cfg.API)
}

// InitConfig initializes the Config with user provided args
func (cfg *Config) InitConfig() error {
	// get $HOME
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot get user home directory: %w", err)
	}

	// flags
	// logging
	cfg.Log.Level = *pflag.String("logLevel", "info", "log level (debug, info, warn, error, fatal)")
	cfg.Log.Output = *pflag.String("logOutput", "stdout", "log output (stdout, stderr or filepath)")
	cfg.Log.ErrorFile = *pflag.String("logErrorFile", "", "log errors and warnings to a file")
	// common
	pflag.StringVar(&cfg.DataDir, "dataDir", home+"/.faucet", "directory where data is stored")
	// faucet
	cfg.Faucet.EnableEVM = *pflag.Bool("enableEVM", true, "enable evm faucet")
	cfg.Faucet.EnableVocdoni = *pflag.Bool("enableVocdoni", true, "enable vocdoni faucet")
	cfg.Faucet.EVMPrivKeys = *pflag.StringSlice("evmPrivKeys", []string{},
		"hexString privKeys for EVM faucet accounts")
	cfg.Faucet.VocdoniPrivKey = *pflag.String("vocdoniPrivKey",
		"", "hexString privKeys for vocdoni faucet accounts")
	cfg.Faucet.EVMEndpoints = *pflag.StringSlice("evmEndpoints", []string{},
		"evm endpoints to connect with (requied for the evm faucet)")
	cfg.Faucet.EVMNetwork = *pflag.String("evmNetwork",
		"", "one of the available evm chains")
	cfg.Faucet.VocdoniNetworks = *pflag.StringSlice("vocdoniNetworks",
		[]string{}, "one or more of the available vocdoni networks")
	cfg.Faucet.EVMAmount = *pflag.Uint64(
		"faucetEVMAmount",
		1,
		"evm faucet amount in wei (1000000000000000000 == 1 ETH)",
	)
	cfg.Faucet.VocdoniAmount = *pflag.Uint64("faucetVocdoniAmount", 100, "vocdoni faucet amount")
	cfg.Faucet.EVMSendConditions.Balance = *pflag.Uint64(
		"faucetEVMAmountThreshold",
		1,
		"minimum EVM amount threshold for transfer",
	)
	cfg.Faucet.EVMSendConditions.Challenge = *pflag.Bool(
		"faucetEVMEnableChallenge",
		false,
		"if true a EVM faucet challenge must be solved",
	)
	cfg.Faucet.VocdoniSendConditions.Balance = *pflag.Uint64(
		"faucetVocdoniAmountThreshold",
		100,
		"minimum vocdoni amount threshold for transfer",
	)
	cfg.Faucet.VocdoniSendConditions.Challenge = *pflag.Bool(
		"faucetVocdoniEnableChallenge",
		false,
		"if true a vocdoni faucet challenge must be solved",
	)
	// api
	cfg.API.Route = *pflag.String("apiRoute", "/", "dvote API route")
	cfg.API.ListenHost = *pflag.String("apiListenHost", "0.0.0.0", "API endpoint listen address")
	cfg.API.ListenPort = *pflag.Int("apiListenPort", 8000, "API endpoint http port")
	cfg.API.Ssl.Domain = *pflag.String("apiTLSDomain", "",
		"enaapiLle TLS secure API domain with LetsEncrypt auto-generated certificate")
	cfg.API.AllowedAddrs = *pflag.String(
		"apiWhitelist",
		"",
		"bearer token whitelist for accepting requests (comma separated string)",
	)
	// parse flags
	pflag.Parse()

	// setting up viper
	viper := viper.New()
	viper.SetEnvPrefix("VOCDONIFAUCET")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// binding flags to viper
	// logging
	if err := viper.BindPFlag("log.Level", pflag.Lookup("logLevel")); err != nil {
		return fmt.Errorf("%s: %s", ErrBindPFlag, err)
	}
	if err := viper.BindPFlag("log.ErrorFile", pflag.Lookup("logErrorFile")); err != nil {
		return fmt.Errorf("%s: %s", ErrBindPFlag, err)
	}
	if err := viper.BindPFlag("log.Output", pflag.Lookup("logOutput")); err != nil {
		return fmt.Errorf("%s: %s", ErrBindPFlag, err)
	}
	// common
	if err := viper.BindPFlag("dataDir", pflag.Lookup("dataDir")); err != nil {
		return fmt.Errorf("%s: %s", ErrBindPFlag, err)
	}
	// faucet
	if err := viper.BindPFlag("faucet.EnableEVM", pflag.Lookup("enableEVM")); err != nil {
		return fmt.Errorf("%s: %s", ErrBindPFlag, err)
	}
	if err := viper.BindPFlag("faucet.EnableVocdoni", pflag.Lookup("enableVocdoni")); err != nil {
		return fmt.Errorf("%s: %s", ErrBindPFlag, err)
	}
	if err := viper.BindPFlag("faucet.EVMPrivKeys", pflag.Lookup("evmPrivKeys")); err != nil {
		return fmt.Errorf("%s: %s", ErrBindPFlag, err)
	}
	if err := viper.BindPFlag("faucet.VocdoniPrivKey", pflag.Lookup("vocdoniPrivKey")); err != nil {
		return fmt.Errorf("%s: %s", ErrBindPFlag, err)
	}
	if err := viper.BindPFlag("faucet.EVMEndpoints", pflag.Lookup("evmEndpoints")); err != nil {
		return fmt.Errorf("%s: %s", ErrBindPFlag, err)
	}
	if err := viper.BindPFlag("faucet.EVMNetwork", pflag.Lookup("evmNetwork")); err != nil {
		return fmt.Errorf("%s: %s", ErrBindPFlag, err)
	}
	if err := viper.BindPFlag("faucet.VocdoniNetworks", pflag.Lookup("vocdoniNetworks")); err != nil {
		return fmt.Errorf("%s: %s", ErrBindPFlag, err)
	}
	if err := viper.BindPFlag("faucet.EVMAmount", pflag.Lookup("faucetEVMAmount")); err != nil {
		return fmt.Errorf("%s: %s", ErrBindPFlag, err)
	}
	if err := viper.BindPFlag(
		"faucet.VocdoniAmount",
		pflag.Lookup("faucetVocdoniAmount"),
	); err != nil {
		return fmt.Errorf("%s: %s", ErrBindPFlag, err)
	}
	if err := viper.BindPFlag(
		"faucet.EVMSendConditions.Balance",
		pflag.Lookup("faucetEVMAmountThreshold"),
	); err != nil {
		return fmt.Errorf("%s: %s", ErrBindPFlag, err)
	}
	if err := viper.BindPFlag(
		"faucet.EVMSendConditions.Challenge",
		pflag.Lookup("faucetEVMEnableChallenge"),
	); err != nil {
		return fmt.Errorf("%s: %s", ErrBindPFlag, err)
	}
	if err := viper.BindPFlag(
		"faucet.VocdoniSendConditions.Balance",
		pflag.Lookup("faucetVocdoniAmountThreshold"),
	); err != nil {
		return fmt.Errorf("%s: %s", ErrBindPFlag, err)
	}
	if err := viper.BindPFlag(
		"faucet.VocdoniSendConditions.Challenge",
		pflag.Lookup("faucetVocdoniEnableChallenge"),
	); err != nil {
		return fmt.Errorf("%s: %s", ErrBindPFlag, err)
	}
	// api
	if err := viper.BindPFlag("api.Route", pflag.Lookup("apiRoute")); err != nil {
		return fmt.Errorf("%s: %s", ErrBindPFlag, err)
	}
	if err := viper.BindPFlag("api.ListenHost", pflag.Lookup("apiListenHost")); err != nil {
		return fmt.Errorf("%s: %s", ErrBindPFlag, err)
	}
	if err := viper.BindPFlag("api.ListenPort", pflag.Lookup("apiListenPort")); err != nil {
		return fmt.Errorf("%s: %s", ErrBindPFlag, err)
	}
	if err := viper.BindPFlag("api.AllowedAddrs", pflag.Lookup("apiWhitelist")); err != nil {
		return fmt.Errorf("%s: %s", ErrBindPFlag, err)
	}
	viper.Set("api.Ssl.DirCert", cfg.DataDir+"/tls")
	if err := viper.BindPFlag("api.Ssl.Domain", pflag.Lookup("apiTLSDomain")); err != nil {
		return fmt.Errorf("%s: %s", ErrBindPFlag, err)
	}
	if err := viper.Unmarshal(&cfg); err != nil {
		return err
	}
	return nil
}
