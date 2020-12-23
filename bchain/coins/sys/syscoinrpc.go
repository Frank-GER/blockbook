package syscoin

import (
	"encoding/json"
	
	"github.com/golang/glog"
	"github.com/syscoin/blockbook/bchain"
	"github.com/syscoin/blockbook/bchain/coins/btc"
)

// SyscoinRPC is an interface to JSON-RPC bitcoind service
type SyscoinRPC struct {
	*btc.BitcoinRPC
}

// NewSyscoinRPC returns new SyscoinRPC instance
func NewSyscoinRPC(config json.RawMessage, pushHandler func(notificationType bchain.NotificationType)) (bchain.BlockChain, error) {
	b, err := btc.NewBitcoinRPC(config, pushHandler)
	if err != nil {
		return nil, err
	}

	s := &SyscoinRPC{
		b.(*btc.BitcoinRPC),
	}
	s.RPCMarshaler = btc.JSONMarshalerV2{}
	s.ChainConfig.SupportsEstimateFee = false

	return s, nil
}

// Initialize initializes SyscoinRPC instance.
func (b *SyscoinRPC) Initialize() error {
	ci, err := b.GetChainInfo()
	if err != nil {
		return err
	}
	chainName := ci.Chain

	glog.Info("Chain name ", chainName)
	params := GetChainParams(chainName)

	// always create parser
	b.Parser = NewSyscoinParser(params, b.ChainConfig)

	// parameters for getInfo request
	if params.Net == MainnetMagic {
		b.Testnet = false
		b.Network = "livenet"
	} else {
		b.Testnet = true
		b.Network = "testnet"
	}

	glog.Info("rpc: block chain ", params.Name)

	return nil
}

// GetBlock returns block with given hash
func (b *SyscoinRPC) GetBlock(hash string, height uint32) (*bchain.Block, error) {
	var err error
	if hash == "" {
		hash, err = b.GetBlockHash(height)
		if err != nil {
			return nil, err
		}
	}
	if !b.ParseBlocks {
		return b.GetBlockFull(hash)
	}
	return b.GetBlockWithoutHeader(hash, height)
}

type ResSyscoinSend struct {
	Error  *bchain.RPCError `json:"error"`
	Result json.RawMessage      `json:"result"`
}
type GetSyscoinTxHex struct {
	Hex string `json:"hex"`
}

func (b *SyscoinRPC) GetChainTips() (string, error) {
	glog.V(1).Info("rpc: getchaintips")
	result, err := b.GetChainTips();
	if err != nil {
		return "", err
	}
	return result, nil
}