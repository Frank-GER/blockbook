// +build unittest

package db

import (
	"blockbook/bchain"
	"blockbook/common"
	"blockbook/bchain/coins/btc"
	"blockbook/bchain/coins/sys"
	"blockbook/tests/dbtestdata"
	"encoding/hex"
	"reflect"
	"testing"

	"github.com/juju/errors"
)

type testSyscoinParser struct {
	*syscoin.SyscoinParser
}

func syscoinTestParser() *syscoin.SyscoinParser {
	return syscoin.NewSyscoinParser(syscoin.GetChainParams("main"),
	&btc.Configuration{BlockAddressesToKeep: 1})
}

func verifyAfterSyscoinTypeBlock1(t *testing.T, d *RocksDB, afterDisconnect bool) {
	if err := checkColumn(d, cfHeight, []keyPair{
		{
			"0003cf7f",
			"78ae6476a514897c8a6984032e5d0e4a44424055f0c2d7b5cf664ae8c8c20487" + uintToHex(1574279564) + varuintToHex(2) + varuintToHex(1551),
			nil,
		},
	}); err != nil {
		{
			t.Fatal(err)
		}
	}
	// the vout is encoded as signed varint, i.e. value * 2 for non negative values
	if err := checkColumn(d, cfAddresses, []keyPair{
		{addressKeyHex(dbtestdata.AddrS1, 249727, d), txIndexesHex(dbtestdata.TxidS1T0, []int32{0}, d), nil},
		{addressKeyHex(dbtestdata.AddrS2, 249727, d), txIndexesHex(dbtestdata.TxidS1T0, []int32{1}, d), nil},
		{addressKeyHex(dbtestdata.AddrS3, 249727, d), txIndexesHex(dbtestdata.TxidS1T1, []int32{^1045909988, 1}, d), nil},
	}); err != nil {
		{
			t.Fatal(err)
		}
	}
	if err := checkColumn(d, cfAddressBalance, []keyPair{
		{
			dbtestdata.AddressToPubKeyHex(dbtestdata.AddrS1, d.chainParser),
			"01" + bigintToHex(dbtestdata.SatZero, d) + bigintToHex(dbtestdata.SatS1T0A1, d) +
			"00" +	dbtestdata.TxidS1T0 + varuintToHex(0) + varuintToHex(249727) + bigintToHex(dbtestdata.SatS1T0A1, d),
			nil,
		},
		{
			dbtestdata.AddressToPubKeyHex(dbtestdata.AddrS2, d.chainParser),
			"01" + bigintToHex(dbtestdata.SatZero, d) + bigintToHex(dbtestdata.SatS1T0A2, d) +
			"00" + dbtestdata.TxidS1T0 + varuintToHex(1) + varuintToHex(249727) + bigintToHex(dbtestdata.SatS1T0A2, d),
			nil,
		},
		{
			dbtestdata.AddressToPubKeyHex(dbtestdata.AddrS3, d.chainParser),
			"01" + bigintToHex(dbtestdata.SatZero, d) + bigintToHex(dbtestdata.SatS1T1A1, d) +
			"01" + varuintToHex(1045909988) + bigintToHex(dbtestdata.SatZero, d) + bigintToHex(dbtestdata.SatZero, d) + varuintToHex(1) +
				dbtestdata.TxidS1T1 + varuintToHex(1) + varuintToHex(249727) + bigintToHex(dbtestdata.SatS1T1A1, d),
			nil,
		},
	}); err != nil {
		{
			t.Fatal(err)
		}
	}

	var blockTxsKp []keyPair
	if afterDisconnect {
		blockTxsKp = []keyPair{}
	} else {
		blockTxsKp = []keyPair{
			{
				"0003cf7f",
				dbtestdata.TxidS1T0 + "01" + "0000000000000000000000000000000000000000000000000000000000000000" + "00" +
				dbtestdata.TxidS1T1 + "01" + dbtestdata.TxidS1T1INPUT0 + "02",
				nil,
			},
		}
	}

	if err := checkColumn(d, cfBlockTxs, blockTxsKp); err != nil {
		{
			t.Fatal(err)
		}
	}
}

