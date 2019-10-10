package signer

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/Zou-XueYan/btc_relayer/observer"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcutil/base58"
	"strings"
	"testing"
	"time"
)

var rawTx string = "010000000111101696001639891ae61d717c1605384a6dd50e143c93263b94820ff83345970000000000ffffffff02e8030000000000001976a91428d2e8cee08857f569e5a1b147c5d5e87339e08188ac703a0f000000000017a91487a9652e9b396545598c0fc72cb5a98848bf93d38700000000"
var prevRawTx string = "0100000001f8f9f2b01451dad75aefdfd677cda8bfc2315d0de28ba0ef5e615af5a612fcc800000000fd600200483045022100c3a08b37888fe1c96aa49f1beb0741f61b4ef2dc7a38b1d7eb56ad31194bd008022035995931503589787a5cd123265298a1e55a7fbd234bdad2507d2dbf687c5b7f01483045022100e1574ee0e942d363f0019333d8a623d20145a7b1f157146d5862ef5e57b039d8022005b4f3f382860ea011dd96f4a17c5bc310dfca0a186ebf6f478942f42db975ac01483045022100f5f4375b955606f1669d3486c716d4025b31fb34b81390e9660a7b8d0688c639022071adf11f06e1490f0bd3d39d2808dae74e7158021ee8dec031fe6061eda2e9fe0147304402203b8305dae5cf251669b60f0ede97af3790cf131818a1fc44eacc795650ebafc8022007ba588e4e672ac43f581e1c0fa5c5c7cfd6a9169c051a335485f0d0f992c98901483045022100fddec2b4545c21303432c3312c208854fa51bd93e8305419b20cf1975f816e70022071614cb4189f133c2dba2e941b00470bdb06cd574964adbe34642b0f3514debe014cf15521023ac710e73e1410718530b2686ce47f12fa3c470a9eb6085976b70b01c64c9f732102c9dc4d8f419e325bbef0fe039ed6feaf2079a2ef7b27336ddb79be2ea6e334bf2102eac939f2f0873894d8bf0ef2f8bbdd32e4290cbf9632b59dee743529c0af9e802103378b4a3854c88cca8bfed2558e9875a144521df4a75ab37a206049ccef12be692103495a81957ce65e3359c114e6c2fe9f97568be491e3f24d6fa66cc542e360cd662102d43e29299971e802160a92cfcd4037e8ae83fb8f6af138684bebdc5686f3b9db21031e415c04cbc9b81fbee6e04d8c902e8f61109a2c9883a959ba528c52698c055a57aeffffffff01282300000000000017a91487a9652e9b396545598c0fc72cb5a98848bf93d38700000000"

func buildStaff() (*MultiSigner, *wire.MsgTx, *wire.MsgTx, error) {
	prevRawTxB, err := hex.DecodeString(prevRawTx)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Failed to decode prev tx: %v", err)
	}
	prevTx := wire.NewMsgTx(wire.TxVersion)
	err = prevTx.BtcDecode(bytes.NewBuffer(prevRawTxB), wire.ProtocolVersion, wire.LatestEncoding)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Failed to decode prev tx: %v", err)
	}

	redeemTx := wire.NewMsgTx(wire.TxVersion)
	txHash := prevTx.TxHash()
	prevOut := wire.NewOutPoint(&txHash, 0)
	txIn := wire.NewTxIn(prevOut, nil, nil)
	redeemTx.AddTxIn(txIn)

	out := wire.NewTxOut(8e3, prevTx.TxOut[0].PkScript)
	redeemTx.AddTxOut(out)
	var buf1 bytes.Buffer
	err = redeemTx.BtcEncode(&buf1, wire.ProtocolVersion, wire.LatestEncoding)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("%v", err)
	}

	signer, err := NewMultiSigner(buf1.Bytes(), &chaincfg.TestNet3Params, [][]byte{prevTx.TxOut[0].PkScript}, 5)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Failed to new signer: %v", err)
	}

	return signer, prevTx, redeemTx, nil
}

