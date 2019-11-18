package btc_relayer

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/btcsuite/btcd/wire"
	"github.com/ontio/btcrelayer/db"
	"github.com/ontio/btcrelayer/log"
	"github.com/ontio/btcrelayer/observer"
	sdk "github.com/ontio/multi-chain-go-sdk"
	"github.com/ontio/multi-chain-go-sdk/client"
	"github.com/ontio/multi-chain/common/password"
	"io/ioutil"
	"os"
	"time"
)

type BtcRelayer struct {
	btcOb      *observer.BtcObserver
	alliaOb    *observer.AllianceObserver
	account    *sdk.Account
	relaying   chan *observer.CrossChainItem
	collecting chan *observer.FromAllianceItem
	allia      *sdk.MultiChainSdk
	config     *RelayerConfig
	cli        *observer.RestCli
	retryDB    *db.RetryDB
}

func NewBtcRelayer(conf *RelayerConfig) (*BtcRelayer, error) {
	allia := sdk.NewMultiChainSdk()
	allia.NewRpcClient().SetAddress(conf.AlliaObConf.AllianceJsonRpcAddress)
	acct, err := GetAccountByPassword(allia, conf.AlliaObConf.WalletFile, conf.AlliaObConf.WalletPwd)
	if err != nil {
		return nil, fmt.Errorf("GetAccountByPassword failed: %v", err)
	}

	if !checkIfExist(conf.RetryDBPath) {
		os.Mkdir(conf.RetryDBPath, os.ModePerm)
	}

	rdb, err := db.NewRetryDB(conf.RetryDBPath, conf.RetryTimes, conf.RetryDuration, conf.MaxReadSize)
	if err != nil {
		return nil, fmt.Errorf("failed to new retry db: %v", err)
	}

	cli := observer.NewRestCli(conf.BtcObConf.BtcJsonRpcAddress, conf.BtcObConf.User, conf.BtcObConf.Pwd)
	return &BtcRelayer{
		btcOb:      observer.NewBtcObserver(conf.BtcObConf, cli, rdb),
		alliaOb:    observer.NewAllianceObserver(allia, conf.AlliaObConf, rdb),
		account:    acct,
		relaying:   make(chan *observer.CrossChainItem, 10),
		collecting: make(chan *observer.FromAllianceItem, 10),
		allia:      allia,
		config:     conf,
		cli:        cli,
		retryDB:    rdb,
	}, nil
}

func (relayer *BtcRelayer) BtcListen() {
	relayer.btcOb.Listen(relayer.relaying)
}

func (relayer *BtcRelayer) AllianceListen() {
	relayer.alliaOb.Listen(relayer.collecting)
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
			for i := 0; i < len(txArr); i++ {
				txb, _ := hex.DecodeString(txArr[i])
				mtx := wire.NewMsgTx(wire.TxVersion)
				mtx.BtcDecode(bytes.NewBuffer(txb), wire.ProtocolVersion, wire.LatestEncoding)
				txid, err := relayer.cli.BroadcastTx(txArr[i])
				if err != nil {
					switch err.(type) {
					case observer.NeedToRetryErr:
						log.Errorf("[BtcRelayer] rebroadcast %s failed: %v", mtx.TxHash().String(), err)
					case observer.NetErr:
						i--
						log.Errorf("[BtcRelayer] net err happened, rebroadcast %s failed: %v", mtx.TxHash().String(), err)
						<-time.Tick(time.Second * observer.SleepTime)
					default:
						log.Infof("[BtcRelayer] no need to rebroadcast and delete this tx %s...%s: %v", txArr[i][:16],
							txArr[i][len(txArr[i])-16:], err)
						err = relayer.retryDB.Del(txArr[i])
						if err != nil {
							log.Errorf("[BtcRelayer] failed to delete tx %s(%s): %v", txid, txArr[i], err)
						}
					}
				} else {
					log.Infof("[BtcRelayer] rebroadcast and delete tx: %s", txid)
					err = relayer.retryDB.Del(txArr[i])
					if err != nil {
						log.Errorf("[BtcRelayer] failed to delete tx %s(%s): %v", txid, txArr[i], err)
					}
				}
			}
		}
	}
}

func (relayer *BtcRelayer) Broadcast() {
	log.Infof("[BtcRelayer] start broadcasting")
	for item := range relayer.collecting {
		txid, err := relayer.cli.BroadcastTx(item.Tx)
		if err != nil {
			switch err.(type) {
			case observer.NeedToRetryErr:
				log.Infof("[BtcRelayer] need to rebroadcast this tx %s...%s: %v", item.Tx[:16], item.Tx[len(item.Tx)-16:], err)
				err = relayer.retryDB.Put(item.Tx)
				if err != nil {
					log.Errorf("[BtcRelayer] failed to put tx in db: %v", err)
				}
			case observer.NetErr:
				relayer.collecting <- item
				log.Errorf("[BtcRelayer] net err happened, put it(%s...%s) back to channel: %v", item.Tx[:16],
					item.Tx[len(item.Tx)-16:], err)
				<-time.Tick(time.Second * observer.SleepTime)
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
			switch err.(type) {
			case client.PostErr:
				log.Errorf("[BtcRelayer] failed to relay and post err: %v", err)
				go func() {
					relayer.relaying <- item
				}()
				<-time.Tick(time.Second * observer.SleepTime)
			default:
				log.Errorf("[BtcRelayer] invokeNativeContract error: %v", err)
			}
			continue
		}
		log.Infof("[BtcRelayer] %s sent to alliance : txid: %s, height: %d", txHash.ToHexString(),
			item.Txid, item.Height)
	}
}

type RelayerConfig struct {
	BtcObConf     *observer.BtcObConfig      `json:"btc_ob_conf"`
	AlliaObConf   *observer.AllianceObConfig `json:"allia_ob_conf"`
	RetryDuration int                        `json:"retry_duration"`
	RetryTimes    int                        `json:"retry_times"`
	RetryDBPath   string                     `json:"retry_db_path"`
	LogLevel      int                        `json:"log_level"`
	SleepTime     int                        `json:"sleep_time"`
	MaxReadSize   uint64                     `json:"max_read_size"`
}

func NewRelayerConfig(file string) (*RelayerConfig, error) {
	conf := &RelayerConfig{}
	err := conf.Init(file)
	if err != nil {
		return conf, fmt.Errorf("failed to new config: %v", err)
	}
	return conf, nil
}

func (this *RelayerConfig) Init(fileName string) error {
	err := this.loadConfig(fileName)
	if err != nil {
		return fmt.Errorf("loadConfig error:%s", err)
	}
	return nil
}

func (this *RelayerConfig) loadConfig(fileName string) error {
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

func (this *RelayerConfig) readFile(fileName string) ([]byte, error) {
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
	pwdb := []byte{}
	if pwd == "" {
		pwdb, err = password.GetPassword()
		if err != nil {
			return nil, fmt.Errorf("getPassword error: %v", err)
		}
	} else {
		pwdb = []byte(pwd)
	}
	user, err := wallet.GetDefaultAccount(pwdb)
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
