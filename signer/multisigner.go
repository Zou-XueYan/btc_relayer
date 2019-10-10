package signer

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/Zou-XueYan/btc_relayer"
	"github.com/Zou-XueYan/btc_relayer/log"
	"github.com/Zou-XueYan/btc_relayer/observer"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcutil/base58"
)

type SigningMachine struct {
	signer       *MultiSigner
	cli          *observer.RestCli
	netParam     *chaincfg.Params
	broadcasting chan *SignedTxItem
}

func NewSigningMachine(confFile string) (*SigningMachine, error) {
	conf, err := btc_relayer.NewBtcConfig(confFile)
	if err != nil {
		return nil, err
	}

	var param *chaincfg.Params
	switch conf.NetType {
	case "test":
		param = &chaincfg.TestNet3Params
	case "sim":
		param = &chaincfg.SimNetParams
	case "regtest":
		param = &chaincfg.RegressionNetParams
	default:
		param = &chaincfg.MainNetParams
	}

	return &SigningMachine{
		signer:       &MultiSigner{},
		cli:          observer.NewRestCli(conf.BtcJsonRpcAddress, conf.User, conf.Pwd),
		netParam:     param,
		broadcasting: make(chan *SignedTxItem, 10),
	}, nil
}

func (m *SigningMachine) getPrevPubkScripts(ins []*wire.TxIn) (pubks [][]byte, err error) {
	for i, in := range ins {
		s, err := m.cli.GetScriptPubKey(in.PreviousOutPoint.Hash.String(), in.PreviousOutPoint.Index)
		if err != nil {
			return nil, fmt.Errorf("failed to get no.%d input's scriptPubKey: %v", i, err)
		}

		sb, err := hex.DecodeString(s)
		if err != nil {
			return nil, fmt.Errorf("failed to decode no.%d input's hex: %v", i, err)
		}

		pubks = append(pubks, sb)
	}

	return pubks, nil
}

func (m *SigningMachine) Signing(collecting chan *observer.FromAllianceItem) {
	log.Infof("[SigningMachine] start signing")
	for item := range collecting {
		txb, err := hex.DecodeString(item.Tx)
		if err != nil {
			log.Errorf("[SigningMachine] failed to decode hex string: %v", err)
			continue
		}

		unsigned := wire.NewMsgTx(wire.TxVersion)
		err = unsigned.BtcDecode(bytes.NewBuffer(txb), wire.ProtocolVersion, wire.LatestEncoding)
		if err != nil {
			log.Errorf("[SigningMachine] failed to btcdecode transaction in bytes: %v", err)
			continue
		}

		pubks, err := m.getPrevPubkScripts(unsigned.TxIn)
		if err != nil {
			log.Errorf("[SigningMachine] failed to getPrevPubkScripts: %v", err)
			continue
		}

		signer, err := NewMultiSigner(txb, m.netParam, pubks, 5)
		if err != nil {
			log.Errorf("[SigningMachine] failed to NewMultiSigner: %v", err)
			continue
		}
		m.signer = signer
		err = m.signer.Sign()
		if err != nil || !m.signer.IsSigned {
			log.Errorf("[SigningMachine] failed to sign transaction: %v", err)
			continue
		}

		var buf bytes.Buffer
		err = m.signer.Mtx.BtcEncode(&buf, wire.ProtocolVersion, wire.LatestEncoding)
		if err != nil {
			log.Errorf("[SigningMachine] failed to encode transaction: %v", err)
			continue
		}

		txid := m.signer.Mtx.TxHash().String()
		m.broadcasting <- &SignedTxItem{
			Txid: txid,
			Data: hex.EncodeToString(buf.Bytes()),
		}
		log.Infof("[SigningMachine] send %s to broadcast", txid)
	}
}