func verifyAfterSyscoinTypeBlock2(t *testing.T, d *RocksDB) {
	if err := checkColumn(d, cfHeight, []keyPair{
		{
			"0003cf7f",
			"78ae6476a514897c8a6984032e5d0e4a44424055f0c2d7b5cf664ae8c8c20487" + uintToHex(1574279564) + varuintToHex(2) + varuintToHex(1551),
			nil,
		},
		{
			"00054cb2",
			"6609d44688868613991b0cd5ed981a76526caed6b0f7b1be242f5a93311636c6" + uintToHex(1580142055) + varuintToHex(2) + varuintToHex(1611),
			nil,
		},
	}); err != nil {
		{
			t.Fatal(err)
		}
	}
	if err := checkColumn(d, cfAddresses, []keyPair{
		{addressKeyHex(dbtestdata.AddrS1, 249727, d), txIndexesHex(dbtestdata.TxidS1T0, []int32{0}, d), nil},
		{addressKeyHex(dbtestdata.AddrS2, 249727, d), txIndexesHex(dbtestdata.TxidS1T0, []int32{1}, d), nil},
		{addressKeyHex(dbtestdata.AddrS3, 249727, d), txIndexesHex(dbtestdata.TxidS1T1, []int32{^1045909988, 1}, d), nil},
		{addressKeyHex(dbtestdata.AddrS4, 347314, d), txIndexesHex(dbtestdata.TxidS2T0, []int32{0}, d), nil},
		{addressKeyHex(dbtestdata.AddrS5, 347314, d), txIndexesHex(dbtestdata.TxidS2T0, []int32{1}, d), nil},
		{addressKeyHex(dbtestdata.AddrS3, 347314, d), txIndexesHex(dbtestdata.TxidS2T1, []int32{^1045909988, 1}, d), nil},
		{addressKeyHex(dbtestdata.AddrS6, 347314, d), txIndexesHex(dbtestdata.TxidS2T1, []int32{1045909988}, d), nil},
	}); err != nil {
		{
			t.Fatal(err)
		}
	}
	if err := checkColumn(d, cfAddressBalance, []keyPair{
		{
			dbtestdata.AddressToPubKeyHex(dbtestdata.AddrS1, d.chainParser),
			"01" + bigintToHex(dbtestdata.SatZero, d) + bigintToHex(dbtestdata.SatS1T0A1, d) +
			"00" + dbtestdata.TxidS1T0 + varuintToHex(0) + varuintToHex(249727) + bigintToHex(dbtestdata.SatS1T0A1, d),
			nil,
		},
		{
			dbtestdata.AddressToPubKeyHex(dbtestdata.AddrS2, d.chainParser),
			"01" + bigintToHex(dbtestdata.SatZero, d) + bigintToHex(dbtestdata.SatS1T0A2, d) +
			"00" + dbtestdata.TxidS1T0 + varuintToHex(1) + varuintToHex(249727) + bigintToHex(dbtestdata.SatS1T0A2, d),
			nil,
		},
		{
			dbtestdata.AddressToPubKeyHex(dbtestdata.AddrS3, d.chainParser),
			"02" + bigintToHex(dbtestdata.SatZero, d) + bigintToHex(dbtestdata.SatS1T1A1.Add(dbtestdata.SatS1T1A1, dbtestdata.SatS2T1A1), d) +
			"01" + varuintToHex(1045909988) + bigintToHex(dbtestdata.SatZero, d) + bigintToHex(dbtestdata.SatAssetSent, d) + varuintToHex(2) +
				dbtestdata.TxidS1T1 + varuintToHex(1) + varuintToHex(249727) + bigintToHex(dbtestdata.SatS1T1A1, d) +
				dbtestdata.TxidS2T1 + varuintToHex(1) + varuintToHex(347314) + bigintToHex(dbtestdata.SatS2T1A1, d),
			nil,
		},
		{
			dbtestdata.AddressToPubKeyHex(dbtestdata.AddrS4, d.chainParser),
			"01" + bigintToHex(dbtestdata.SatZero, d) + bigintToHex(dbtestdata.SatS2T0A1, d) +
			"00" + dbtestdata.TxidS1T0 + varuintToHex(0) + varuintToHex(347314) + bigintToHex(dbtestdata.SatS2T0A1, d),
			nil,
		},
		{
			dbtestdata.AddressToPubKeyHex(dbtestdata.AddrS5, d.chainParser),
			"01" + bigintToHex(dbtestdata.SatZero, d) + bigintToHex(dbtestdata.SatS2T0A2, d) +
			"00" + dbtestdata.TxidS1T0 + varuintToHex(1) + varuintToHex(347314) + bigintToHex(dbtestdata.SatS2T0A2, d),
			nil,
		},
		// burn should have a address balance as asset output from S2T1
		{
			dbtestdata.AddressToPubKeyHex(dbtestdata.AddrS6, d.chainParser),
			"01" + bigintToHex(dbtestdata.SatZero, d) + bigintToHex(dbtestdata.SatZero, d) +
			"01" + varuintToHex(1045909988) + bigintToHex(dbtestdata.SatAssetSent, d) + bigintToHex(dbtestdata.SatZero, d) + varuintToHex(1),
			nil,
		},
	}); err != nil {
		{
			t.Fatal(err)
		}
	}
	if err := checkColumn(d, cfBlockTxs, []keyPair{
		{
			"00054cb2",
			dbtestdata.TxidS2T0 + "01" + "0000000000000000000000000000000000000000000000000000000000000000" + "00" +
			dbtestdata.TxidS2T1 + "01" + dbtestdata.TxidS2T1INPUT0 + "02",
			nil,
		},
	}); err != nil {
		{
			t.Fatal(err)
		}
	}
}

