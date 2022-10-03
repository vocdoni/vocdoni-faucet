package config

import (
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	vocdoniConfig "go.vocdoni.io/dvote/config"
)

type LogConfig struct {
	Level,
	Output,
	ErrorFile string
}

type FaucetTxOptions struct {
	GasLimit,
	TxCost,
	Tip uint64
	GasPrice *big.Int
}

type FaucetConfig struct {
	Amount *big.Int
	EVMNetwork,
	VocdoniNetwork string
	EVMTxOptions,
	VocdoniTxOptions *FaucetTxOptions
	EVMPrivKeys,
	VocdoniPrivKeys,
	EVMEndpoints,
	VocdoniEndpoints []string
	EVMTimeout,
	VocdoniTimeout time.Duration
}

type MetricsConfig struct {
	Enable          bool
	RefreshInterval int
}

type Config struct {
	DataDir, SigningKey string
	Save                bool
	Log                 *LogConfig
	Faucet              *FaucetConfig
	API                 *vocdoniConfig.API
	Metrics             *MetricsConfig
}

func NewConfig() *Config {
	return &Config{
		Log:     new(LogConfig),
		Faucet:  new(FaucetConfig),
		API:     new(vocdoniConfig.API),
		Metrics: new(MetricsConfig),
	}
}

func (cfg *Config) String() string {
	return fmt.Sprintf("DataDir: %s, SigningKey: %s, Save: %v, Log: %+v, Faucet: %+v, API: %+v, Metrics: %+v",
		cfg.DataDir, cfg.SigningKey, cfg.Save, cfg.Log, cfg.Faucet, cfg.API, cfg.Metrics)
}

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
	cfg.SigningKey = *pflag.String("signingPrivKey", "",
		"signing private key (if not specified, a new one will be created)")
	// metrics
	cfg.Metrics.Enable = *pflag.Bool("enableMetrics", true, "enable prometheus metrics")
	cfg.Metrics.RefreshInterval =
		*pflag.Int("metricsRefreshInterval", 10, "metrics refresh interval in seconds")
	//faucet
	cfg.Faucet.EVMPrivKeys = *pflag.StringArray("evmPrivKeys", []string{},
		"hexString privKeys for EVM faucet accounts")
	cfg.Faucet.VocdoniPrivKeys = *pflag.StringArray("vocdoniPrivKeys",
		[]string{}, "hexString privKeys for Vocdoni faucet accounts")
	cfg.Faucet.EVMEndpoints = *pflag.StringArray("evmEndpoints", []string{},
		"evm endpoints to connect with (requied for the evm faucet)")
	cfg.Faucet.VocdoniEndpoints = *pflag.StringArray("vocdoniEndpoints",
		[]string{}, "vocdoni endpoints to connect with (requied for the vocdoni faucet)")
	// api
	cfg.API.Route = *pflag.String("apiRoute", "/", "dvote API route")
	cfg.API.ListenHost = *pflag.String("apiListenHost", "0.0.0.0", "API endpoint listen address")
	cfg.API.ListenPort = *pflag.Int("apiListenPort", 8000, "API endpoint http port")
	cfg.API.Ssl.Domain = *pflag.String("apiSSLDomain", "",
		"enable TLS secure API domain with LetsEncrypt auto-generated certificate")
	cfg.API.AllowPrivate = *pflag.Bool("apiAllowPrivate", false,
		"allows private methods over the APIs")
	cfg.API.AllowedAddrs = *pflag.String("apiAllowedAddrs", "",
		"comma-delimited list of allowed client ETH addresses for private methods")
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
	viper.BindPFlag("signingKey", pflag.Lookup("signingPrivKey"))
	// metrics
	viper.BindPFlag("metrics.Enable", pflag.Lookup("enableMetrics"))
	viper.BindPFlag("metrics.RefreshInterval", pflag.Lookup("metricsRefreshInterval"))
	// faucet
	viper.BindPFlag("faucet.EVMPrivKeys", pflag.Lookup("evmPrivKeys"))
	viper.BindPFlag("faucet.VocdoniPrivKeys", pflag.Lookup("vocdoniPrivKeys"))
	viper.BindPFlag("faucet.EVMEndpoints", pflag.Lookup("evmEndpoints"))
	viper.BindPFlag("faucet.VocdoniEndpoints", pflag.Lookup("vocdoniEndpoints"))
	// api
	viper.BindPFlag("api.Route", pflag.Lookup("apiRoute"))
	viper.BindPFlag("api.ListenHost", pflag.Lookup("listenHost"))
	viper.BindPFlag("api.ListenPort", pflag.Lookup("listenPort"))
	viper.BindPFlag("api.AllowPrivate", pflag.Lookup("apiAllowPrivate"))
	viper.BindPFlag("api.AllowedAddrs", pflag.Lookup("apiAllowedAddrs"))
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
