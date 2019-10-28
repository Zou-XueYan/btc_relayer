package observer

import (
	"bytes"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/ontio/multi-chain/native/service/cross_chain_manager/btc"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

const (
	REDEEM_SCRIPT_HEX string = "5521023ac710e73e1410718530b2686ce47f12fa3c470a9eb6085976b70b01c64c9f732102c9dc4d8f419e325bbef0fe039ed6feaf2079a2ef7b27336ddb79be2ea6e334bf2102eac939f2f0873894d8bf0ef2f8bbdd32e4290cbf9632b59dee743529c0af9e802103378b4a3854c88cca8bfed2558e9875a144521df4a75ab37a206049ccef12be692103495a81957ce65e3359c114e6c2fe9f97568be491e3f24d6fa66cc542e360cd662102d43e29299971e802160a92cfcd4037e8ae83fb8f6af138684bebdc5686f3b9db21031e415c04cbc9b81fbee6e04d8c902e8f61109a2c9883a959ba528c52698c055a57ae"
	MULTISIG_ADDR     string = "2N5cY8y9RtbbvQRWkX5zAwTPCxSZF9xEj2C"
	BTC_ID            uint64 = 0
)

type CrossChainItem struct {
	Tx     []byte
	Proof  []byte
	Height uint32
	Txid   chainhash.Hash
}

type FromAllianceItem struct {
	Tx string
}

func checkIfCrossChainTx(tx *wire.MsgTx, netParam *chaincfg.Params) bool {
	if len(tx.TxOut) < 2 {
		return false
	}
	if tx.TxOut[0].Value <= 0 {
		return false
	}

	redeem, _ := hex.DecodeString(REDEEM_SCRIPT_HEX)
	c1 := txscript.GetScriptClass(tx.TxOut[0].PkScript)
	if c1 == txscript.MultiSigTy {
		if !bytes.Equal(redeem, tx.TxOut[0].PkScript) {
			return false
		}
	} else if c1 == txscript.ScriptHashTy {
		addr, _ := btcutil.NewAddressScriptHash(redeem, netParam)
		h, _ := txscript.PayToAddrScript(addr)
		if !bytes.Equal(h, tx.TxOut[0].PkScript) {
			return false
		}
	} else {
		return false
	}

	c2 := txscript.GetScriptClass(tx.TxOut[1].PkScript)
	if c2 != txscript.NullDataTy {
		return false
	}

	//if int(tx.TxOut[1].PkScript[1]) != btc.OP_RETURN_DATA_LEN {
	//	return false
	//}

	if tx.TxOut[1].PkScript[2] != btc.OP_RETURN_SCRIPT_FLAG {
		return false
	}

	return true
}

type Request struct {
	Jsonrpc string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	Id      int           `json:"id"`
}

type Response struct {
	Result interface{}       `json:"result"`
	Error  *btcjson.RPCError `json:"error"` //maybe wrong
	Id     int               `json:"id"`
}

// Get tx in block; Get proof;
type RestCli struct {
	Addr string
	Cli  *http.Client
}

func NewRestCli(addr, user, pwd string) *RestCli {
	return &RestCli{
		Cli: &http.Client{
			Transport: &http.Transport{
				MaxIdleConnsPerHost:   5,
				DisableKeepAlives:     false,
				IdleConnTimeout:       time.Second * 300,
				ResponseHeaderTimeout: time.Second * 300,
				TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
				Proxy: func(req *http.Request) (*url.URL, error) {
					req.SetBasicAuth(user, pwd)
					return nil, nil
				},
			},
			Timeout: time.Second * 300,
		},
		Addr: addr,
	}
}

func (cli *RestCli) sendPostReq(req []byte) (*Response, error) {
	resp, err := cli.Cli.Post(cli.Addr, "application/json;charset=UTF-8",
		bytes.NewReader(req))
	if err != nil {
		return nil, fmt.Errorf("failed to post: %v", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body error:%s", err)
	}

	response := new(Response)
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}
	return response, nil
}

func (cli *RestCli) GetProof(txids []string) (string, error) {
	req, err := json.Marshal(Request{
		Jsonrpc: "1.0",
		Method:  "gettxoutproof",
		Params:  []interface{}{txids},
		Id:      1,
	})
	if err != nil {
		return "", fmt.Errorf("failed to get proof: %v", err)
	}

	resp, err := cli.sendPostReq(req)
	if err != nil {
		return "", fmt.Errorf("failed to send post: %v", err)
	}
	if resp.Error != nil {
		return "", fmt.Errorf("response shows failure: %v", resp.Error.Message)
	}

	return resp.Result.(string), nil
}

func (cli *RestCli) GetTxsInBlock(hash string) ([]*wire.MsgTx, string, error) {
	req, err := json.Marshal(Request{
		Jsonrpc: "1.0",
		Method:  "getblock",
		Params:  []interface{}{hash, false},
		Id:      1,
	})
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal request: %v", err)
	}

	resp, err := cli.sendPostReq(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to send post: %v", err)
	}
	if resp.Error != nil {
		return nil, "", fmt.Errorf("response shows failure: %v", resp.Error.Message)
	}
	bhex := resp.Result.(string)
	bb, err := hex.DecodeString(bhex)
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode hex string: %v", err)
	}

	block := wire.MsgBlock{}
	err = block.BtcDecode(bytes.NewBuffer(bb), wire.ProtocolVersion, wire.LatestEncoding)
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode block: %v", err)
	}

	return block.Transactions, block.Header.PrevBlock.String(), nil
}

