package btc_relayer

import (
	"fmt"
	"github.com/ontio/btcrelayer/log"
	"os"
	"testing"
	"time"
)

var (
	txArr = []string{
		"01000000019f074c07f34ffdcac88f76aa403e0725a90870b974c777a7236d6db067481ff2020000006b483045022100c5647452812dd245de91536de723d35239cbd49bb4dd924a5b6376b099a8a716022078938060af6771a44913893eaf0b091de365ee3a7a6ecefa830bf5d4caf6c996012103128a2c4525179e47f38cf3fefca37a61548ca4610255b3fb4ee86de2d3e80c0fffffffff03204e00000000000017a91487a9652e9b396545598c0fc72cb5a98848bf93d3870000000000000000276a256600000000000000020000000000000000f3b8a17f1f957f60c88f105e32ebff3f022e56a4a8ae0800000000001976a91428d2e8cee08857f569e5a1b147c5d5e87339e08188ac00000000",
		"01000000014d369f59ba828a4996f47b03229cbb976a0d0ed841c8c2f7a8e843289b15e631020000006a47304402202676093014919f3aa5dd5237566e4690dfc3503809246c3601cac38c4bd7636202207277e47ec95a28cf6eb30bceb36252704086e2b04c13e35797b0862741d44528012103128a2c4525179e47f38cf3fefca37a61548ca4610255b3fb4ee86de2d3e80c0fffffffff03204e00000000000017a91487a9652e9b396545598c0fc72cb5a98848bf93d3870000000000000000276a256600000000000000020000000000000000f3b8a17f1f957f60c88f105e32ebff3f022e56a4d8230900000000001976a91428d2e8cee08857f569e5a1b147c5d5e87339e08188ac00000000",
		"01000000012de9998067a46027dff12f8c114cb67f9788f52af731caa22a4c0b68babc58d0020000006a4730440220014f7b9c643ce47275552583fe3785f1d72307207b9e95ec293b16cbb15e10bc02205b6452982da43ebc2ddba531cba43a88baad003f2fcaef265e4ec30e8c077b0a012103128a2c4525179e47f38cf3fefca37a61548ca4610255b3fb4ee86de2d3e80c0fffffffff03204e00000000000017a91487a9652e9b396545598c0fc72cb5a98848bf93d3870000000000000000276a256600000000000000020000000000000000f3b8a17f1f957f60c88f105e32ebff3f022e56a4e84a0900000000001976a91428d2e8cee08857f569e5a1b147c5d5e87339e08188ac00000000",
		"0100000001e94f4feea73d3a44ad41cc52ea9ba5ccba9094dbd351310497eafff8233e2bec020000006b483045022100b1d0ec22de2404bf25d620e2c4488d2489d0eb46beca29f5dc219ee464ecd9e3022068a3f0c045b86e14fb99c72173ff9dd8d65dd784fc3af2050085a7c8e9b67cdb012103128a2c4525179e47f38cf3fefca37a61548ca4610255b3fb4ee86de2d3e80c0fffffffff03204e00000000000017a91487a9652e9b396545598c0fc72cb5a98848bf93d3870000000000000000276a256600000000000000020000000000000000f3b8a17f1f957f60c88f105e32ebff3f022e56a418c00900000000001976a91428d2e8cee08857f569e5a1b147c5d5e87339e08188ac00000000",
		"0100000001cbf0d729953a6227098a45752ed871c621f938260a53bfc6c998a347e2fd62e4020000006b483045022100f2e9590d6a9fcd5abce3775b0d263194547890f825350e81927468d807c43dd70220165a5edbca9f1cd42a6197a20ac4d97985dcde02f303b061541c1f86e026c398012103128a2c4525179e47f38cf3fefca37a61548ca4610255b3fb4ee86de2d3e80c0fffffffff03204e00000000000017a91487a9652e9b396545598c0fc72cb5a98848bf93d3870000000000000000276a256600000000000000020000000000000000f3b8a17f1f957f60c88f105e32ebff3f022e56a430390a00000000001976a91428d2e8cee08857f569e5a1b147c5d5e87339e08188ac00000000",
		"01000000017b60622809f978db6b95b09cec9789e2d68539b5eb7b2015ae2f7d79028b320c020000006b48304502210088a67aa604db724f34b2410e813927d26ad3489655cfb252934b25713890a17b02203bbe698595467da469fd20a09049c5920e8f70cae591605375d16bc801253c1f012103128a2c4525179e47f38cf3fefca37a61548ca4610255b3fb4ee86de2d3e80c0fffffffff03204e00000000000017a91487a9652e9b396545598c0fc72cb5a98848bf93d3870000000000000000276a256600000000000000020000000000000000f3b8a17f1f957f60c88f105e32ebff3f022e56a430390a00000000001976a91428d2e8cee08857f569e5a1b147c5d5e87339e08188ac00000000",
		"010000000140b1e5b05e26757f799ebf9f9018f970e77b0e6f7123d19fabaf373a32c3d0a0020000006a47304402204fb2d2d3edebd65675aa8b27f24d2b2398cec0fbf9cac2cfcb7def6df9b7cd6202205d6881aae4b4d65d63efbc1ed1cbb669dc493811b94f2c1629573a5ef5568706012103128a2c4525179e47f38cf3fefca37a61548ca4610255b3fb4ee86de2d3e80c0fffffffff03204e00000000000017a91487a9652e9b396545598c0fc72cb5a98848bf93d3870000000000000000276a256600000000000000020000000000000000f3b8a17f1f957f60c88f105e32ebff3f022e56a430390a00000000001976a91428d2e8cee08857f569e5a1b147c5d5e87339e08188ac00000000",
	}
)

