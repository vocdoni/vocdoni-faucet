package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	vocdoniConfig "go.vocdoni.io/dvote/config"
)

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
	// Amount to send by the faucet
	Amount uint64
	// EVM network name to connect with.
	// Accepted one of SupportedFaucetNetworksMap
	EVMNetwork,
	// Vocdoni network name to connect with.
	// Accepted one of SupportedFaucetNetworksMap
	VocdoniNetwork,
	// VocdoniPrivKey Vocdoni faucet signer key
	VocdoniPrivKey string
	// EVMPrivKeys EVM faucet signers keys
	EVMPrivKeys,
	// EVMEndpoints endpoints to connect the EVM faucet with
	EVMEndpoints []string
	// EVMTimeout faucet global timeout for EVM operations in seconds
	EVMTimeout time.Duration
	// SendConditions config for sendConditions
	SendConditions SendConditionsConfig
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
	// Save save the config if true
	Save   bool
	Log    *LogConfig
	Faucet *FaucetConfig
	API    *vocdoniConfig.API
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
	return fmt.Sprintf("DataDir: %s, Save: %v, Log: %+v, Faucet: %+v, API: %+v",
		cfg.DataDir, cfg.Save, cfg.Log, cfg.Faucet, cfg.API)
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
	cfg.Save = *pflag.Bool("saveConfig", false,
		"overwrites an existing config file with the CLI provided flags")
	//faucet
	cfg.Faucet.EnableEVM = *pflag.Bool("enableEVM", true, "enable evm faucet")
	cfg.Faucet.EnableVocdoni = *pflag.Bool("enableVocdoni", true, "enable vocdoni faucet")
	cfg.Faucet.EVMPrivKeys = *pflag.StringArray("evmPrivKeys", []string{},
		"hexString privKeys for EVM faucet accounts")
	cfg.Faucet.VocdoniPrivKey = *pflag.String("vocdoniPrivKey",
		"", "hexString privKeys for vocdoni faucet accounts")
	cfg.Faucet.EVMEndpoints = *pflag.StringArray("evmEndpoints", []string{},
		"evm endpoints to connect with (requied for the evm faucet)")
	cfg.Faucet.EVMNetwork = *pflag.String("evmNetwork",
		"", "one of the available evm chains")
	cfg.Faucet.VocdoniNetwork = *pflag.String("vocdoniNetwork",
		"", "one of the available vocdoni networks")
	cfg.Faucet.Amount = *pflag.Uint64("faucetAmount", 100, "faucet amount")
	cfg.Faucet.SendConditions.Balance = *pflag.Uint64("faucetAmountThreshold", 100, "minimum amount threshold for transfer")
	cfg.Faucet.SendConditions.Challenge = *pflag.Bool("faucetEnableChallenge", false, "if true a faucet challenge must be solved")
	// api
	cfg.API.Route = *pflag.String("apiRoute", "/", "dvote API route")
	cfg.API.ListenHost = *pflag.String("apiListenHost", "0.0.0.0", "API endpoint listen address")
	cfg.API.ListenPort = *pflag.Int("apiListenPort", 8000, "API endpoint http port")
	cfg.API.Ssl.Domain = *pflag.String("apiSSLDomain", "",
		"enable TLS secure API domain with LetsEncrypt auto-generated certificate")
	// parse flags
	pflag.Parse()

	// setting up viper
	viper := viper.New()
	viper.AddConfigPath(cfg.DataDir)
	viper.SetConfigName("vocdoni-faucet")
	viper.SetConfigType("yml")
	viper.SetEnvPrefix("VOCDONIFAUCET")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// binding flags to viper
	// logging
	viper.BindPFlag("logLevel", pflag.Lookup("logLevel"))
	viper.BindPFlag("logErrorFile", pflag.Lookup("logErrorFile"))
	viper.BindPFlag("logOutput", pflag.Lookup("logOutput"))
	// common
	viper.BindPFlag("dataDir", pflag.Lookup("dataDir"))
	// faucet
	viper.BindPFlag("faucet.EnableEVM", pflag.Lookup("enableEVM"))
	viper.BindPFlag("faucet.EnableVocdoni", pflag.Lookup("enableVocdoni"))
	viper.BindPFlag("faucet.EVMPrivKeys", pflag.Lookup("evmPrivKeys"))
	viper.BindPFlag("faucet.VocdoniPrivKey", pflag.Lookup("vocdoniPrivKey"))
	viper.BindPFlag("faucet.EVMEndpoints", pflag.Lookup("evmEndpoints"))
	viper.BindPFlag("faucet.EVMNetwork", pflag.Lookup("evmNetwork"))
	viper.BindPFlag("faucet.VocdoniNetwork", pflag.Lookup("vocdoniNetwork"))
	viper.BindPFlag("faucet.Amount", pflag.Lookup("faucetAmount"))
	viper.BindPFlag("faucet.SendConditions.Balance", pflag.Lookup("faucetAmountThreshold"))
	viper.BindPFlag("faucet.SendConditions.Challenge", pflag.Lookup("faucetEnableChallenge"))
	// api
	viper.BindPFlag("api.Route", pflag.Lookup("apiRoute"))
	viper.BindPFlag("api.ListenHost", pflag.Lookup("listenHost"))
	viper.BindPFlag("api.ListenPort", pflag.Lookup("listenPort"))
	viper.Set("api.Ssl.DirCert", cfg.DataDir+"/tls")
	viper.BindPFlag("api.Ssl.Domain", pflag.Lookup("apiSSLDomain"))

	// check if config file exists
	_, err = os.Stat(cfg.DataDir + "/vocdoni-faucet.yml")
	if os.IsNotExist(err) {
		// creting config folder if not exists
		err = os.MkdirAll(cfg.DataDir, os.ModePerm)
		if err != nil {
			return fmt.Errorf("cannot create data directory: %s", err)
		}
		// create config file if not exists
		if err := viper.SafeWriteConfig(); err != nil {
			return fmt.Errorf("cannot write config file into config dir: %s", err)

		}
	} else {
		// read config file
		err = viper.ReadInConfig()
		if err != nil {
			return fmt.Errorf("cannot read loaded config file in %s: %s", cfg.DataDir, err)

		}
	}
	if err := viper.Unmarshal(&cfg); err != nil {
		return fmt.Errorf("cannot unmarshal loaded config file: %s", err)
	}
	// save config if required
	if cfg.Save {
		viper.Set("saveConfig", false)
		if err := viper.SafeWriteConfig(); err != nil {
			return fmt.Errorf("cannot overwrite config file into config dir: %s", err)
		}
	}
	return nil
}
