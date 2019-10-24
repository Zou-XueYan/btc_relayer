package btc_relayer

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/Zou-XueYan/btc_relayer/db"
	"github.com/Zou-XueYan/btc_relayer/log"
	"github.com/Zou-XueYan/btc_relayer/observer"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/wire"
	sdk "github.com/ontio/multi-chain-go-sdk"
	"io/ioutil"
	"os"
	"time"
)

type BtcRelayer struct {
	btcOb      *observer.BtcObserver
	alliaOb    *observer.AllianceObserver
	account    *sdk.Account
	relaying   chan *observer.CrossChainItem
	Collecting chan *observer.FromAllianceItem
	allia      *sdk.MultiChainSdk
	config     *BtcConfig
	cli        *observer.RestCli
	retryDB    *db.RetryDB
}

func NewBtcRelayer(confFile string) (*BtcRelayer, error) {
	conf, err := NewBtcConfig(confFile)
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

	allia := sdk.NewMultiChainSdk()
	allia.NewRpcClient().SetAddress(conf.AllianceJsonRpcAddress)
	acct, err := GetAccountByPassword(allia, conf.WalletFile, conf.WalletPwd)
	if err != nil {
		return nil, fmt.Errorf("GetAccountByPassword failed: %v", err)
	}

	if !checkIfExist(conf.RetryDBPath) {
		os.Mkdir(conf.RetryDBPath, os.ModePerm)
	}

	rdb, err := db.NewRetryDB(conf.RetryDBPath, conf.RetryTimes, conf.RetryDuration)
	if err != nil {
		return nil, fmt.Errorf("failed to new retry db: %v", err)
	}

	return &BtcRelayer{
		btcOb: observer.NewBtcObserver(conf.BtcJsonRpcAddress, conf.User, conf.Pwd, param, &observer.BtcObConfig{
			FirstN:        conf.BtcObFirstN,
			LoopWaitTime:  conf.BtcObLoopWaitTime,
			Confirmations: conf.BtcObConfirmations,
		}),
		alliaOb: observer.NewAllianceObserver(allia, &observer.AllianceObConfig{
			FirstN:       conf.AlliaObFirstN,
			LoopWaitTime: conf.AlliaObLoopWaitTime,
			WatchingKey:  conf.WatchingKey,
		}),
		account:    acct,
		relaying:   make(chan *observer.CrossChainItem, 10),
		Collecting: make(chan *observer.FromAllianceItem, 10),
		allia:      allia,
		config:     conf,
		cli:        observer.NewRestCli(conf.BtcJsonRpcAddress, conf.User, conf.Pwd),
		retryDB:    rdb,
	}, nil
}

func (relayer *BtcRelayer) BtcListen() {
	relayer.btcOb.Listen(relayer.relaying)
}

func (relayer *BtcRelayer) AllianceListen() {
	relayer.alliaOb.Listen(relayer.Collecting)
}

func (relayer *BtcRelayer) ReBroadcast() {
	log.Info("[BtcRelayer] rebroadcasting")
	tick := time.NewTicker(time.Duration(relayer.config.RetryDuration) * time.Minute)
	for {
		select {
		case <-tick.C:
			txArr, err := relayer.retryDB.GetAll()
			if err != nil {
				log.Debugf("[BtcRelayer] failed to get retry tx: %v", err)
				continue
			}
			for _, s := range txArr {
				txb, _ := hex.DecodeString(s)
				mtx := wire.NewMsgTx(wire.TxVersion)
				mtx.BtcDecode(bytes.NewBuffer(txb), wire.ProtocolVersion, wire.LatestEncoding)
				txid, err := relayer.cli.BroadcastTx(s)
				if err != nil {
					switch err.(type) {
					case observer.NeedToRetryErr:
						log.Debugf("[BtcRelayer] rebroadcast %s failed: %v", mtx.TxHash().String(), err)
					default:
						log.Infof("[BtcRelayer] no need to rebroadcast and delete this tx %s...%s: %v", s[:6], s[len(s)-6:], err)
						err = relayer.retryDB.Del(s)
						if err != nil {
							log.Errorf("[BtcRelayer] failed to delete tx %s(%s): %v", txid, s, err)
						}
					}
				} else {
					log.Infof("[BtcRelayer] rebroadcast and delete tx: %s", txid)
					err = relayer.retryDB.Del(s)
					if err != nil {
						log.Errorf("[BtcRelayer] failed to delete tx %s(%s): %v", txid, s, err)
					}
				}
			}
		}
	}
}

