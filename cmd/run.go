package main

import (
	"flag"
	"github.com/ontio/btcrelayer"
	"github.com/ontio/btcrelayer/log"
	"github.com/ontio/btcrelayer/observer"
	"time"
)

var (
	confFile string
)

func init() {
	flag.StringVar(&confFile, "conf-file", "../conf.json", "configuration file for btc relayer")
}

func main() {
	flag.Parse()

	conf, err := btc_relayer.NewRelayerConfig(confFile)
	if err != nil {
		log.Errorf("failed to new a config: %v", err)
		return
	}

	log.InitLog(conf.LogLevel, log.Stdout)
	r, err := btc_relayer.NewBtcRelayer(conf)
	if err != nil {
		log.Errorf("Failed to new a relayer: %v", err)
		return
	}
	if conf.SleepTime > 0 {
		observer.SleepTime = time.Duration(conf.SleepTime)
	}
	go r.BtcListen()
	go r.Relay()
	go r.AllianceListen()
	go r.Broadcast()
	go r.ReBroadcast()

	select {}
}