func (m *SigningMachine) Broadcasting() {
	log.Infof("[SigningMachine] start broadcasting")
	for item := range m.broadcasting {
		err := m.cli.BroadcastTx(item.Txid, item.Data)
		if err != nil {
			log.Errorf("[SigningMachine] failed to broadcast tx: %v", err)
			continue
		}
		log.Infof("[SigningMachine] already broadcast tx: %s", item.Txid)
	}
}

type MultiSigner struct {
	Mtx           *wire.MsgTx
	IsSigned      bool
	NetParam      *chaincfg.Params
	PrevPkScripts [][]byte
	Require       int
}

func NewMultiSigner(rawTx []byte, net *chaincfg.Params, pks [][]byte, require int) (*MultiSigner, error) {
	if len(rawTx) == 0 || rawTx == nil {
		return nil, errors.New("raw tx can't be nil")
	}
	mtx := wire.NewMsgTx(wire.TxVersion)
	err := mtx.BtcDecode(bytes.NewBuffer(rawTx), wire.ProtocolVersion, wire.LatestEncoding)
	if err != nil {
		return nil, fmt.Errorf("Failed to decode raw tx: %v", err)
	}
	return &MultiSigner{
		Mtx:           mtx,
		IsSigned:      false,
		NetParam:      net,
		PrevPkScripts: pks,
		Require:       require,
	}, nil
}

func (signer *MultiSigner) Sign() error {
	if signer.IsSigned {
		return errors.New("Already signed")
	}

	for i, _ := range signer.Mtx.TxIn {
		sig, err := txscript.SignTxOutput(signer.NetParam, signer.Mtx, i, signer.PrevPkScripts[i],
			txscript.SigHashAll, txscript.KeyClosure(lookUpOneKey), txscript.ScriptClosure(signer.lookUpScript), nil)
		if err != nil {
			return fmt.Errorf("Failed to sign tx's No.%d input: %v", i, err)
		}
		signer.Mtx.TxIn[i].SignatureScript = sig
	}
	signer.IsSigned = true

	return nil
}

func (signer *MultiSigner) buildScript(pubks [][]byte) ([]byte, error) {
	if len(pubks) == 0 || signer.Require <= 0 {
		return nil, errors.New("Wrong public keys or require number")
	}
	var addrPks []*btcutil.AddressPubKey
	for _, v := range pubks {
		addrPk, err := btcutil.NewAddressPubKey(v, signer.NetParam)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse address pubkey: %v", err)
		}
		addrPks = append(addrPks, addrPk)
	}
	s, err := txscript.MultiSigScript(addrPks, signer.Require)
	if err != nil {
		return nil, fmt.Errorf("Failed to build multi-sig script: %v", err)
	}

	return s, nil
}

func (signer *MultiSigner) lookUpScript(addr btcutil.Address) ([]byte, error) {
	script, err := signer.buildScript(getPubKeys())
	if err != nil {
		return nil, fmt.Errorf("Failed to build script: %v", err)
	}

	return script, nil
}

func (signer *MultiSigner) SignWithOnePrivk(privk string) ([][]byte, error) {
	privKey, _ := btcec.PrivKeyFromBytes(btcec.S256(), base58.Decode(privk))
	sigs, err := signer.SignWithOnePrivk1(privKey)
	if err != nil {
		return nil, err
	}
	return sigs, nil
}

func (signer *MultiSigner) SignWithOnePrivk1(privk *btcec.PrivateKey) ([][]byte, error) {
	s, err := signer.buildScript(getPubKeys())
	if err != nil {
		return nil, err
	}

	sigs := make([][]byte, 0)
	for i, _ := range signer.Mtx.TxIn {
		sig, err := txscript.RawTxInSignature(signer.Mtx, i, s, txscript.SigHashAll, privk)
		if err != nil {
			return nil, fmt.Errorf("Failed to sign tx's No.%d input: %v", i, err)
		}
		//signer.Mtx.TxIn[i].SignatureScript = sig
		sigs = append(sigs, sig)
	}

	return sigs, nil
}
