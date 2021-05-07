package main

import (
	"flag"

	"github.com/ilyakaznacheev/cleanenv"
	log "github.com/sirupsen/logrus"
)

type arrayFlags []string

func (i *arrayFlags) String() string {
	return "List of strings"
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

type Config struct {
	SubNets                arrayFlags `yaml:"SubNets" toml:"subnets" env:"GONSQUID_SUBNETS"`
	IgnorList              arrayFlags `yaml:"IgnorList" toml:"ignorlist" env:"GONSQUID_IGNOR_LIST"`
	ConfigFilename         string     `yaml:"ConfigFilename" toml:"configfilename" env:"GONSQUID_CONFIG"`
	LogLevel               string     `yaml:"LogLevel" toml:"loglevel" env:"GONSQUID_LOG_LEVEL"`
	FlowAddr               string     `yaml:"FlowAddr" toml:"flowaddr" env:"GONSQUID_FLOW_ADDR" env-default:"0.0.0.0:2055"`
	NameFileToLog          string     `yaml:"FileToLog" toml:"log" env:"GONSQUID_FLOW_LOG"`
	BindAddr               string     `yaml:"BindAddr" toml:"bindaddr" env:"GONSQUID_ADDR_M4M" env-default:":3030"`
	MTAddr                 string     `yaml:"MTAddr" toml:"mtaddr" env:"GONSQUID_ADDR_MT"`
	MTUser                 string     `yaml:"MTUser" toml:"mtuser" env:"GONSQUID_USER_MT"`
	MTPass                 string     `yaml:"MTPass" toml:"mtpass" env:"GONSQUID_PASS_MT"`
	loc                    string     `yaml:"loc" toml:"loc" env:"GONSQUID_LOC"`
	Interval               string
	ReceiveBufferSizeBytes int  `yaml:"receiveBufferSizeBytes" toml:"receiveBufferSizeBytes" env:"GONSQUID_BUFSIZE"`
	NumOfTryingConnectToMT int  `yaml:"numOfTryingConnectToMT" toml:"numOfTryingConnectToMT" env:"GONSQUID_NUMOFTRY"`
	DefaultQuotaHourly     uint `yaml:"defaultQuotaHourly" toml:"defaultQuotaHourly" env:"GONSQUID_QUOTAH"`
	DefaultQuotaDaily      uint `yaml:"defaultQuotaDaily" toml:"defaultQuotaDaily" env:"GONSQUID_QUOTAD"`
	DefaultQuotaMonthly    uint `yaml:"defaultQuotaMonthly" toml:"defaultQuotaMonthly" env:"GONSQUID_QUOTAM"`
	SizeOneMegabyte        uint `yaml:"SizeOneMegabyte" toml:"SizeOneMegabyte" env:"GONSQUID_SIZEOFMB"`
	UseTLS                 bool `yaml:"tls" toml:"tls" env:"GONSQUID_TLS"`
	CSV                    bool `yaml:"csv" toml:"csv" env:"GONSQUID_CSV"`
}

var (
	cfg Config
)

func newConfig() *Config {
	/* Parse command-line arguments */
	flag.IntVar(&cfg.ReceiveBufferSizeBytes, "buffer", 212992, "Size of RxQueue, i.e. value for SO_RCVBUF in bytes")
	flag.IntVar(&cfg.NumOfTryingConnectToMT, "num", 10, "The number of attempts to connect to the microtik router")
	flag.UintVar(&cfg.DefaultQuotaHourly, "quotah", 0, "Default hourly traffic consumption quota")
	flag.UintVar(&cfg.DefaultQuotaDaily, "quotad", 0, "Default daily traffic consumption quota")
	flag.UintVar(&cfg.DefaultQuotaMonthly, "quotam", 0, "Default monthly traffic consumption quota")
	flag.UintVar(&cfg.SizeOneMegabyte, "sizeofmb", 1048576, "The number of bytes in one megabyte")
	flag.StringVar(&cfg.FlowAddr, "addr", "0.0.0.0:2055", "Address and port to listen NetFlow packets")
	flag.StringVar(&cfg.LogLevel, "loglevel", "info", "Log level")
	flag.Var(&cfg.SubNets, "subnet", "List of subnets traffic between which will not be counted")
	flag.Var(&cfg.IgnorList, "ignorlist", "List of lines that will be excluded from the final log")
	flag.StringVar(&cfg.NameFileToLog, "log", "", "The file where logs will be written in the format of squid logs")
	flag.StringVar(&cfg.loc, "loc", "Asia/Yekaterinburg", "Location for time")
	flag.StringVar(&cfg.MTAddr, "mtaddr", "", "The address of the Mikrotik router, from which the data on the comparison of the MAC address and IP address is taken")
	flag.StringVar(&cfg.MTUser, "u", "", "User of the Mikrotik router, from which the data on the comparison of the MAC address and IP address is taken")
	flag.StringVar(&cfg.MTPass, "p", "", "The password of the user of the Mikrotik router, from which the data on the comparison of the mac-address and IP-address is taken")
	flag.StringVar(&cfg.BindAddr, "m4maddr", ":3030", "Listen address for response mac-address from mikrotik")
	flag.StringVar(&cfg.Interval, "interval", "10m", "Interval to getting info from Mikrotik")
	flag.StringVar(&cfg.ConfigFilename, "config", "config.toml", "Path to config file")
	flag.BoolVar(&cfg.UseTLS, "tls", false, "Using TLS to connect to a router")
	flag.BoolVar(&cfg.CSV, "csv", false, "Output to csv")

	flag.Parse()

	var config_source string
	err := cleanenv.ReadConfig(cfg.ConfigFilename, &cfg)
	if err != nil {
		log.Warningf("No config file(%v) found: %v", cfg.ConfigFilename, err)
		config_source = "ENV/CFG"
	} else {
		config_source = "CLI"
	}

	lvl, err := log.ParseLevel(cfg.LogLevel)
	if err != nil {
		log.Errorf("Error parse the level of logs (%v). Installed by default = Info", cfg.LogLevel)
		lvl, _ = log.ParseLevel("info")
	}
	log.SetLevel(lvl)

	log.Debugf("Config read from %s: %#v",
		config_source,
		cfg)

	return &cfg
}
