package main

import (
	"os"
	"strings"
	"time"

	"github.com/cristalhq/aconfig"
	"github.com/cristalhq/aconfig/aconfigyaml"
	"github.com/sirupsen/logrus"
)

type Config struct {
	// ConfigFilename         string   `default:"" usage:`
	SubNets                []string `default:"" usage:"List of subnets traffic between which will not be counted"`
	IgnorList              []string `default:"" usage:"List of lines that will be excluded from the final log"`
	LogLevel               string   `default:"info" usage:"Log level: panic, fatal, error, warn, info, debug, trace"`
	GomtcAddr              string   `default:"http://127.0.0.1:3034" usage:"Address and port for connect to gomtc API"`
	FlowAddr               string   `default:"0.0.0.0:2055" usage:"Address and port to listen NetFlow packets"`
	NameFileToLog          string   `default:"" usage:"The file where logs will be written in the format of squid logs"`
	MTAddr                 string   `default:"" usage:"The address of the Mikrotik router, from which the data on the comparison of the MAC address and IP address is taken"`
	MTUser                 string   `default:"" usage:"User of the Mikrotik router, from which the data on the comparison of the MAC address and IP address is taken"`
	MTPass                 string   `default:"" usage:"The password of the user of the Mikrotik router, from which the data on the comparison of the mac-address and IP-address is taken"`
	Loc                    string   `default:"Asia/Yekaterinburg" usage:"Location for time"`
	Interval               string   `default:"10m" usage:"Interval to getting info from Mikrotik"`
	ReceiveBufferSizeBytes int      `default:"" usage:"Size of RxQueue, i.e. value for SO_RCVBUF in bytes"`
	NumOfTryingConnectToMT int      `default:"10" usage:"The number of attempts to connect to the microtik router"`
	DefaultQuotaHourly     uint     `default:"0" usage:"Default hourly traffic consumption quota"`
	DefaultQuotaDaily      uint     `default:"0" usage:"Default daily traffic consumption quota"`
	DefaultQuotaMonthly    uint     `default:"0" usage:"Default monthly traffic consumption quota"`
	SizeOneMegabyte        uint     `default:"1048576" usage:"The number of bytes in one megabyte"`
	UseTLS                 bool     `default:"false" usage:"Using TLS to connect to a router"`
	CSV                    bool     `default:"false" usage:"Output to csv"`
	UseOnlyAPI             bool     `default:"true" usage:"Use only gomt API"`
	Location               *time.Location
}

func newConfig() *Config {
	// fix for https://github.com/cristalhq/aconfig/issues/82
	args := []string{}
	for _, a := range os.Args {
		if !strings.HasPrefix(a, "-test.") {
			args = append(args, a)
		}
	}
	// fix for https://github.com/cristalhq/aconfig/issues/82

	var cfg Config
	loader := aconfig.LoaderFor(&cfg, aconfig.Config{
		// feel free to skip some steps :)
		// SkipEnv:      true,
		SkipFiles:          false,
		AllowUnknownFields: true,
		SkipDefaults:       false,
		SkipFlags:          false,
		EnvPrefix:          "GONSQUID",
		FlagPrefix:         "",
		Files: []string{
			"./config.yaml",
			"./config/config.yaml",
			"/etc/gonsquid/config.yaml",
			"/bin/local/bin/gonsquid/config.yaml",
			"/bin/local/bin/gonsquid/config/config.yaml",
			"/opt/gonsquid/config.yaml",
			"/opt/gonsquid/config/config.yaml",
		},
		FileDecoders: map[string]aconfig.FileDecoder{
			// from `aconfigyaml` submodule
			// see submodules in repo for more formats
			".yaml": aconfigyaml.New(),
			// ".toml": aconfigtoml.New(),
		},
		Args: args[1:], // [1:] важно, см. доку к FlagSet.Parse
	})

	if err := loader.Load(); err != nil {
		panic(err)
	}

	lvl, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		logrus.Errorf("Error parse the level of logs (%v). Installed by default = Info", cfg.LogLevel)
		lvl, _ = logrus.ParseLevel("info")
	}
	logrus.SetLevel(lvl)

	cfg.Location, err = time.LoadLocation(cfg.Loc)
	if err != nil {
		logrus.Errorf("Error loading Location(%v):%v", cfg.Loc, err)
		cfg.Location = time.UTC
	}

	logrus.Debugf("Config %#v:", cfg)

	return &cfg
}
