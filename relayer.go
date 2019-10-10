package btc_relayer

import (
	"encoding/json"
	"fmt"
	"github.com/Zou-XueYan/btc_relayer/log"
	"github.com/Zou-XueYan/btc_relayer/observer"
	"github.com/btcsuite/btcd/chaincfg"
	sdk "github.com/ontio/multi-chain-go-sdk"
	"io/ioutil"
	"os"
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
	}, nil
}

func (relayer *BtcRelayer) BtcListen() {
	relayer.btcOb.Listen(relayer.relaying)
}

func (relayer *BtcRelayer) AllianceListen() {
	relayer.alliaOb.Listen(relayer.Collecting)
}

func (relayer *BtcRelayer) Broadcast() {
	log.Infof("[BtcRelayer] start broadcasting")
	for item := range relayer.Collecting {
		txid, err := relayer.cli.BroadcastTx(item.Tx)
		if err != nil {
			log.Errorf("[BtcRelayer] failed to broadcast tx: %v", err)
			continue
		}
		log.Infof("[BtcRelayer] already broadcast tx: %s", txid)
	}
}

func (relayer *BtcRelayer) Relay() {
	for item := range relayer.relaying {
		log.Infof("[BtcRelayer] ralaying an item: txid: %s, height: %d", item.Txid, item.Height)

		txHash, err := relayer.allia.Native.Ccm.ImportOuterTransfer(observer.BTC_ID, item.Tx, uint32(item.Height),
			item.Proof, relayer.account.Address.ToBase58(), 0, "", relayer.account)
		if err != nil { //TODO: retry ??
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
	BtcObFirstN            int // BtcOb:
	BtcObLoopWaitTime      int64
	BtcObConfirmations     int32
	AlliaObFirstN          int // AlliaOb:
	AlliaObLoopWaitTime    int64
	WatchingKey            string
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
