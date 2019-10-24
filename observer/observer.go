package observer

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/Zou-XueYan/btc_relayer/log"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/wire"
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
START:
	top, hash, err := observer.cli.GetCurrentHeightAndHash()
	if err != nil {
		log.Errorf("[BtcObserver] retry per 30 sec: %v", err)
		time.Sleep(time.Second * 30)
		goto START
	}

	// first to start, check FirstN blocks from top
	log.Infof("[BtcObserver] first to start Listen(), check %d blocks from top %d", observer.conf.FirstN, top)
	num := observer.conf.FirstN
	if num > top {
		num = top
	}
	h := top
	cnt := 0
	for num > 0 {
		txns, prev, err := observer.cli.GetTxsInBlock(hash)
		if err != nil {
			log.Errorf("[BtcObserver] failed to check block %s: %v", hash, err)
			time.Sleep(time.Second * 10)
			continue
		}
		count, err := observer.SearchTxInBlock(txns, h, relaying)
		if err != nil {
			log.Errorf("[BtcObserver] failed to search in block %s, retry after 10 sec: %v", hash, err)
			time.Sleep(time.Second * 10)
			continue
		}
		if count > 0 {
			log.Infof("[BtcObserver] %d tx found in block(height:%d) %s", count, h, hash)
		}

		cnt += count
		num--
		h--
		hash = prev
	}
	log.Infof("[BtcObserver] %d tx found from top(height:%d) to block %d", cnt, top, h)

	log.Infof("[BtcObserver] next, check once %d seconds", observer.conf.LoopWaitTime)
	for {
		time.Sleep(time.Duration(observer.conf.LoopWaitTime) * time.Second)
		newTop, hash, err := observer.cli.GetCurrentHeightAndHash()
		if err != nil {
			log.Errorf("[BtcObserver] GetCurrentHeightAndHash failed, loop continue: %v", err)
			continue
		}
		log.Tracef("[BtcObserver] start observing from block %s at height %d", hash, newTop)

		num := newTop - top
		if num <= observer.conf.Confirmations-1 { // Prevent rollback
			log.Infof("[BtcObserver] height not enough: now is %d, prev is %d", newTop, top)
			continue
		}
		h := newTop
		for num+observer.conf.Confirmations > 0 {
			txns, prev, err := observer.cli.GetTxsInBlock(hash)
			if err != nil {
				log.Errorf("[BtcObserver] failed to check block %s, retry after 10 sec: %v", hash, err)
				time.Sleep(time.Second * 10)
				continue
			}

			count, err := observer.SearchTxInBlock(txns, h, relaying)
			if err != nil {
				log.Errorf("[BtcObserver] failed to search in block %s, retry after 10 sec: %v", hash, err)
				time.Sleep(time.Second * 10)
				continue
			}

			if count > 0 {
				log.Infof("[BtcObserver] %d tx found in block(height:%d) %s", count, h, hash)
			}
			num--
			h--
			hash = prev
		}

		top = newTop
	}
}

func (observer *BtcObserver) SearchTxInBlock(txns []*wire.MsgTx, height int32, relaying chan *CrossChainItem) (int, error) {
	count := 0
	for _, tx := range txns {
		if !checkIfCrossChainTx(tx, observer.NetParam) {
			continue
		}
		var buf bytes.Buffer
		err := tx.BtcEncode(&buf, wire.ProtocolVersion, wire.LatestEncoding)
		if err != nil {
			log.Errorf("[SearchTxInBlock] failed to encode transaction: %v", err)
			continue
		}

		proof, err := observer.cli.GetProof([]string{tx.TxHash().String()})
		if err != nil {
			log.Errorf("[SearchTxInBlock] failed to get proof for tx %s", tx.TxHash().String())
			continue
		}
		proofBytes, err := hex.DecodeString(proof)
		if err != nil {
			log.Errorf("[SearchTxInBlock] failed to decode proof in hex: %v", err)
			continue
		}

		fmt.Println(hex.EncodeToString(buf.Bytes()))
		relaying <- &CrossChainItem{
			Proof:  proofBytes,
			Tx:     buf.Bytes(),
			Height: height,
			Txid:   tx.TxHash(),
		}
		log.Infof("[SearchTxInBlock] eligible transaction found, txid: %s", tx.TxHash().String())
		count++
	}

	return count, nil
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

		if newTop - top == 0 {
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
		if count > 0 {
			log.Infof("[AllianceObserver] total %d transactions captured this time", count)
		}
		top = newTop
	}
}
