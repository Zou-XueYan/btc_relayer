package main

import (
	"flag"
	"github.com/Zou-XueYan/btc_relayer"
	"github.com/Zou-XueYan/btc_relayer/log"
	"github.com/Zou-XueYan/btc_relayer/signer"
)

var confFile string
var logPath string
var logLevel int

func init() {
	flag.StringVar(&confFile, "conf-file", "../conf.json", "configuration file for btc relayer")
	flag.StringVar(&logPath, "log-path", log.PATH, "log path for btc relayer")
	flag.IntVar(&logLevel, "log-level", 0, "log level: 0 trace, 1 debug, 2 info, 3 warn, 4 error, 5 fatal")
}

func main() {
	flag.Parse()

	log.InitLog(logLevel, log.Stdout)
	r, err := btc_relayer.NewBtcRelayer(confFile)
	if err != nil {
		log.Errorf("Failed to new a relayer: %v", err)
	}

	m, err := signer.NewSigningMachine(confFile)
	if err != nil {
		log.Errorf("Failed to new a signing machine: %v", err)
	}

	go r.BtcListen()
	go r.Relay()
	go r.AllianceListen()

	go m.Signing(r.Collecting)
	go m.Broadcasting()

	select {}
}
