package signer

import (
	"encoding/hex"
	"fmt"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

func GetSigScripts(sigMap map[uint64]map[string][]byte, redeem []byte, params *chaincfg.Params) ([][]byte, error) {
	cls, addrs, require, err := txscript.ExtractPkScriptAddrs(redeem, params)
	if err != nil {
		return nil, fmt.Errorf("failed to extract pkscript addrs: %v", err)
	}
	if cls.String() != "multisig" {
		return nil, fmt.Errorf("wrong class of redeem: %s", cls.String())
	}

	sigScripts := make([][]byte, 0)
	pubks := make([]string, 0)
	for _, addr := range addrs {
		pubks = append(pubks, hex.EncodeToString(addr.(*btcutil.AddressPubKey).PubKey().SerializeCompressed()))
	}

	for idx, m := range sigMap {
		builder := txscript.NewScriptBuilder().AddOp(txscript.OP_FALSE)
		cnt := 0
		for _, pubk := range pubks {
			val, ok := m[pubk]
			if !ok {
				continue
			}
			builder.AddData(val)
			cnt++
		}
		if cnt != require {
			return nil, fmt.Errorf("wrong number of added sig, should be %d not %d", require, cnt)
		}
		builder.AddData(redeem)
		script, err := builder.Script()
		if err != nil {
			return nil, fmt.Errorf("failed to build sigscript for input %d: %v", idx, err)
		}
		sigScripts = append(sigScripts, script)
	}

	return sigScripts, nil
}

func VerifySigs(sigs [][]byte, addr string, redeem []byte, tx *wire.MsgTx, params *chaincfg.Params) error {
	cls, addrs, _, err := txscript.ExtractPkScriptAddrs(redeem, params)
	if err != nil {
		return fmt.Errorf("failed to extract pkscript addrs: %v", err)
	}
	if cls.String() != "multisig" {
		return fmt.Errorf("wrong class of redeem: %s", cls.String())
	}
	if len(sigs) != len(tx.TxIn) {
		return fmt.Errorf("not enough sig, only %d sigs but %d required", len(sigs), len(tx.TxIn))
	}

	var signerAddr btcutil.Address = nil
	for _, a := range addrs {
		if a.EncodeAddress() == addr {
			signerAddr = a
		}
	}
	if signerAddr == nil {
		return fmt.Errorf("address %s not found in redeem script", addr)
	}

	for i, sig := range sigs {
		if len(sig) < 1 {
			return fmt.Errorf("length of no.%d sig is less than 1", i)
		}
		tSig := sig[:len(sig)-1]
		pSig, err := btcec.ParseDERSignature(tSig, btcec.S256())
		if err != nil {
			return fmt.Errorf("failed to parse no.%d sig: %v", i, err)
		}

		hash, err := txscript.CalcSignatureHash(redeem, txscript.SigHashType(sig[len(sig)-1]), tx, i)
		if err != nil {
			return fmt.Errorf("failed to calculate sig hash: %v", err)
		}

		if !pSig.Verify(hash, signerAddr.(*btcutil.AddressPubKey).PubKey()) {
			return fmt.Errorf("verify no.%d sig and not pass", i)
		}
	}

	return nil
}