// TestRocksDB_Index_SyscoinType is an integration test probing the whole indexing functionality for Syscoin which is a BitcoinType chain
// It does the following:
// 1) Connect two blocks (inputs from 2nd block are spending some outputs from the 1st block)
// 2) GetTransactions for various addresses / low-high ranges
// 3) GetBestBlock, GetBlockHash
// 4) Test tx caching functionality
// 5) Disconnect the block 2 using BlockTxs column
// 6) Reconnect block 2 and check
// After each step, the content of DB is examined and any difference against expected state is regarded as failure
func TestRocksDB_Index_SyscoinType(t *testing.T) {
	d := setupRocksDB(t, &testSyscoinParser{
		SyscoinParser: syscoinTestParser(),
	})
	defer closeAndDestroyRocksDB(t, d)

	if len(d.is.BlockTimes) != 0 {
		t.Fatal("Expecting is.BlockTimes 0, got ", len(d.is.BlockTimes))
	}

	// connect 1st block - will log warnings about missing UTXO transactions in txAddresses column
	block1 := dbtestdata.GetTestSyscoinTypeBlock1(d.chainParser)
	if err := d.ConnectBlock(block1); err != nil {
		t.Fatal(err)
	}
	verifyAfterSyscoinTypeBlock1(t, d, false)

	if len(d.is.BlockTimes) != 1 {
		t.Fatal("Expecting is.BlockTimes 1, got ", len(d.is.BlockTimes))
	}

	// connect 2nd block - use some outputs from the 1st block as the inputs and 1 input uses tx from the same block
	block2 := dbtestdata.GetTestSyscoinTypeBlock2(d.chainParser)
	if err := d.ConnectBlock(block2); err != nil {
		t.Fatal(err)
	}
	verifyAfterSyscoinTypeBlock2(t, d)

	if len(d.is.BlockTimes) != 2 {
		t.Fatal("Expecting is.BlockTimes 1, got ", len(d.is.BlockTimes))
	}

	// get transactions for various addresses / low-high ranges
	verifyGetTransactions(t, d, dbtestdata.AddrS3, 0, 1000000, []txidIndex{
		{dbtestdata.TxidS1T1, 1},
		{dbtestdata.TxidS1T1, ^1045909988}, // asset is used as input to update asset
		{dbtestdata.TxidS2T1, 1},
		{dbtestdata.TxidS2T1, ^1045909988}, // asset is used as input to send to addr6 (burn)
	}, nil)
	verifyGetTransactions(t, d, dbtestdata.AddrS3, 249727, 249727, []txidIndex{
		{dbtestdata.TxidS1T1, 1},
		{dbtestdata.TxidS1T1, ^1045909988},
	}, nil)
	verifyGetTransactions(t, d, dbtestdata.AddrS3, 347314, 1000000, []txidIndex{
		{dbtestdata.TxidS2T1, 1},
		{dbtestdata.TxidS2T1, ^1045909988},
	}, nil)
	verifyGetTransactions(t, d, dbtestdata.AddrS3, 500000, 1000000, []txidIndex{}, nil)
	verifyGetTransactions(t, d, dbtestdata.AddrS4, 0, 1000000, []txidIndex{
		{dbtestdata.TxidS2T1, 0},
	}, nil)
	verifyGetTransactions(t, d, dbtestdata.AddrS6, 0, 1000000, []txidIndex{
		{dbtestdata.TxidS2T1, 1045909988}, // sent to addr6 burn as asset
	}, nil)
	verifyGetTransactions(t, d, "SgBVZhGLjqRz8ufXFwLhZvXpUMKqoduBad", 500000, 1000000, []txidIndex{}, errors.New("checksum mismatch"))

	// GetBestBlock
	height, hash, err := d.GetBestBlock()
	if err != nil {
		t.Fatal(err)
	}
	if height != 347314 {
		t.Fatalf("GetBestBlock: got height %v, expected %v", height, 347314)
	}
	if hash != "6609d44688868613991b0cd5ed981a76526caed6b0f7b1be242f5a93311636c6" {
		t.Fatalf("GetBestBlock: got hash %v, expected %v", hash, "6609d44688868613991b0cd5ed981a76526caed6b0f7b1be242f5a93311636c6")
	}

	// GetBlockHash
	hash, err = d.GetBlockHash(249727)
	if err != nil {
		t.Fatal(err)
	}
	if hash != "78ae6476a514897c8a6984032e5d0e4a44424055f0c2d7b5cf664ae8c8c20487" {
		t.Fatalf("GetBlockHash: got hash %v, expected %v", hash, "78ae6476a514897c8a6984032e5d0e4a44424055f0c2d7b5cf664ae8c8c20487")
	}

	// Not connected block
	hash, err = d.GetBlockHash(347315)
	if err != nil {
		t.Fatal(err)
	}
	if hash != "" {
		t.Fatalf("GetBlockHash: got hash '%v', expected ''", hash)
	}

	// GetBlockHash
	info, err := d.GetBlockInfo(347314)
	if err != nil {
		t.Fatal(err)
	}
	iw := &bchain.DbBlockInfo{
		Hash:   "6609d44688868613991b0cd5ed981a76526caed6b0f7b1be242f5a93311636c6",
		Txs:    2,
		Size:   1611,
		Time:   1580142055,
		Height: 347314,
	}
	if !reflect.DeepEqual(info, iw) {
		t.Errorf("GetBlockInfo() = %+v, want %+v", info, iw)
	}

	// Test tx caching functionality, leave one tx in db to test cleanup in DisconnectBlock
	testTxCache(t, d, block1, &block1.Txs[0])
	testTxCache(t, d, block2, &block2.Txs[0])
	if err = d.PutTx(&block2.Txs[1], block2.Height, block2.Txs[1].Blocktime); err != nil {
		t.Fatal(err)
	}
	// check that there is only the last tx in the cache
	packedTx, err := d.chainParser.PackTx(&block2.Txs[1], block2.Height, block2.Txs[1].Blocktime)
	if err := checkColumn(d, cfTransactions, []keyPair{
		{block2.Txs[1].Txid, hex.EncodeToString(packedTx), nil},
	}); err != nil {
		{
			t.Fatal(err)
		}
	}

	// try to disconnect both blocks, however only the last one is kept, it is not possible
	err = d.DisconnectBlockRangeBitcoinType(249727, 347314)
	if err == nil || err.Error() != "Cannot disconnect blocks with height 249727 and lower. It is necessary to rebuild index." {
		t.Fatal(err)
	}
	verifyAfterSyscoinTypeBlock2(t, d)

	// disconnect the 2nd block, verify that the db contains only data from the 1st block with restored unspentTxs
	// and that the cached tx is removed
	err = d.DisconnectBlockRangeBitcoinType(347314, 347314)
	if err != nil {
		t.Fatal(err)
	}
	verifyAfterSyscoinTypeBlock1(t, d, true)
	if err := checkColumn(d, cfTransactions, []keyPair{}); err != nil {
		{
			t.Fatal(err)
		}
	}

	if len(d.is.BlockTimes) != 1 {
		t.Fatal("Expecting is.BlockTimes 1, got ", len(d.is.BlockTimes))
	}

	// connect block again and verify the state of db
	if err := d.ConnectBlock(block2); err != nil {
		t.Fatal(err)
	}
	verifyAfterSyscoinTypeBlock2(t, d)

	if len(d.is.BlockTimes) != 2 {
		t.Fatal("Expecting is.BlockTimes 1, got ", len(d.is.BlockTimes))
	}

	// test public methods for address balance and tx addresses
	ab, err := d.GetAddressBalance(dbtestdata.AddrS3, bchain.AddressBalanceDetailUTXO)
	if err != nil {
		t.Fatal(err)
	}
	abw := &bchain.AddrBalance{
		Txs:        2,
		SentSat:    *dbtestdata.SatZero,
		BalanceSat: *dbtestdata.SatS1T1A1.Add(dbtestdata.SatS1T1A1, dbtestdata.SatS2T1A1),
		Utxos: []bchain.Utxo{
			{
				BtxID:    hexToBytes(dbtestdata.TxidS1T1),
				Vout:     1,
				Height:   249727,
				ValueSat: *dbtestdata.SatS1T1A1,
			},
			{
				BtxID:    hexToBytes(dbtestdata.TxidS2T1),
				Vout:     1,
				Height:   347314,
				ValueSat: *dbtestdata.SatS2T1A1,
			},
		},
		AssetBalances: map[uint32]*bchain.AssetBalance {
			1045909988: &bchain.AssetBalance{
				SentAssetSat: 	dbtestdata.SatAssetSent,
				BalanceAssetSat: dbtestdata.SatZero,
				Transfers:	2,
			},
		},
	}
	if !reflect.DeepEqual(ab, abw) {
		t.Errorf("GetAddressBalance() = %+v, want %+v", ab, abw)
	}
	rs := ab.ReceivedSat()
	rsw := dbtestdata.SatS1T1A1.Add(dbtestdata.SatS1T1A1, dbtestdata.SatS2T1A1)
	if rs.Cmp(rsw) != 0 {
		t.Errorf("GetAddressBalance().ReceivedSat() = %v, want %v", rs, rsw)
	}

	rsa := bchain.ReceivedSatFromBalances(dbtestdata.SatZero, dbtestdata.SatAssetSent)
	rswa := dbtestdata.SatAssetSent
	if rsa.Cmp(rswa) != 0 {
		t.Errorf("GetAddressBalance().ReceivedSatFromBalances() = %v, want %v", rsa, rswa)
	}

	ta, err := d.GetTxAddresses(dbtestdata.TxidS2T1)
	if err != nil {
		t.Fatal(err)
	}
	tokenRecipient := &bchain.TokenTransferRecipient{
		To: dbtestdata.AddrS6,
		Value: (*bchain.Amount)(dbtestdata.SatAssetSent),
	}
	taw := &bchain.TxAddresses{
		Height: 347314,
		Inputs: []bchain.TxInput{
			{
				AddrDesc: addressToAddrDesc(dbtestdata.TxidS2T1INPUT0, d.chainParser),
				ValueSat: *dbtestdata.SatS2T1INPUT0,
			},
		},
		Outputs: []bchain.TxOutput{
			{
				AddrDesc: hexToBytes(dbtestdata.TxidS2T1OutputReturn),
				Spent:    false,
				ValueSat: *dbtestdata.SatZero,
			},
			{
				AddrDesc: addressToAddrDesc(dbtestdata.AddrS3, d.chainParser),
				Spent:    false,
				ValueSat: *dbtestdata.SatS2T1A1,
			},
		},
		TokenTransferSummary: &bchain.TokenTransferSummary {
			Type:   bchain.SPTAssetSendType,
			From:	dbtestdata.AddrS3,
			Token:  "1045909988", 
			Symbol: "SYSX",
			Decimals: 8,
			Value:	 (*bchain.Amount)(dbtestdata.SatAssetSent),
			Fee:     (*bchain.Amount)(dbtestdata.SatZero),
			Recipients: []*bchain.TokenTransferRecipient{tokenRecipient},
		},
	}
	if !reflect.DeepEqual(ta, taw) {
		t.Errorf("GetTxAddresses() = %+v, want %+v", ta, taw)
	}
	ia, _, err := ta.Inputs[0].Addresses(d.chainParser)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ia, []string{dbtestdata.AddrS3}) {
		t.Errorf("GetTxAddresses().Inputs[0].Addresses() = %v, want %v", ia, []string{dbtestdata.AddrS3})
	}

}