func (cli *RestCli) GetCurrentHeightAndHash() (uint32, string, error) {
	reqTips, err := json.Marshal(Request{
		Jsonrpc: "1.0",
		Method:  "getchaintips",
		Params:  nil,
		Id:      1,
	})
	if err != nil {
		return 0, "", fmt.Errorf("failed to marshal request: %v", err)
	}

	resp, err := cli.sendPostReq(reqTips)
	if err != nil {
		return 0, "", fmt.Errorf("failed to send post: %v", err)
	}
	if resp.Error != nil {
		return 0, "", fmt.Errorf("response shows failure: %v", resp.Error.Message)
	}

	m := resp.Result.([]interface{})[0].(map[string]interface{})
	return uint32(m["height"].(float64)), m["hash"].(string), nil
}

func (cli *RestCli) GetScriptPubKey(txid string, index uint32) (string, error) {
	req, err := json.Marshal(Request{
		Jsonrpc: "1.0",
		Method:  "getrawtransaction",
		Params:  []interface{}{txid, true},
		Id:      1,
	})
	if err != nil {
		return "", fmt.Errorf("[GetScriptPubKey] failed to marshal request: %v", err)
	}

	resp, err := cli.sendPostReq(req)
	if err != nil {
		return "", fmt.Errorf("[GetScriptPubKey] failed to send post: %v", err)
	}
	if resp.Error != nil {
		return "", fmt.Errorf("[GetScriptPubKey] response shows failure: %v", resp.Error.Message)
	}

	return resp.Result.(map[string]interface{})["vout"].([]interface{})[index].(map[string]interface{})["scriptPubKey"].(map[string]interface{})["hex"].(string), nil
}

func (cli *RestCli) BroadcastTx(tx string) (string, error) {
	req, err := json.Marshal(Request{
		Jsonrpc: "1.0",
		Method:  "sendrawtransaction",
		Params:  []interface{}{tx},
		Id:      1,
	})
	if err != nil {
		return "", fmt.Errorf("[BroadcastTx] failed to marshal request: %v", err)
	}

	resp, err := cli.sendPostReq(req)
	if err != nil {
		return "", fmt.Errorf("[BroadcastTx] failed to send post: %v", err)
	}
	if resp.Error != nil {
		switch resp.Error.Code {
		case btcjson.ErrRPCTxError:
			return "", NeedToRetryErr{
				Err: fmt.Errorf("[BroadcastTx] response shows failure and retry: code:%d; %v", resp.Error.Code, resp.Error.Message),
			}
		case btcjson.ErrRPCTxRejected:
			return "", NeedToRetryErr{
				Err: fmt.Errorf("[BroadcastTx] response shows failure and retry: code:%d; %v", resp.Error.Code, resp.Error.Message),
			}
		default:
			return "", fmt.Errorf("[BroadcastTx] response shows failure: %v", resp.Error.Message)
		}
	}

	return resp.Result.(string), nil
}

type NeedToRetryErr struct {
	Err error
}

func (err NeedToRetryErr) Error() string {
	return err.String()
}

func (err *NeedToRetryErr) String() string {
	return err.Err.Error()
}
