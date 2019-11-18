package btc_relayer

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcutil/base58"
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

func getPrivks() []*btcec.PrivateKey {
	arr := []string {
		"cTqbqa1YqCf4BaQTwYDGsPAB4VmWKUU67G5S1EtrHSWNRwY6QSag",
		"cT2HP4QvL8c6otn4LrzUWzgMBfTo1gzV2aobN1cTiuHPXH9Jk2ua",
		"cSQmGg6spbhd23jHQ9HAtz3XU7GYJjYaBmFLWHbyKa9mWzTxEY5A",
		"cPYAx61EjwshK5SQ6fqH7QGjc8L48xiJV7VRGpYzPSbkkZqrzQ5b",
		"cVV9UmtnnhebmSQgHhbDZWCb7zBHbiAGDB9a5M2ffe1WpqvwD5zg",
		//"cNK7BwHmi8rZiqD2QfwJB1R6bF6qc7iVTMBNjTr2ACbsoq1vWau8",
		//"cUZdDF9sL11ya5civzMRYVYojoojjHbmWWm1yC5uRzfBRePVbQTZ",
	}
	res := make([]*btcec.PrivateKey, 5)
	for i, v := range arr {
		privk, _ := btcec.PrivKeyFromBytes(btcec.S256(), base58.Decode(v))
		res[i] = privk
	}

	return res
}

func TestS(t *testing.T) {
	redeem := "5521023ac710e73e1410718530b2686ce47f12fa3c470a9eb6085976b70b01c64c9f732102c9dc4d8f419e325bbef0fe039ed6feaf2079a2ef7b27336ddb79be2ea6e334bf2102eac939f2f0873894d8bf0ef2f8bbdd32e4290cbf9632b59dee743529c0af9e802103378b4a3854c88cca8bfed2558e9875a144521df4a75ab37a206049ccef12be692103495a81957ce65e3359c114e6c2fe9f97568be491e3f24d6fa66cc542e360cd662102d43e29299971e802160a92cfcd4037e8ae83fb8f6af138684bebdc5686f3b9db21031e415c04cbc9b81fbee6e04d8c902e8f61109a2c9883a959ba528c52698c055a57ae"
	rb, _ := hex.DecodeString(redeem)
	hasher := sha256.New()
	hasher.Write(rb)

	addr, err := btcutil.NewAddressWitnessScriptHash(hasher.Sum(nil), &chaincfg.RegressionNetParams)
	if err != nil {
		t.Fatal(err)
	}
	str, _ := txscript.DisasmString(rb)
	fmt.Println(str)

	script, err := txscript.PayToAddrScript(btcutil.Address(addr))
	if err != nil {
		t.Fatal(err)
	}

	str, _ = txscript.DisasmString(script)
	fmt.Println(str)
	txHash, _ := chainhash.NewHashFromStr("25b9a71dc448c43ccebb7afaf78f505c9e8ee729dafe0483b69be5fefc52d868")
	prevOut := wire.NewOutPoint(txHash, 0)
	txIn := wire.NewTxIn(prevOut, nil, nil)

	mtx := wire.NewMsgTx(wire.TxVersion)
	mtx.AddTxIn(txIn)
	targetAddr, _ := btcutil.DecodeAddress("mjEoyyCPsLzJ23xMX6Mti13zMyN36kzn57", &chaincfg.RegressionNetParams)
	s, _ := txscript.PayToAddrScript(targetAddr)
	txOut := wire.NewTxOut(btcutil.SatoshiPerBitcoin - 2000, s)
	mtx.AddTxOut(txOut)

	sigs := make([][]byte, 6)
	sh := txscript.NewTxSigHashes(mtx)
	for i, privk := range getPrivks() {
		sig, err := txscript.RawTxInWitnessSignature(mtx, sh, 0, btcutil.SatoshiPerBitcoin, rb, txscript.SigHashAll, privk)
		if err != nil {
			t.Fatalf("no%d: %v", i, err)
		}
		sigs[i+1] = sig
	}
	sigs = append(sigs, rb)
	mtx.TxIn[0].Witness = wire.TxWitness(sigs)
	fmt.Println(mtx.TxIn[0].Witness.SerializeSize())

	for _, v := range mtx.TxIn[0].Witness {
		fmt.Printf("%x ", v)
	}
	fmt.Println()

	var buf bytes.Buffer
	err = mtx.BtcEncode(&buf, wire.ProtocolVersion, wire.LatestEncoding)
	if err != nil {
		t.Fatal(err)
	}

	vm, err := txscript.NewEngine(script, mtx,0, txscript.StandardVerifyFlags,nil, nil, btcutil.SatoshiPerBitcoin)
	if err != nil {
		t.Fatal(err)
	}
	//info, err := vm.DisasmPC()
	//if err != nil {
	//	t.Fatal(err)
	//}
	//fmt.Println(info)
	err = vm.Execute()
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("tx: %s\n", hex.EncodeToString(buf.Bytes()))

}