func TestNewRelayerConfig(t *testing.T) {
	conf, err := NewRelayerConfig("./conf.json")
	if err != nil {
		t.Fatalf("new btc config failed: %v", err)
	}

	fmt.Printf("config btc_addr: %s, nettype: %s\n", conf.BtcObConf.BtcJsonRpcAddress, conf.AlliaObConf.AllianceJsonRpcAddress)
}

func TestNewBtcRelayer(t *testing.T) {
	conf, _ := NewRelayerConfig("./conf.json")
	r, err := NewBtcRelayer(conf)
	if err != nil {
		t.Fatalf("Failed to new relayer: %v", err)
	}

	fmt.Printf("relayer's addr on allia-chain: %s\n", r.account.Address.ToHexString())
}

func TestBtcRelayer_BtcListen(t *testing.T) {
	conf, _ := NewRelayerConfig("./conf.json")
	r, err := NewBtcRelayer(conf)
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

//func TestBtcRelayer_Relay(t *testing.T) {
//	r, err := NewBtcRelayer("./conf.json")
//	if err != nil {
//		t.Fatalf("Failed to new relayer: %v", err)
//	}
//	log.InitLog(log.InfoLog, log.PATH, log.Stdout)
//
//	go func() {
//		test := &observer.CrossChainItem{
//			Proof:  "000000204149a82a4db84c25eabdd220ae55e568f3332f9a9d6bcc21be8d010000000000f783cb176b1c29fcb191eeb7299a105fc5db9a42be7cec34d08b8d819bb64fe44d1c495d71a5021a2ad64e1e26000000077702820166697756300bb36b2268ff36d93bbe63d09d42b42c7eb52a06aa9153320007b74b0935cbd73dd85deb23a2cc2268514e72d3795b563db1f77f8503aac3690bf489db8b0f3630a0f50a6767790c6f178d1027385f14d7e70ce2622a4a125da8708c3ddfb554fd8a636152007ca6f7ad7251c2514a07ea19a3718fb6b464259f0e6b7b06e34ae8f6c2e54d4d10c603cda1d2c1ebaf093c074e5b51e3a131b237e55e259bf74174441256a61f9d62d250d06ddcec3f6f94a3f6f43e3e3e59a4fc0e7c7dc59b926c2de2f4e9176ffbf7545e17b763cdc962d829500c321002bf00",
//			Tx:     "0100000001ba32eb944a29e6c0d26189cc0cc67c5bd34d48ba876de114255bb6e3284ea7d1000000006a473044022040f94d2f640377d28f6aa0176477d0924c13a4772d1344c824ed69aac0d8c48b02200f9d475ff9f877a37b7d3e418f9cca6c0cb1909d3aa16361fd256c7aa05f80e9012103128a2c4525179e47f38cf3fefca37a61548ca4610255b3fb4ee86de2d3e80c0fffffffff0300350c000000000017a91487a9652e9b396545598c0fc72cb5a98848bf93d38700000000000000002c6a2a66000000000000000200000000000003e81727e090b158ee5c69c7e46076a996c4bd6159286ef9621225a0860100000000001976a91428d2e8cee08857f569e5a1b147c5d5e87339e08188ac00000000",
//			Height: 1572760,
//			Txid:   "aa03857ff7b13d565b79d3724e516822cca223eb5dd83dd7cb35094bb7070032",
//		}
//		time.Sleep(time.Second * 5)
//		r.relaying <- test
//	}()
//	go r.Relay()
//
//	time.Sleep(time.Second * 30)
//}

func TestBtcRelayer_AllianceListen(t *testing.T) {
	conf, _ := NewRelayerConfig("./conf.json")
	r, err := NewBtcRelayer(conf)
	if err != nil {
		t.Fatalf("Failed to new relayer: %v", err)
	}
	log.InitLog(log.InfoLog, log.PATH, log.Stdout)
	go r.AllianceListen()
	go func() {
		for item := range r.collecting {
			fmt.Printf("Item tx: %s\n", item.Tx)
		}
	}()

	time.Sleep(time.Second * 60)
}

func TestBtcRelayer_ReBroadcast(t *testing.T) {
	defer os.RemoveAll("./retry.bin")
	conf, _ := NewRelayerConfig("./conf.json")
	r, err := NewBtcRelayer(conf)
	if err != nil {
		t.Fatal(err)
	}
	log.InitLog(0, log.Stdout)
	for _, tx := range txArr[:3] {
		r.retryDB.Put(tx)
	}

	go r.ReBroadcast()
	time.Sleep(3 * time.Minute)
}

func TestS(t *testing.T) {
	arr := []int{1, 2, 3}
	for _, val := range arr {
		fmt.Println(val)
		<-time.Tick(3 * time.Second)
	}
}
