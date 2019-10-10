package observer

import (
	"encoding/hex"
	"fmt"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"testing"
	"time"
)

const (
	ADDR = "http://139.219.140.220:20332"
	USER = "test"
	PWD  = "test"
)

func TestRestCli_GetProof(t *testing.T) {
	cli := NewRestCli(ADDR, USER, PWD)
	//p := make([][]string, 0)
	//p = append(p, []string{"aa03857ff7b13d565b79d3724e516822cca223eb5dd83dd7cb35094bb7070032"})
	proof, err := cli.GetProof([]string{"aa03857ff7b13d565b79d3724e516822cca223eb5dd83dd7cb35094bb7070032"})
	if err != nil {
		t.Fatalf("Failed to get proof: %v", err)
	}

	fmt.Printf("Proof is %s\n", proof)
}

func TestRestCli_GetCurrentHeight(t *testing.T) {
	cli := NewRestCli(ADDR, USER, PWD)
	h, hash, err := cli.GetCurrentHeightAndHash()
	if err != nil {
		t.Fatalf("Failed to get height: %v", err)
	}

	fmt.Printf("height is %d and hash is %s\n", h, hash)
}

func TestRestCli_GetTxsInBlock(t *testing.T) {
	cli := NewRestCli(ADDR, USER, PWD)
	txns, hash, err := cli.GetTxsInBlock("000000000000024a13137f661835a5f1fbc52b1b21e031f82637ad4301cebafa")
	if err != nil {
		t.Fatalf("Failed to get txns: %v", err)
	}

	fmt.Printf("Successful to get tx from block %s\n", hash)
	for i, tx := range txns {
		fmt.Printf("No%d, prevOP: %x\n", i, tx.TxIn[0].PreviousOutPoint.Hash[:])
	}
}

func TestBtcObserver_SearchTxInBlock(t *testing.T) {
	line := make(chan *CrossChainItem, 2)
	o := NewBtcObserver(ADDR, USER, PWD, &chaincfg.TestNet3Params)
	txns, _, err := o.cli.GetTxsInBlock("00000000000000e8480643ba362b80f449a94743b65cefc03a460bf42167d3fc")
	if err != nil {
		t.Fatalf("Failed to get txns: %v", err)
	}

	count, err := o.SearchTxInBlock(txns, 1574156, line)
	if err != nil {
		t.Fatalf("Failed to search: %v", err)
	}

	fmt.Printf("Total count : %d\n", count)

	item, ok := <-line
	if ok {
		fmt.Printf("Item txid: %s\n", item.Txid)
		fmt.Printf("Item heigh: %d\n", item.Height)
		fmt.Printf("Item proof: %s\n", item.Proof)
		fmt.Printf("Item tx: %s\n", item.Tx)
	}
}

func TestBtcObserver_Listen(t *testing.T) {
	line := make(chan *CrossChainItem, 10)
	o := NewBtcObserver(ADDR, USER, PWD, &chaincfg.TestNet3Params)
	go o.Listen(line)
	go func() {
		for item := range line {
			fmt.Printf("Item heigh: %d\t", item.Height)
			fmt.Printf("Item proof: %s\t", item.Proof)
			fmt.Printf("Item txid: %s\n", item.Txid)
		}
	}()
	time.Sleep(time.Second * 30)
}

func TestRestCli_GetScriptPubKey(t *testing.T) {
	cli := NewRestCli(ADDR, USER, PWD)
	s, err := cli.GetScriptPubKey("8aa56bcc191e51b3214343f31b09c228626a3891f6791ff198195da76088f29b", 0)
	if err != nil {
		t.Fatalf("Failed to get txns: %v", err)
	}

	b, _ := hex.DecodeString(s)
	str, _ := txscript.DisasmString(b)

	fmt.Println(str)
}

func TestAllianceObserver_Listen(t *testing.T) {

}