func TestSS(t *testing.T) {
	//locktx := "01000000000101cc51b096b532718ad910c3aa5aebf2a43233708cb4726c0e5dc522bfb80f9d360000000000ffffffff0540a0dfb2000000002200200a618b712d918bb1ba59b737c2a37b40d557374754ef2575ce41d08d5f782df940d64477000000002200200a618b712d918bb1ba59b737c2a37b40d557374754ef2575ce41d08d5f782df900ca9a3b000000002200206ffd48f065e61dd8e1091f1aa9819cf5b45692d68e1ce3691aaf69014e267155e04c18100000000022002014b288dca5d59caa8868d1668c97c971e58ab3ccf10534ac567ea51aa8aba299c0366aac010000002200202122f4719add322f4d727f48379f8a8ba36a40ec4473fd99a2fdcfd89a16e048040047304402204341fbd204ea7c6715db7863aa00d7b8333a6598113deb38134bd7d5619d3dab0220047126fc26778e15b32129d7d55f857f0a2f8c5ded50f0910c1fedecc2fde7dc014730440220694d8b364c41b938492e8953e5d7c697ae054d798eeeda3e0f33753d8487d75902200f157a494d7015c3136b63b4accf351cd7f98f437cf67873e48bf85cc4d0cd7a01695221022dfa322241a4946b9ead36ab9c8c55bd4c4340a1290b5bf71d23a695aeb1240a21034d82610a17c332852205e063c64fee21a77fabc7ac0e6d7ada2a820922c9a5dc2103c96d495bfdd5ba4145e3e046fee45e84a8a48ad05bd8dbb395c011a32cf9f88053ae00000000"
	//lb, _ := hex.DecodeString(locktx)
	//mtxLock := wire.NewMsgTx(1)
	//mtxLock.BtcDecode(bytes.NewBuffer(lb), wire.ProtocolVersion, wire.LatestEncoding)
	//
	//ex := "01000000000101891b2106801ef96bbfbd7d064bcca26a50022a550233347a070ef8f37cee9dd40400000000ffffffff0740717759000000002200203eb5062a0b0850b23a599425289a091c374ca934101d03144f060c5b46a979be40717759000000002200206ffd48f065e61dd8e1091f1aa9819cf5b45692d68e1ce3691aaf69014e26715500ca9a3b00000000220020701a8d401c84fb13e6baf169d59684e17abd9fa216c8cc5b9fc63d622ff8c58d00ca9a3b000000002200200a618b712d918bb1ba59b737c2a37b40d557374754ef2575ce41d08d5f782df900ca9a3b000000002200203eb5062a0b0850b23a599425289a091c374ca934101d03144f060c5b46a979be004791130000000022002014b288dca5d59caa8868d1668c97c971e58ab3ccf10534ac567ea51aa8aba29940ec1833000000002200202122f4719add322f4d727f48379f8a8ba36a40ec4473fd99a2fdcfd89a16e048040047304402206b8213ea25faa7023176fffc8f3151c80a3fa7ff95e37c0faf5ad2a800b65591022075396d01ee61d0280120c9cc1b580172821f539c0bd6925717b509a79dc1071b01473044022001f1139d32223cfc39c73d502b36d9a0d249415af6dcce17993ae0debcd44f4d02201350bef52485ff8e5327a6fb36662667375add614067cf3489f1b9b69e31dc8c01695221022dfa322241a4946b9ead36ab9c8c55bd4c4340a1290b5bf71d23a695aeb1240a21034d82610a17c332852205e063c64fee21a77fabc7ac0e6d7ada2a820922c9a5dc2103c96d495bfdd5ba4145e3e046fee45e84a8a48ad05bd8dbb395c011a32cf9f88053ae00000000"
	//exb, _ := hex.DecodeString(ex)
	//mtxEx := wire.NewMsgTx(1)
	//mtxEx.BtcDecode(bytes.NewBuffer(exb), wire.ProtocolVersion, wire.LatestEncoding)
	//
	//for _, v := range mtxEx.TxIn[0].Witness {
	//	fmt.Printf("%x ", v)
	//}
	//fmt.Println()
	//
	//str, _ := txscript.DisasmString(mtxEx.TxIn[0].Witness[3])
	//fmt.Println(str)
	//
	//vm, err := txscript.NewEngine(mtxLock.TxOut[4].PkScript, mtxEx,0, txscript.StandardVerifyFlags,nil,nil, mtxLock.TxOut[4].Value)
	//if err != nil {
	//	t.Fatal(err)
	//}
	//
	//err = vm.Execute()
	//if err != nil {
	//	t.Fatal(err)
	//}
	fmt.Println(hex.EncodeToString(base58.Decode("mjEoyyCPsLzJ23xMX6Mti13zMyN36kzn57")))
}
