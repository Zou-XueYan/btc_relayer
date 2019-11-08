package observer

import (
	"bytes"
	"encoding/hex"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/wire"
	"github.com/ontio/btcrelayer/log"
	sdk "github.com/ontio/multi-chain-go-sdk"
	"time"
)

type BtcObConfig struct {
	FirstN        uint32
	LoopWaitTime  int64
	Confirmations uint32
}

type BtcObserver struct {
	cli      *RestCli
	NetParam *chaincfg.Params
	conf     *BtcObConfig
}

func NewBtcObserver(addr, user, pwd string, param *chaincfg.Params, conf *BtcObConfig) *BtcObserver {
	var observer BtcObserver
	observer.cli = NewRestCli(addr, user, pwd)
	observer.NetParam = param
	observer.conf = conf

	return &observer
}

func (observer *BtcObserver) Listen(relaying chan *CrossChainItem) {
	top := btcCheckPoints[observer.NetParam.Name].Height
	log.Infof("[BtcObserver] get start height %d from checkpoint, check once %d seconds", top, observer.conf.LoopWaitTime)

	tick := time.NewTicker(time.Duration(observer.conf.LoopWaitTime) * time.Second)
	for {
		select {
		case <-tick.C:
			newTop, hash, err := observer.cli.GetCurrentHeightAndHash()
			if err != nil {
				log.Errorf("[BtcObserver] GetCurrentHeightAndHash failed, loop continue: %v", err)
				continue
			}
			log.Tracef("[BtcObserver] start observing from block %s at height %d", hash, newTop)

			if newTop <= top { // Prevent rollback
				log.Tracef("[BtcObserver] height not enough: now is %d, prev is %d", newTop, top)
				continue
			}
			for h := top - observer.conf.Confirmations + 2; h <= newTop - observer.conf.Confirmations + 1; h++ { // TODO: double check?
				txns, hash, err := observer.cli.GetTxsInBlockByHeight(h)
				if err != nil {
					log.Errorf("[BtcObserver] failed to check block %s, retry after 10 sec: %v", hash, err)
					h--
					time.Sleep(time.Second * 10)
					continue
				}
				count := observer.SearchTxInBlock(txns, h, relaying)
				if count > 0 {
					log.Infof("[BtcObserver] %d tx found in block(height:%d) %s", count, h, hash)
				}
			}

			top = newTop
		}
	}
}

func (observer *BtcObserver) SearchTxInBlock(txns []*wire.MsgTx, height uint32, relaying chan *CrossChainItem) int {
	count := 0
	for i := 0; i < len(txns); i++ {
		if !checkIfCrossChainTx(txns[i], observer.NetParam) {
			continue
		}
		var buf bytes.Buffer
		err := txns[i].BtcEncode(&buf, wire.ProtocolVersion, wire.LatestEncoding)
		if err != nil {
			log.Errorf("[SearchTxInBlock] failed to encode transaction: %v", err)
			continue
		}
		txid := txns[i].TxHash()
		proof, err := observer.cli.GetProof([]string{txid.String()}) // TODO: continue 的处理, 区分网络问题和get不存在问题
		if err != nil {
			log.Errorf("[SearchTxInBlock] failed to get proof for tx %s", txid.String())
			i--
			continue
		}
		proofBytes, _ := hex.DecodeString(proof)
		relaying <- &CrossChainItem{
			Proof:  proofBytes,
			Tx:     buf.Bytes(),
			Height: height,
			Txid:   txid,
		}
		log.Infof("[SearchTxInBlock] eligible transaction found, txid: %s", txid.String())
		count++
	}

	return count
}

type AllianceObConfig struct {
	FirstN       uint32
	LoopWaitTime int64
	WatchingKey  string
}

type AllianceObserver struct {
	allia *sdk.MultiChainSdk
	conf  *AllianceObConfig
}

func NewAllianceObserver(allia *sdk.MultiChainSdk, conf *AllianceObConfig) *AllianceObserver {
	return &AllianceObserver{
		allia: allia,
		conf:  conf,
	}
}

func (observer *AllianceObserver) Listen(collecting chan *FromAllianceItem) {
START:
	top, err := observer.allia.GetCurrentBlockHeight()
	if err != nil {
		log.Errorf("[AllianceObserver] failed to get current height: %v", err)
		time.Sleep(time.Second * 30)
		goto START
	}

	num := observer.conf.FirstN
	if top < num {
		num = top
	}
	h := top - num + 1
	count := 0
	log.Infof("[AllianceObserver] first to start Listen(), check %d blocks from top %d", num, top)
	for h <= top {
		events, err := observer.allia.GetSmartContractEventByBlock(h)
		if err != nil {
			log.Errorf("[AllianceObserver] GetSmartContractEventByBlock failed, retry after 10 sec: %v", err)
			time.Sleep(time.Second * 10)
			continue //TODO:
		}

		for _, e := range events {
			for _, n := range e.Notify {
				states, ok := n.States.([]interface{})
				if !ok {
					continue
				}

				name, ok := states[0].(string)
				if ok && name == observer.conf.WatchingKey {
					tx, ok := states[1].(string)
					if !ok {
						continue
					}
					collecting <- &FromAllianceItem{
						Tx: tx,
					}
					count++
					log.Infof("[AllianceObserver] captured: %s when height is %d", tx, h)
				}
			}
		}
		h++
	}
	log.Infof("[AllianceObserver] total %d transactions captured from %d blocks", count, observer.conf.FirstN)

	log.Infof("[AllianceObserver] next, check once %d seconds", observer.conf.LoopWaitTime)
	for {
		time.Sleep(time.Second * time.Duration(observer.conf.LoopWaitTime))
		count = 0
		newTop, err := observer.allia.GetCurrentBlockHeight()
		if err != nil {
			log.Errorf("[AllianceObserver] failed to get current height, retry after 10 sec: %v", err)
			continue
		}
		log.Tracef("[AllianceObserver] start observing from height %d", newTop)

		if newTop-top == 0 {
			//log.Infof("[AllianceObserver] height not change: height is %d", newTop)
			continue
		}

		h := top + 1
		for h <= newTop {
			events, err := observer.allia.GetSmartContractEventByBlock(h)
			if err != nil {
				log.Errorf("[AllianceObserver] GetSmartContractEventByBlock failed, retry after 10 sec: %v", err)
				time.Sleep(time.Second * 10)
				continue
			}

			for _, e := range events {
				for _, n := range e.Notify {
					states, ok := n.States.([]interface{})
					if !ok {
						continue
					}
					name, ok := states[0].(string)
					if ok && name == observer.conf.WatchingKey {
						tx := states[1].(string)
						collecting <- &FromAllianceItem{
							Tx: tx,
						}
						count++
						log.Infof("[AllianceObserver] captured: %s when height is %d", tx, h)
					}
				}
			}

			h++
		}
		if count > 0 {
			log.Infof("[AllianceObserver] total %d transactions captured this time", count)
		}
		top = newTop
	}
}