func Test_BulkConnect_SyscoinType(t *testing.T) {
	d := setupRocksDB(t, &testSyscoinParser{
		SyscoinParser: syscoinTestParser(),
	})
	defer closeAndDestroyRocksDB(t, d)

	bc, err := d.InitBulkConnect()
	if err != nil {
		t.Fatal(err)
	}

	if d.is.DbState != common.DbStateInconsistent {
		t.Fatal("DB not in DbStateInconsistent")
	}

	if len(d.is.BlockTimes) != 0 {
		t.Fatal("Expecting is.BlockTimes 0, got ", len(d.is.BlockTimes))
	}

	if err := bc.ConnectBlock(dbtestdata.GetTestSyscoinTypeBlock1(d.chainParser), false); err != nil {
		t.Fatal(err)
	}
	if err := checkColumn(d, cfBlockTxs, []keyPair{}); err != nil {
		{
			t.Fatal(err)
		}
	}

	if err := bc.ConnectBlock(dbtestdata.GetTestSyscoinTypeBlock2(d.chainParser), true); err != nil {
		t.Fatal(err)
	}

	if err := bc.Close(); err != nil {
		t.Fatal(err)
	}

	if d.is.DbState != common.DbStateOpen {
		t.Fatal("DB not in DbStateOpen")
	}

	verifyAfterSyscoinTypeBlock2(t, d)

	if len(d.is.BlockTimes) != 347315 {
		t.Fatal("Expecting is.BlockTimes 347315, got ", len(d.is.BlockTimes))
	}
}
