package btc_relayer

import (
	"encoding/hex"
	"fmt"
	"github.com/Zou-XueYan/btc_relayer/log"
	"github.com/Zou-XueYan/btc_relayer/observer"
	"github.com/btcsuite/btcd/txscript"
	"testing"
	"time"
)

func TestNewBtcConfig(t *testing.T) {
	conf, err := NewBtcConfig("./conf.json")
	if err != nil {
		t.Fatalf("new btc config failed: %v", err)
	}

	fmt.Printf("config btc_addr: %s, nettype: %s\n", conf.BtcJsonRpcAddress, conf.NetType)
}

func TestNewBtcRelayer(t *testing.T) {
	r, err := NewBtcRelayer("./conf.json")
	if err != nil {
		t.Fatalf("Failed to new relayer: %v", err)
	}

	fmt.Printf("relayer's addr on allia-chain: %s\n", r.account.Address.ToHexString())
}

func TestBtcRelayer_BtcListen(t *testing.T) {
	r, err := NewBtcRelayer("./conf.json")
	if err != nil {
		t.Fatalf("Failed to new relayer: %v", err)
	}
	log.InitLog(log.InfoLog, log.PATH, log.Stdout)
	go r.BtcListen()
	go func() {
		for item := range r.relaying {
			fmt.Printf("Item heigh: %d\t", item.Height)
			fmt.Printf("Item proof: %s\t", item.Proof)
			fmt.Printf("Item txid: %s\n", item.Txid)
		}
	}()

	time.Sleep(time.Second * 60)
}

func TestBtcRelayer_Relay(t *testing.T) {
	r, err := NewBtcRelayer("./conf.json")
	if err != nil {
		t.Fatalf("Failed to new relayer: %v", err)
	}
	log.InitLog(log.InfoLog, log.PATH, log.Stdout)

	go func() {
		test := &observer.CrossChainItem{
			Proof:  "000000204149a82a4db84c25eabdd220ae55e568f3332f9a9d6bcc21be8d010000000000f783cb176b1c29fcb191eeb7299a105fc5db9a42be7cec34d08b8d819bb64fe44d1c495d71a5021a2ad64e1e26000000077702820166697756300bb36b2268ff36d93bbe63d09d42b42c7eb52a06aa9153320007b74b0935cbd73dd85deb23a2cc2268514e72d3795b563db1f77f8503aac3690bf489db8b0f3630a0f50a6767790c6f178d1027385f14d7e70ce2622a4a125da8708c3ddfb554fd8a636152007ca6f7ad7251c2514a07ea19a3718fb6b464259f0e6b7b06e34ae8f6c2e54d4d10c603cda1d2c1ebaf093c074e5b51e3a131b237e55e259bf74174441256a61f9d62d250d06ddcec3f6f94a3f6f43e3e3e59a4fc0e7c7dc59b926c2de2f4e9176ffbf7545e17b763cdc962d829500c321002bf00",
			Tx:     "0100000001ba32eb944a29e6c0d26189cc0cc67c5bd34d48ba876de114255bb6e3284ea7d1000000006a473044022040f94d2f640377d28f6aa0176477d0924c13a4772d1344c824ed69aac0d8c48b02200f9d475ff9f877a37b7d3e418f9cca6c0cb1909d3aa16361fd256c7aa05f80e9012103128a2c4525179e47f38cf3fefca37a61548ca4610255b3fb4ee86de2d3e80c0fffffffff0300350c000000000017a91487a9652e9b396545598c0fc72cb5a98848bf93d38700000000000000002c6a2a66000000000000000200000000000003e81727e090b158ee5c69c7e46076a996c4bd6159286ef9621225a0860100000000001976a91428d2e8cee08857f569e5a1b147c5d5e87339e08188ac00000000",
			Height: 1572760,
			Txid:   "aa03857ff7b13d565b79d3724e516822cca223eb5dd83dd7cb35094bb7070032",
		}
		time.Sleep(time.Second * 5)
		r.relaying <- test
	}()
	go r.Relay()

	time.Sleep(time.Second * 30)
}

func TestBtcRelayer_AllianceListen(t *testing.T) {
	r, err := NewBtcRelayer("./conf.json")
	if err != nil {
		t.Fatalf("Failed to new relayer: %v", err)
	}
	log.InitLog(log.InfoLog, log.PATH, log.Stdout)
	go r.AllianceListen()
	go func() {
		for item := range r.Collecting {
			fmt.Printf("Item tx: %s\n", item.Tx)
		}
	}()

	time.Sleep(time.Second * 60)
}

func TestS(t *testing.T) {
	s := "3045022100a45595aa914ed049a8fa5e3c132ea14422dc8adc5ec21f3944115057ab91979202206fe57b423324aba1cf883986a3b59f4a6660459b57153167bcaab0cffd27075901 0265c641e2412265377533220cac84cb54f1566e29826c0e1a48aa6ed05257c601"

	b, _ := hex.DecodeString(s)
	sb, _ := txscript.DisasmString(b)
	fmt.Println(sb)
}