func (relayer *BtcRelayer) Broadcast() {
	log.Infof("[BtcRelayer] start broadcasting")
	for item := range relayer.Collecting {
		txid, err := relayer.cli.BroadcastTx(item.Tx)
		if err != nil {
			switch err.(type) {
			case observer.NeedToRetryErr:
				log.Infof("[BtcRelayer] need to rebroadcast this tx %s...%s: %v", item.Tx[:6], item.Tx[len(item.Tx)-6:], err)
				err = relayer.retryDB.Put(item.Tx)
				if err != nil {
					log.Errorf("[BtcRelayer] failed to put tx in db: %v", err)
				}
			default:
				log.Errorf("[BtcRelayer] failed to broadcast tx: %v", err)
			}
			continue
		}
		log.Infof("[BtcRelayer] broadcast tx: %s", txid)
	}
}

func (relayer *BtcRelayer) Relay() {
	for item := range relayer.relaying {
		log.Infof("[BtcRelayer] ralaying an item: txid: %s, height: %d", item.Txid, item.Height)
		txHash, err := relayer.allia.Native.Ccm.ImportOuterTransfer(observer.BTC_ID, item.Txid[:], item.Tx, uint32(item.Height),
			item.Proof, relayer.account.Address[:], relayer.account)
		if err != nil {
			log.Errorf("[BtcRelayer] invokeNativeContract error: %v", err)
			continue
		}
		log.Infof("[BtcRelayer] %s sent to alliance : txid: %s, height: %d", txHash.ToHexString(),
			item.Txid, item.Height)
	}
}

func (relayer *BtcRelayer) Print() {
	for item := range relayer.relaying {
		fmt.Printf("Item heigh: %d\t", item.Height)
		fmt.Printf("Item proof: %s\t", item.Proof)
		fmt.Printf("Item txid: %s\n", item.Txid)
	}
}

type BtcConfig struct {
	BtcJsonRpcAddress      string
	User                   string
	Pwd                    string
	AllianceJsonRpcAddress string
	NetType                string
	GasPrice               uint64
	GasLimit               uint64
	WalletFile             string
	WalletPwd              string
	BtcObFirstN            uint32 // BtcOb:
	BtcObLoopWaitTime      int64
	BtcObConfirmations     uint32
	AlliaObFirstN          uint32 // AlliaOb:
	AlliaObLoopWaitTime    int64
	WatchingKey            string
	RetryDuration          int
	RetryTimes             int
	RetryDBPath            string
}

func NewBtcConfig(file string) (*BtcConfig, error) {
	conf := &BtcConfig{}
	err := conf.Init(file)
	if err != nil {
		return conf, fmt.Errorf("failed to new config: %v", err)
	}
	return conf, nil
}

func (this *BtcConfig) Init(fileName string) error {
	err := this.loadConfig(fileName)
	if err != nil {
		return fmt.Errorf("loadConfig error:%s", err)
	}
	return nil
}

func (this *BtcConfig) loadConfig(fileName string) error {
	data, err := this.readFile(fileName)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, this)
	if err != nil {
		return fmt.Errorf("json.Unmarshal TestConfig:%s error:%s", data, err)
	}
	return nil
}

func (this *BtcConfig) readFile(fileName string) ([]byte, error) {
	file, err := os.OpenFile(fileName, os.O_RDONLY, 0666)
	if err != nil {
		return nil, fmt.Errorf("OpenFile %s error %s", fileName, err)
	}
	defer func() {
		err := file.Close()
		if err != nil {
			fmt.Println(fmt.Errorf("file %s close error %s", fileName, err))
		}
	}()
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("ioutil.ReadAll %s error %s", fileName, err)
	}
	return data, nil
}

func GetAccountByPassword(sdk *sdk.MultiChainSdk, path, pwd string) (*sdk.Account, error) {
	wallet, err := sdk.OpenWallet(path)
	if err != nil {
		return nil, fmt.Errorf("open wallet error: %v", err)
	}
	//pwd, err := password.GetPassword()
	//if err != nil {
	//	return nil, fmt.Errorf("getPassword error: %v", err)
	//}
	user, err := wallet.GetDefaultAccount([]byte(pwd))
	if err != nil {
		return nil, fmt.Errorf("getDefaultAccount error: %v", err)
	}
	return user, nil
}

func checkIfExist(dir string) bool {
	_, err := os.Stat(dir)
	if err != nil && !os.IsExist(err) {
		return false
	}
	return true
}