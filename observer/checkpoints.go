package observer

type Checkpoint struct {
	Height uint32
}

var btcCheckPoints map[string]*Checkpoint
var alliaCheckPoints map[string]*Checkpoint

func init() {
	btcCheckPoints = make(map[string]*Checkpoint)
	alliaCheckPoints = make(map[string]*Checkpoint)

	btcCheckPoints["regtest"] = &Checkpoint{
		Height: 5,
	}
	btcCheckPoints["mainnet"] = &Checkpoint{
		Height: 602805, // need to set
	}
	btcCheckPoints["testnet3"] = &Checkpoint{
		Height: 1607304,
	}

	alliaCheckPoints["testnet"] = &Checkpoint{
		Height: 1,
	}
	alliaCheckPoints["regtest"] = &Checkpoint{
		Height: 1,
	}
}
