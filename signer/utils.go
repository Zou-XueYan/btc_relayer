package signer

import (
	"errors"
	"fmt"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcutil/base58"
)

var addr1 = "mj3LUsSvk9ZQH1pSHvC8LBtsYXsZvbky8H"
var priv1 = "cTqbqa1YqCf4BaQTwYDGsPAB4VmWKUU67G5S1EtrHSWNRwY6QSag"
var addr2 = "mtNiC48WWbGRk2zLqiTMwKLhrCk6rBqBen"
var priv2 = "cT2HP4QvL8c6otn4LrzUWzgMBfTo1gzV2aobN1cTiuHPXH9Jk2ua"
var addr3 = "mi1bYK8SR3Qsf2cdrxgak3spzFx4EVH1pf"
var priv3 = "cSQmGg6spbhd23jHQ9HAtz3XU7GYJjYaBmFLWHbyKa9mWzTxEY5A"
var addr4 = "mz3bTZaQ2tNzsn4szNE8R6gp5zyHuqN29V"
var priv4 = "cPYAx61EjwshK5SQ6fqH7QGjc8L48xiJV7VRGpYzPSbkkZqrzQ5b"
var addr5 = "mfzbFf6njbEuyvZGDiAdfKamxWfAMv47NG"
var priv5 = "cVV9UmtnnhebmSQgHhbDZWCb7zBHbiAGDB9a5M2ffe1WpqvwD5zg"
var addr6 = "n4ESieuFJq5HCvE5GU8B35YTfShZmFrCKM"
var priv6 = "cNK7BwHmi8rZiqD2QfwJB1R6bF6qc7iVTMBNjTr2ACbsoq1vWau8"
var addr7 = "msK9xpuXn5xqr4UK7KyWi9VCaFhiwCqqq6"
var priv7 = "cUZdDF9sL11ya5civzMRYVYojoojjHbmWWm1yC5uRzfBRePVbQTZ"

func getPrivkMap() map[string]*btcec.PrivateKey {
	privv1, _ := btcec.PrivKeyFromBytes(btcec.S256(), base58.Decode(priv1))
	privv2, _ := btcec.PrivKeyFromBytes(btcec.S256(), base58.Decode(priv2))
	privv3, _ := btcec.PrivKeyFromBytes(btcec.S256(), base58.Decode(priv3))
	privv4, _ := btcec.PrivKeyFromBytes(btcec.S256(), base58.Decode(priv4))
	privv5, _ := btcec.PrivKeyFromBytes(btcec.S256(), base58.Decode(priv5))
	privv6, _ := btcec.PrivKeyFromBytes(btcec.S256(), base58.Decode(priv6))
	privv7, _ := btcec.PrivKeyFromBytes(btcec.S256(), base58.Decode(priv7))

	//privv, _ := btcec.PrivKeyFromBytes(btcec.S256(), base58.Decode("cRRMYvoHPNQu1tCz4ajPxytBVc2SN6GWLAVuyjzm4MVwyqZVrAcX"))
	privks := map[string]*btcec.PrivateKey{
		addr1: privv1,
		addr2: privv2,
		addr3: privv3,
		addr4: privv4,
		addr5: privv5,
		addr6: privv6,
		addr7: privv7,
	}

	return privks
}

func lookUpMultiKeys(addr btcutil.Address) (*btcec.PrivateKey, bool, error) {
	m := getPrivkMap()
	if len(m) == 0 || m == nil {
		return nil, false, errors.New("Private key not ready")
	}
	val, ok := m[addr.EncodeAddress()]
	if !ok {
		return nil, false, fmt.Errorf("Private key not found")
	}
	return val, true, nil
}

func lookUpOneKey(addr btcutil.Address) (*btcec.PrivateKey, bool, error) {
	m := getPrivkMap()
	if len(m) == 0 || m == nil {
		return nil, false, errors.New("Private key not ready")
	}
	val, ok := m[addr.EncodeAddress()]
	if !ok {
		return nil, false, fmt.Errorf("Private key not found")
	}
	return val, true, nil
}

func getPubKeys() [][]byte {
	_, pubk1 := btcec.PrivKeyFromBytes(btcec.S256(), base58.Decode(priv1))
	_, pubk2 := btcec.PrivKeyFromBytes(btcec.S256(), base58.Decode(priv2))
	_, pubk3 := btcec.PrivKeyFromBytes(btcec.S256(), base58.Decode(priv3))
	_, pubk4 := btcec.PrivKeyFromBytes(btcec.S256(), base58.Decode(priv4))
	_, pubk5 := btcec.PrivKeyFromBytes(btcec.S256(), base58.Decode(priv5))
	_, pubk6 := btcec.PrivKeyFromBytes(btcec.S256(), base58.Decode(priv6))
	_, pubk7 := btcec.PrivKeyFromBytes(btcec.S256(), base58.Decode(priv7))

	pubks := make([][]byte, 0)
	pubks = append(pubks, pubk1.SerializeCompressed(), pubk2.SerializeCompressed(), pubk3.SerializeCompressed(),
		pubk4.SerializeCompressed(), pubk5.SerializeCompressed(), pubk6.SerializeCompressed(), pubk7.SerializeCompressed())
	return pubks
}

type SignedTxItem struct {
	Txid string
	Data string
}