func TestMultiSigner_Sign(t *testing.T) {
	signer, prevTx, _, err := buildStaff()
	if err != nil {
		t.Fatal(err)
	}

	err = signer.Sign()
	if err != nil {
		t.Fatalf("Failed to sign: %v", err)
	}

	str, err := txscript.DisasmString(signer.Mtx.TxIn[0].SignatureScript)
	if err != nil {
		t.Fatalf("Failed to disasm: %v", err)
	}
	fmt.Println(str)

	strs := strings.Split(str, " ")
	fmt.Println("len of strs is", len(strs))
	for _, s := range strs {
		fmt.Println(s)
	}

	var buf bytes.Buffer
	err = signer.Mtx.BtcEncode(&buf, wire.ProtocolVersion, wire.LatestEncoding)
	if err != nil {
		t.Fatalf("Failed to encode: %v", err)
	}

	fmt.Printf("signed tx: %x\n", buf.Bytes())

	fmt.Printf("tx len: %d\n", len(buf.Bytes()))
	fmt.Printf("sig len: %d\n", len(signer.Mtx.TxIn[0].SignatureScript))

	script, _ := signer.buildScript(getPubKeys())
	fmt.Printf("redeem len: %d\n", len(script))

	fmt.Printf("out[0] len: %d\n", signer.Mtx.TxOut[0].SerializeSize())
	fmt.Printf("pks len: %d\n", len(signer.Mtx.TxOut[0].PkScript))
	//fmt.Println(1 + 5 * (1 + 72) + 1 + 1 + 7 * (1 + 33) + 1 + 1)

	flags := txscript.ScriptBip16 | txscript.ScriptVerifyDERSignatures |
		txscript.ScriptStrictMultiSig |
		txscript.ScriptDiscourageUpgradableNops
	vm, err := txscript.NewEngine(prevTx.TxOut[0].PkScript, signer.Mtx, 0,
		flags, nil, nil, -1)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if err := vm.Execute(); err != nil {
		t.Fatalf("%v", err)
	}
}

func TestSigningMachine_Signing(t *testing.T) {
	m, err := NewSigningMachine("../conf.json")
	if err != nil {
		t.Fatalf("Failed to new m: %v", err)
	}

	c := make(chan *observer.FromAllianceItem, 10)
	go m.Signing(c)
	go func() {
		for item := range m.broadcasting {
			fmt.Printf("Signed tx: %s\nTx data: %s\n", item.Txid, item.Data)
		}
	}()

	time.Sleep(3 * time.Second)
	c <- &observer.FromAllianceItem{
		Tx: rawTx,
	}

	time.Sleep(3 * time.Second)
}

func TestSigningMachine_Broadcasting(t *testing.T) {
	m, err := NewSigningMachine("../conf.json")
	if err != nil {
		t.Fatalf("Failed to new m: %v", err)
	}

	c := make(chan *observer.FromAllianceItem, 10)
	go m.Signing(c)
	go m.Broadcasting()

	time.Sleep(3 * time.Second)
	c <- &observer.FromAllianceItem{
		Tx: "0100000001b40d4dbd1330b570a1eaa9100318071614ec43b6d32d7b59a18bbf734c5918130000000000ffffffff020a000000000000001976a91428d2e8cee08857f569e5a1b147c5d5e87339e08188ac0e310c000000000017a91487a9652e9b396545598c0fc72cb5a98848bf93d38700000000",
	}

	time.Sleep(3 * time.Second)
}

func TestOnePrivkToSign(t *testing.T) {
	signer, _, _, err := buildStaff()
	if err != nil {
		t.Fatal(err)
	}

	sigs, err := signer.SignWithOnePrivk(priv2)
	if err != nil {
		t.Fatalf("Faild to sign: %v", err)
	}

	fmt.Println("len is", len(sigs))
	// 3044022055b76cbd2755c27652a566465a6646e6b3f17e688bc20556434bb3f69d261e2e02206f23363b9eb4fc0d925f8f9580e0e119ca5c92a05473cdb61f1ec905d0104f8301
	fmt.Printf("%x\n", sigs[0])
	fmt.Printf("%x\n", signer.Mtx.TxIn[0].SignatureScript)
	redeem, err := signer.buildScript(getPubKeys())
	if err != nil {
		t.Fatalf("%v", err)
	}
	sh, err := btcutil.NewAddressScriptHash(redeem, signer.NetParam)
	if err != nil {
		t.Fatalf("%v", err)
	}

	fmt.Println(sh.EncodeAddress())
}

func TestCheckSig(t *testing.T) {
	signer, _, redeemTx, err := buildStaff()
	if err != nil {
		t.Fatal(err)
	}
	pks := getPubKeys()
	redeem, _ := signer.buildScript(pks)

	fmt.Println(hex.EncodeToString(redeem))
	fmt.Println(string(redeem))

	hash, err := txscript.CalcSignatureHash(redeem, txscript.SigHashAll, redeemTx, 0)
	if err != nil {
		t.Fatal(err)
	}

	_, pubk1 := btcec.PrivKeyFromBytes(btcec.S256(), base58.Decode(priv1))

	sigs, err := signer.SignWithOnePrivk(priv1)
	if err != nil {
		t.Fatal(err)
	}
	s, err := btcec.ParseDERSignature(sigs[0][:len(sigs[0])-1], btcec.S256())
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(s.Verify(hash, pubk1))
}

func TestMakeSigScript(t *testing.T) {
	signer, _, _, err := buildStaff()
	if err != nil {
		t.Fatal(err)
	}

	pks := getPubKeys()
	redeem, _ := signer.buildScript(pks)
	fmt.Printf("%x\n", redeem)
	cls, addrs, r, err := txscript.ExtractPkScriptAddrs(redeem, &chaincfg.TestNet3Params)
	fmt.Printf("class is %s, length is %d, require is %d\n", cls.String(), len(addrs), r)
}
