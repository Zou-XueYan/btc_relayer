package signer

import (
	"encoding/hex"
	"fmt"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcutil"
	"strings"
	"testing"
)

func TestGetSigScripts(t *testing.T) {
	signer, _, _, err := buildStaff()
	if err != nil {
		t.Fatal(err)
	}

	map1 := make(map[string][]byte)
	privks := getPrivkMap()
	cnt := 0
	for _, pk := range getPubKeys() {
		apk, err := btcutil.NewAddressPubKey(pk, &chaincfg.TestNet3Params)
		if err != nil {
			t.Fatal(err)
		}

		pkstr := hex.EncodeToString(apk.PubKey().SerializeCompressed())
		sigs, err := signer.SignWithOnePrivk1(privks[apk.EncodeAddress()])
		if err != nil {
			t.Fatal(err)
		}

		map1[pkstr] = sigs[0]

		if cnt++; cnt == 5 {
			break
		}
	}
	map2 := make(map[uint64]map[string][]byte)
	map2[0] = map1
	redeem, _ := signer.buildScript(getPubKeys())
	res, err := GetSigScripts(map2, redeem, &chaincfg.TestNet3Params)
	if err != nil {
		t.Fatalf("error happen: %v", err)
	}
	err = signer.Sign()
	if err != nil {
		t.Fatalf("Failed to sign: %v", err)
	}

	str1, err := txscript.DisasmString(signer.Mtx.TxIn[0].SignatureScript)
	if err != nil {
		t.Fatalf("Failed to disasm: %v", err)
	}

	str2, err := txscript.DisasmString(res[0])
	if err != nil {
		t.Fatalf("Failed to disasm: %v", err)
	}

	fmt.Println(str1)
	fmt.Println(str2)

	strs := strings.Split(str1, " ")
	fmt.Println("len of str1 is", len(strs))
	for _, s := range strs {
		fmt.Println(s)
	}
	fmt.Println()

	strs = strings.Split(str2, " ")
	fmt.Println("len of str2 is", len(strs))
	for _, s := range strs {
		fmt.Println(s)
	}

	fmt.Println()
	if str1 != str2 {
		t.Fatal("not equal")
	}
}

func TestVerifySigs(t *testing.T) {
	signer, _, redeemTx, err := buildStaff()
	if err != nil {
		t.Fatal(err)
	}
	redeem, _ := signer.buildScript(getPubKeys())
	sig, err := signer.SignWithOnePrivk(priv1)
	if err != nil {
		t.Fatal(err)
	}

	err = VerifySigs(sig, addr1, redeem, redeemTx, &chaincfg.TestNet3Params)
	if err != nil {
		t.Fatal(err)
	}

	sig, err = signer.SignWithOnePrivk(priv2)
	if err != nil {
		t.Fatal(err)
	}

	err = VerifySigs(sig, addr1, redeem, redeemTx, &chaincfg.TestNet3Params)
	if err == nil {
		t.Fatal("neg case failed")
	}
}

