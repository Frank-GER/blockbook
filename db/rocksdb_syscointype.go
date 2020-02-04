package db

import (
	"blockbook/bchain"
	"bytes"
	"strconv"
	"math/big"
	"github.com/golang/glog"
	"github.com/syscoin/btcd/wire"
	"github.com/juju/errors"
)

func (d *RocksDB) ConnectAssetOutput(sptData []byte, balances map[string]*bchain.AddrBalance, version int32, addresses bchain.AddressesMap, btxID []byte, outputIndex int32, txAddresses* bchain.TxAddresses) error {
	r := bytes.NewReader(sptData)
	var asset wire.AssetType
	err := asset.Deserialize(r)
	if err != nil {
		return err
	}
	assetGuid := asset.Asset
	strAssetGuid := strconv.FormatUint(uint64(assetGuid), 10)
	senderAddress := asset.WitnessAddress.ToString("sys")
	assetSenderAddrDesc, err := d.chainParser.GetAddrDescFromAddress(senderAddress)
	if err != nil || len(assetSenderAddrDesc) == 0 || len(assetSenderAddrDesc) > maxAddrDescLen {
		if err != nil {
			// do not log ErrAddressMissing, transactions can be without to address (for example eth contracts)
			if err != bchain.ErrAddressMissing {
				glog.Warningf("ConnectAssetOutput sender with asset %v (%v) could not be decoded error %v", assetGuid, string(assetSenderAddrDesc), err)
			}
		} else {
			glog.Warningf("ConnectAssetOutput sender with asset %v (%v) has invalid length: %d", assetGuid, string(assetSenderAddrDesc), len(assetSenderAddrDesc))
		}
		return errors.New("ConnectAssetOutput Skipping asset tx")
	}
	senderStr := string(assetSenderAddrDesc)
	balance, e := balances[senderStr]
	if !e {
		balance, err = d.GetAddrDescBalance(assetSenderAddrDesc, bchain.AddressBalanceDetailUTXOIndexed)
		if err != nil {
			return err
		}
		if balance == nil {
			balance = &bchain.AddrBalance{}
		}
		balances[senderStr] = balance
		d.cbs.balancesMiss++
	} else {
		d.cbs.balancesHit++
	}

	txAddresses.TokenTransfers = make([]*bchain.TokenTransfer, 1)
	if len(asset.WitnessAddressTransfer.WitnessProgram) > 0 {
		receiverAddress := asset.WitnessAddressTransfer.ToString("sys")
		assetTransferWitnessAddrDesc, err := d.chainParser.GetAddrDescFromAddress(receiverAddress)
		if err != nil || len(assetSenderAddrDesc) == 0 || len(assetSenderAddrDesc) > maxAddrDescLen {
			if err != nil {
				// do not log ErrAddressMissing, transactions can be without to address (for example eth contracts)
				if err != bchain.ErrAddressMissing {
					glog.Warningf("ConnectAssetOutput transferee with asset %v (%v) could not be decoded error %v", assetGuid, string(assetTransferWitnessAddrDesc), err)
				}
			} else {
				glog.Warningf("ConnectAssetOutput transferee with asset %v (%v) has invalid length: %d", assetGuid, string(assetTransferWitnessAddrDesc), len(assetTransferWitnessAddrDesc))
			}
			return errors.New("ConnectAssetOutput Skipping asset transfer tx")
		}
		transferStr := string(assetTransferWitnessAddrDesc)
		balanceTransfer, e1 := balances[transferStr]
		if !e1 {
			balanceTransfer, err = d.GetAddrDescBalance(assetTransferWitnessAddrDesc, bchain.AddressBalanceDetailUTXOIndexed)
			if err != nil {
				return err
			}
			if balanceTransfer == nil {
				balanceTransfer = &bchain.AddrBalance{}
			}
			balances[transferStr] = balanceTransfer
			d.cbs.balancesMiss++
		} else {
			d.cbs.balancesHit++
		}
		counted := addToAddressesMap(addresses, transferStr, btxID, outputIndex)
		if !counted {
			balanceTransfer.Txs++
		}
		// transfer balance from old address to transfered address
		if balanceTransfer.AssetBalances == nil{
			balanceTransfer.AssetBalances = map[uint32]*bchain.AssetBalance{}
		}
		balanceAssetTransfer, ok := balanceTransfer.AssetBalances[assetGuid]
		if !ok {
			balanceAssetTransfer = &bchain.AssetBalance{Transfers: 0, BalanceAssetSat: big.NewInt(0), SentAssetSat: big.NewInt(0), UnallocatedBalanceSat: big.NewInt(0)}
			balanceTransfer.AssetBalances[assetGuid] = balanceAssetTransfer
		}
		balanceAssetTransfer.Transfers++
		balanceAsset, ok := balance.AssetBalances[assetGuid]
		if !ok {
			balanceAsset = &bchain.AssetBalance{Transfers: 0, BalanceAssetSat: big.NewInt(0), SentAssetSat: big.NewInt(0), UnallocatedBalanceSat: big.NewInt(0)}
			balance.AssetBalances[assetGuid] = balanceAsset
		}
		balanceAsset.Transfers++
		// transfer balance to new receiver
		balanceAssetTransfer.UnallocatedBalanceSat.Set(balanceAsset.UnallocatedBalanceSat)
		// clear balance on sender
		copyAmount := *balanceAsset.UnallocatedBalanceSat
		balanceAsset.UnallocatedBalanceSat.Set(big.NewInt(0))
		txAddresses.TokenTransfers[0] = &bchain.TokenTransfer {
			Type:     bchain.SPTAssetUpdateType,
			Token:    strAssetGuid,
			From:     senderAddress,
			To:       receiverAddress,
			Decimals: 8,
			Value:    (*bchain.Amount)(&copyAmount),
			Symbol:   "SPT",
		}
	} else {
		if balance.AssetBalances == nil{
			balance.AssetBalances = map[uint32]*bchain.AssetBalance{}
		}
		balanceAsset, ok := balance.AssetBalances[assetGuid]
		if !ok {
			balanceAsset = &bchain.AssetBalance{Transfers: 0, BalanceAssetSat: big.NewInt(0), SentAssetSat: big.NewInt(0), UnallocatedBalanceSat: big.NewInt(0)}
			balance.AssetBalances[assetGuid] = balanceAsset
		}
		balanceAsset.Transfers++
		valueTo := big.NewInt(asset.Balance)
		balanceAsset.UnallocatedBalanceSat.Add(balanceAsset.UnallocatedBalanceSat, valueTo)
		txAddresses.TokenTransfers[0] = &bchain.TokenTransfer {
			Type:     bchain.SPTAssetTransferType,
			Token:    strAssetGuid,
			From:     senderAddress,
			To:       senderAddress,
			Decimals: 8,
			Value:    (*bchain.Amount)(valueTo),
			Symbol:   "SPT",
		}
	}
	return nil
}

func (d *RocksDB) ConnectAssetAllocationOutput(sptData []byte, balances map[string]*bchain.AddrBalance, version int32, addresses bchain.AddressesMap, btxID []byte, outputIndex int32, txAddresses* bchain.TxAddresses) error {
	r := bytes.NewReader(sptData)
	var assetAllocation wire.AssetAllocationType
	err := assetAllocation.Deserialize(r)
	if err != nil {
		return err
	}
	totalAssetSentValue := big.NewInt(0)
	assetGuid := assetAllocation.AssetAllocationTuple.Asset
	strAssetGuid := strconv.FormatUint(uint64(assetGuid), 10)
	senderAddress := assetAllocation.AssetAllocationTuple.WitnessAddress.ToString("sys")
	assetSenderAddrDesc, err := d.chainParser.GetAddrDescFromAddress(senderAddress)
	if err != nil || len(assetSenderAddrDesc) == 0 || len(assetSenderAddrDesc) > maxAddrDescLen {
		if err != nil {
			// do not log ErrAddressMissing, transactions can be without to address (for example eth contracts)
			if err != bchain.ErrAddressMissing {
				glog.Warningf("ConnectAssetAllocationOutput sender with asset %v (%v) could not be decoded error %v", assetGuid, assetAllocation.AssetAllocationTuple.WitnessAddress.ToString("sys"), err)
			}
		} else {
			glog.Warningf("ConnectAssetAllocationOutput sender with asset %v (%v) has invalid length: %d", assetGuid, assetAllocation.AssetAllocationTuple.WitnessAddress.ToString("sys"), len(assetSenderAddrDesc))
		}
		return errors.New("ConnectAssetAllocationOutput Skipping asset allocation tx")
	}
	txAddresses.TokenTransfers = make([]*bchain.TokenTransfer, len(assetAllocation.ListSendingAllocationAmounts))
	for i, allocation := range assetAllocation.ListSendingAllocationAmounts {
		receiverAddress := allocation.WitnessAddress.ToString("sys")
		addrDesc, err := d.chainParser.GetAddrDescFromAddress(receiverAddress)
		if err != nil || len(addrDesc) == 0 || len(addrDesc) > maxAddrDescLen {
			if err != nil {
				// do not log ErrAddressMissing, transactions can be without to address (for example eth contracts)
				if err != bchain.ErrAddressMissing {
					glog.Warningf("ConnectAssetAllocationOutput receiver with asset %v (%v) could not be decoded error %v", assetGuid, allocation.WitnessAddress.ToString("sys"), err)
				}
			} else {
				glog.Warningf("ConnectAssetAllocationOutput receiver with asset %v (%v) has invalid length: %d", assetGuid, allocation.WitnessAddress.ToString("sys"), len(addrDesc))
			}
			continue
		}
		receiverStr := string(addrDesc)
		balance, e := balances[receiverStr]
		if !e {
			balance, err = d.GetAddrDescBalance(addrDesc, bchain.AddressBalanceDetailUTXOIndexed)
			if err != nil {
				return err
			}
			if balance == nil {
				balance = &bchain.AddrBalance{}
			}
			balances[receiverStr] = balance
			d.cbs.balancesMiss++
		} else {
			d.cbs.balancesHit++
		}

		// for each address returned, add it to map
		counted := addToAddressesMap(addresses, receiverStr, btxID, outputIndex)
		if !counted {
			balance.Txs++
		}

		if balance.AssetBalances == nil {
			balance.AssetBalances = map[uint32]*bchain.AssetBalance{}
		}
		balanceAsset, ok := balance.AssetBalances[assetGuid]
		if !ok {
			balanceAsset = &bchain.AssetBalance{Transfers: 0, BalanceAssetSat: big.NewInt(0), SentAssetSat: big.NewInt(0), UnallocatedBalanceSat: big.NewInt(0)}
			balance.AssetBalances[assetGuid] = balanceAsset
		}
		balanceAsset.Transfers++
		amount := big.NewInt(allocation.ValueSat)
		balanceAsset.BalanceAssetSat.Add(balanceAsset.BalanceAssetSat, amount)
		totalAssetSentValue.Add(totalAssetSentValue, amount)
		txAddresses.TokenTransfers[i] = &bchain.TokenTransfer {
			Type:     bchain.SPTAssetAllocationType,
			Token:    strAssetGuid,
			From:     senderAddress,
			To:       receiverAddress,
			Decimals: 8,
			Value:    (*bchain.Amount)(amount),
			Symbol:   "SPT",
		}
		if d.chainParser.IsAssetSendTx(version) {
			txAddresses.TokenTransfers[i].Type = bchain.SPTAssetSendType
		}
	}
	return d.ConnectAssetAllocationInput(btxID, assetGuid, version, totalAssetSentValue, assetSenderAddrDesc, balances, addresses, outputIndex)
}

func (d *RocksDB) DisconnectAssetAllocationOutput(sptData []byte, balances map[string]*bchain.AddrBalance, version int32, addresses map[string]struct{}) error {
	r := bytes.NewReader(sptData)
	var assetAllocation wire.AssetAllocationType
	err := assetAllocation.Deserialize(r)
	if err != nil {
		return err
	}
	getAddressBalance := func(addrDesc bchain.AddressDescriptor) (*bchain.AddrBalance, error) {
		var err error
		s := string(addrDesc)
		b, fb := balances[s]
		if !fb {
			b, err = d.GetAddrDescBalance(addrDesc, bchain.AddressBalanceDetailUTXOIndexed)
			if err != nil {
				return nil, err
			}
			balances[s] = b
		}
		return b, nil
	}
	totalAssetSentValue := big.NewInt(0)
	assetGuid := assetAllocation.AssetAllocationTuple.Asset
	assetSenderAddrDesc, err := d.chainParser.GetAddrDescFromAddress(assetAllocation.AssetAllocationTuple.WitnessAddress.ToString("sys"))
	if err != nil || len(assetSenderAddrDesc) == 0 || len(assetSenderAddrDesc) > maxAddrDescLen {
		if err != nil {
			// do not log ErrAddressMissing, transactions can be without to address (for example eth contracts)
			if err != bchain.ErrAddressMissing {
				glog.Warningf("DisconnectAssetAllocationOutput sender with asset %v (%v) could not be decoded error %v", assetGuid, string(assetSenderAddrDesc), err)
			}
		} else {
			glog.Warningf("DisconnectAssetAllocationOutput sender with asset %v (%v) has invalid length: %d", assetGuid, string(assetSenderAddrDesc), len(assetSenderAddrDesc))
		}
		return errors.New("DisconnectAssetAllocationOutput Skipping disconnect asset allocation tx")
	}
	for _, allocation := range assetAllocation.ListSendingAllocationAmounts {
		addrDesc, err := d.chainParser.GetAddrDescFromAddress(allocation.WitnessAddress.ToString("sys"))
		if err != nil || len(addrDesc) == 0 || len(addrDesc) > maxAddrDescLen {
			if err != nil {
				// do not log ErrAddressMissing, transactions can be without to address (for example eth contracts)
				if err != bchain.ErrAddressMissing {
					glog.Warningf("DisconnectAssetAllocationOutput receiver with asset %v (%v) could not be decoded error %v", assetGuid, string(addrDesc), err)
				}
			} else {
				glog.Warningf("DisconnectAssetAllocationOutput receiver with asset %v (%v) has invalid length: %d", assetGuid, string(addrDesc), len(addrDesc))
			}
			continue
		}
		receiverStr := string(addrDesc)
		_, exist := addresses[receiverStr]
		if !exist {
			addresses[receiverStr] = struct{}{}
		}
		balance, err := getAddressBalance(addrDesc)
		if err != nil {
			return err
		}
		if balance != nil {
			// subtract number of txs only once
			if !exist {
				balance.Txs--
			}
		} else {
			ad, _, _ := d.chainParser.GetAddressesFromAddrDesc(addrDesc)
			glog.Warningf("DisconnectAssetAllocationOutput Balance for asset address %v (%v) not found", ad, addrDesc)
		}

		if balance.AssetBalances != nil{
			balanceAsset := balance.AssetBalances[assetGuid]
			balanceAsset.Transfers--
			amount := big.NewInt(allocation.ValueSat)
			balanceAsset.BalanceAssetSat.Sub(balanceAsset.BalanceAssetSat, amount)
			if balanceAsset.BalanceAssetSat.Sign() < 0 {
				d.resetValueSatToZero(balanceAsset.BalanceAssetSat, addrDesc, "balance")
			}
			totalAssetSentValue.Add(totalAssetSentValue, amount)
		} else {
			ad, _, _ := d.chainParser.GetAddressesFromAddrDesc(addrDesc)
			glog.Warningf("DisconnectAssetAllocationOutput Asset Balance for asset address %v (%v) not found", ad, addrDesc)
		}
	}
	return d.DisconnectAssetAllocationInput(assetGuid, version, totalAssetSentValue, assetSenderAddrDesc, balances)
}

func (d *RocksDB) ConnectAssetAllocationInput(btxID []byte, assetGuid uint32, version int32, totalAssetSentValue *big.Int, assetSenderAddrDesc bchain.AddressDescriptor, balances map[string]*bchain.AddrBalance, addresses bchain.AddressesMap, outputIndex int32) error {
	if totalAssetSentValue == nil {
		return errors.New("totalAssetSentValue was nil cannot connect allocation input")
	}
	assetStrSenderAddrDesc := string(assetSenderAddrDesc)
	balance, e := balances[assetStrSenderAddrDesc]
	if !e {
		var err error
		balance, err = d.GetAddrDescBalance(assetSenderAddrDesc, bchain.AddressBalanceDetailUTXOIndexed)
		if err != nil {
			return err
		}
		if balance == nil {
			balance = &bchain.AddrBalance{}
		}
		balances[assetStrSenderAddrDesc] = balance
		d.cbs.balancesMiss++
	} else {
		d.cbs.balancesHit++
	}

	if balance.AssetBalances == nil {
		balance.AssetBalances = map[uint32]*bchain.AssetBalance{}
	}
	balanceAsset, ok := balance.AssetBalances[assetGuid]
	if !ok {
		balanceAsset = &bchain.AssetBalance{Transfers: 0, BalanceAssetSat: big.NewInt(0), SentAssetSat: big.NewInt(0), UnallocatedBalanceSat: big.NewInt(0)}
		balance.AssetBalances[assetGuid] = balanceAsset
	}
	balanceAsset.Transfers++
	var balanceAssetSat *big.Int
	if d.chainParser.IsAssetSendTx(version) {
		balanceAssetSat = balanceAsset.UnallocatedBalanceSat
	} else {
		balanceAssetSat = balanceAsset.BalanceAssetSat
		balanceAsset.SentAssetSat.Add(balanceAsset.SentAssetSat, totalAssetSentValue)
	}
	balanceAssetSat.Sub(balanceAssetSat, totalAssetSentValue)
	if balanceAssetSat.Sign() < 0 {
		d.resetValueSatToZero(balanceAssetSat, assetSenderAddrDesc, "balance")
	}
	
	return nil

}

func (d *RocksDB) DisconnectAssetOutput(sptData []byte, balances map[string]*bchain.AddrBalance, version int32, addresses map[string]struct{}) error {
	r := bytes.NewReader(sptData)
	var asset wire.AssetType
	err := asset.Deserialize(r)
	if err != nil {
		return err
	}
	getAddressBalance := func(addrDesc bchain.AddressDescriptor) (*bchain.AddrBalance, error) {
		var err error
		s := string(addrDesc)
		b, fb := balances[s]
		if !fb {
			b, err = d.GetAddrDescBalance(addrDesc, bchain.AddressBalanceDetailUTXOIndexed)
			if err != nil {
				return nil, err
			}
			balances[s] = b
		}
		return b, nil
	}
	assetGuid := asset.Asset
	assetSenderAddrDesc, err := d.chainParser.GetAddrDescFromAddress(asset.WitnessAddress.ToString("sys"))
	assetStrSenderAddrDesc := string(assetSenderAddrDesc)
	_, exist := addresses[assetStrSenderAddrDesc]
	if !exist {
		addresses[assetStrSenderAddrDesc] = struct{}{}
	}
	balance, err := getAddressBalance(assetSenderAddrDesc)
	if err != nil {
		return err
	}
	if balance == nil {
		ad, _, _ := d.chainParser.GetAddressesFromAddrDesc(assetSenderAddrDesc)
		glog.Warningf("DisconnectAssetOutput Balance for asset address %s (%s) not found", ad, assetSenderAddrDesc)
	}
	if len(asset.WitnessAddressTransfer.WitnessProgram) > 0 {
		assetTransferWitnessAddrDesc, err := d.chainParser.GetAddrDescFromAddress(asset.WitnessAddressTransfer.ToString("sys"))
		transferStr := string(assetTransferWitnessAddrDesc)
		_, exist := addresses[transferStr]
		if !exist {
			addresses[transferStr] = struct{}{}
		}
		balanceTransfer, err := getAddressBalance(assetTransferWitnessAddrDesc)
		if err != nil {
			return err
		}
		if balanceTransfer != nil {
			// subtract number of txs only once
			if !exist {
				balanceTransfer.Txs--
			}
		} else {
			ad, _, _ := d.chainParser.GetAddressesFromAddrDesc(assetTransferWitnessAddrDesc)
			glog.Warningf("DisconnectAssetOutput Balance for transfer asset address %s (%s) not found", ad, assetTransferWitnessAddrDesc)
		}

		balanceAsset := balance.AssetBalances[assetGuid]
		balanceAsset.Transfers--
		balanceTransferAsset := balanceTransfer.AssetBalances[assetGuid]
		balanceTransferAsset.Transfers--
		// transfer balance back to sender
		balanceAsset.UnallocatedBalanceSat.Set(balanceTransferAsset.UnallocatedBalanceSat)
		// clear balance on receiver
		balanceTransferAsset.UnallocatedBalanceSat.Set(big.NewInt(0))		
	} else if balance.AssetBalances != nil {
		balanceAsset := balance.AssetBalances[assetGuid]
		balanceAsset.Transfers--
		balanceAsset.UnallocatedBalanceSat.Sub(balanceAsset.UnallocatedBalanceSat, big.NewInt(asset.Balance))
	} else {
		glog.Warningf("DisconnectAssetOutput: Asset Sent balance not found guid %v (%v)", assetGuid, assetStrSenderAddrDesc)
	}
	return nil

}
func (d *RocksDB) DisconnectAssetAllocationInput(assetGuid uint32, version int32, totalAssetSentValue *big.Int, assetSenderAddrDesc bchain.AddressDescriptor, balances map[string]*bchain.AddrBalance) error {
	assetStrSenderAddrDesc := string(assetSenderAddrDesc)
	balance, e := balances[assetStrSenderAddrDesc]
	if !e {
		var err error
		balance, err = d.GetAddrDescBalance(assetSenderAddrDesc, bchain.AddressBalanceDetailUTXOIndexed)
		if err != nil {
			return err
		}
		if balance == nil {
			return errors.New("DisconnectAssetAllocationInput Asset Balance for sender address not found")
		}
		balances[assetStrSenderAddrDesc] = balance
	}
	if balance.AssetBalances != nil {
		balanceAsset := balance.AssetBalances[assetGuid]
		balanceAsset.Transfers--
		var balanceAssetSat *big.Int
		if d.chainParser.IsAssetSendTx(version) {
			balanceAssetSat = balanceAsset.UnallocatedBalanceSat
		} else {
			balanceAssetSat = balanceAsset.BalanceAssetSat
			balanceAsset.SentAssetSat.Sub(balanceAsset.SentAssetSat, totalAssetSentValue)
			if balanceAsset.SentAssetSat.Sign() < 0 {
				d.resetValueSatToZero(balanceAsset.SentAssetSat, assetSenderAddrDesc, "balance")
			}
		}
		balanceAssetSat.Add(balanceAssetSat, totalAssetSentValue)

	} else {
		glog.Warningf("DisconnectAssetAllocationInput: Asset Sent balance not found guid %v (%v)", assetGuid, assetStrSenderAddrDesc)
	}
	return nil

}
func (d *RocksDB) ConnectMintAssetOutput(sptData []byte, balances map[string]*bchain.AddrBalance, version int32, addresses bchain.AddressesMap, btxID []byte, outputIndex int32, txAddresses* bchain.TxAddresses) error {
	r := bytes.NewReader(sptData)
	var mintasset wire.MintSyscoinType
	err := mintasset.Deserialize(r)
	if err != nil {
		return err
	}
	assetGuid := mintasset.AssetAllocationTuple.Asset
	strAssetGuid := strconv.FormatUint(uint64(assetGuid), 10)
	senderAddress := "burn"
	receiverAddress := mintasset.AssetAllocationTuple.WitnessAddress.ToString("sys")
	assetSenderAddrDesc, err := d.chainParser.GetAddrDescFromAddress(senderAddress)
	if err != nil || len(assetSenderAddrDesc) == 0 || len(assetSenderAddrDesc) > maxAddrDescLen {
		if err != nil {
			// do not log ErrAddressMissing, transactions can be without to address (for example eth contracts)
			if err != bchain.ErrAddressMissing {
				glog.Warningf("ConnectMintAssetOutput sender with asset %v (%v) could not be decoded error %v", assetGuid, receiverAddress, err)
			}
		} else {
			glog.Warningf("ConnectMintAssetOutput sender with asset %v (%v) has invalid length: %d", assetGuid, receiverAddress, len(assetSenderAddrDesc))
		}
		return errors.New("ConnectMintAssetOutput Skipping asset mint tx")
	}
	addrDesc, err := d.chainParser.GetAddrDescFromAddress(receiverAddress)
	if err != nil || len(addrDesc) == 0 || len(addrDesc) > maxAddrDescLen {
		if err != nil {
			// do not log ErrAddressMissing, transactions can be without to address (for example eth contracts)
			if err != bchain.ErrAddressMissing {
				glog.Warningf("ConnectMintAssetOutput receiver with asset %v (%v) could not be decoded error %v", assetGuid, receiverAddress, err)
			}
		} else {
			glog.Warningf("ConnectMintAssetOutput receiver with asset %v (%v) has invalid length: %d", assetGuid, receiverAddress, len(addrDesc))
		}
		return errors.New("ConnectMintAssetOutput Skipping asset mint tx")
	}
	receiverStr := string(addrDesc)
	balance, e := balances[receiverStr]
	if !e {
		balance, err = d.GetAddrDescBalance(addrDesc, bchain.AddressBalanceDetailUTXOIndexed)
		if err != nil {
			return err
		}
		if balance == nil {
			balance = &bchain.AddrBalance{}
		}
		balances[receiverStr] = balance
		d.cbs.balancesMiss++
	} else {
		d.cbs.balancesHit++
	}

	// for each address returned, add it to map
	counted := addToAddressesMap(addresses, receiverStr, btxID, outputIndex)
	if !counted {
		balance.Txs++
	}

	if balance.AssetBalances == nil {
		balance.AssetBalances = map[uint32]*bchain.AssetBalance{}
	}
	balanceAsset, ok := balance.AssetBalances[assetGuid]
	if !ok {
		balanceAsset = &bchain.AssetBalance{Transfers: 0, BalanceAssetSat: big.NewInt(0), SentAssetSat: big.NewInt(0), UnallocatedBalanceSat: big.NewInt(0)}
		balance.AssetBalances[assetGuid] = balanceAsset
	}
	balanceAsset.Transfers++
	amount := big.NewInt(mintasset.ValueAsset)
	balanceAsset.BalanceAssetSat.Add(balanceAsset.BalanceAssetSat, amount)
	txAddresses.TokenTransfers = make([]*bchain.TokenTransfer, 1)
	txAddresses.TokenTransfers[0] = &bchain.TokenTransfer {
		Type:     bchain.SPTAssetMintType,
		Token:    strAssetGuid,
		From:     senderAddress,
		To:       receiverAddress,
		Decimals: 8,
		Value:    (*bchain.Amount)(amount),
		Symbol:   "SPT",
	}
	return d.ConnectAssetAllocationInput(btxID, assetGuid, version, amount, assetSenderAddrDesc, balances, addresses, outputIndex)
}
func (d *RocksDB) DisconnectMintAssetOutput(sptData []byte, balances map[string]*bchain.AddrBalance, version int32, addresses map[string]struct{}) error {
	r := bytes.NewReader(sptData)
	var mintasset wire.MintSyscoinType
	err := mintasset.Deserialize(r)
	if err != nil {
		return err
	}
	getAddressBalance := func(addrDesc bchain.AddressDescriptor) (*bchain.AddrBalance, error) {
		var err error
		s := string(addrDesc)
		b, fb := balances[s]
		if !fb {
			b, err = d.GetAddrDescBalance(addrDesc, bchain.AddressBalanceDetailUTXOIndexed)
			if err != nil {
				return nil, err
			}
			balances[s] = b
		}
		return b, nil
	}
	assetGuid := mintasset.AssetAllocationTuple.Asset
	assetSenderAddrDesc, err := d.chainParser.GetAddrDescFromAddress("burn")
	if err != nil || len(assetSenderAddrDesc) == 0 || len(assetSenderAddrDesc) > maxAddrDescLen {
		if err != nil {
			// do not log ErrAddressMissing, transactions can be without to address (for example eth contracts)
			if err != bchain.ErrAddressMissing {
				glog.Warningf("DisconnectMintAssetOutput sender with asset %v (%v) could not be decoded error %v", assetGuid, string(assetSenderAddrDesc), err)
			}
		} else {
			glog.Warningf("DisconnectMintAssetOutput sender with asset %v (%v) has invalid length: %d", assetGuid, string(assetSenderAddrDesc), len(assetSenderAddrDesc))
		}
		return errors.New("DisconnectMintAssetOutput Skipping disconnect asset mint tx")
	}
	
	addrDesc, err := d.chainParser.GetAddrDescFromAddress(mintasset.AssetAllocationTuple.WitnessAddress.ToString("sys"))
	if err != nil || len(addrDesc) == 0 || len(addrDesc) > maxAddrDescLen {
		if err != nil {
			// do not log ErrAddressMissing, transactions can be without to address (for example eth contracts)
			if err != bchain.ErrAddressMissing {
				glog.Warningf("DisconnectMintAssetOutput receiver with asset %v (%v) could not be decoded error %v", assetGuid, string(addrDesc), err)
			}
		} else {
			glog.Warningf("DisconnectMintAssetOutput receiver with asset %v (%v) has invalid length: %d", assetGuid, string(addrDesc), len(addrDesc))
		}
		return errors.New("DisconnectMintAssetOutput Skipping disconnect asset mint tx")
	}
	receiverStr := string(addrDesc)
	_, exist := addresses[receiverStr]
	if !exist {
		addresses[receiverStr] = struct{}{}
	}
	balance, err := getAddressBalance(addrDesc)
	if err != nil {
		return err
	}
	if balance != nil {
		// subtract number of txs only once
		if !exist {
			balance.Txs--
		}
	} else {
		ad, _, _ := d.chainParser.GetAddressesFromAddrDesc(addrDesc)
		glog.Warningf("DisconnectMintAssetOutput Balance for asset address %v (%v) not found", ad, addrDesc)
	}
	var totalAssetSentValue *big.Int
	if balance.AssetBalances != nil{
		balanceAsset := balance.AssetBalances[assetGuid]
		balanceAsset.Transfers--
		totalAssetSentValue := big.NewInt(mintasset.ValueAsset)
		balanceAsset.BalanceAssetSat.Sub(balanceAsset.BalanceAssetSat, totalAssetSentValue)
		if balanceAsset.BalanceAssetSat.Sign() < 0 {
			d.resetValueSatToZero(balanceAsset.BalanceAssetSat, addrDesc, "balance")
		}
	} else {
		ad, _, _ := d.chainParser.GetAddressesFromAddrDesc(addrDesc)
		glog.Warningf("DisconnectMintAssetOutput Asset Balance for asset address %v (%v) not found", ad, addrDesc)
	}
	
	return d.DisconnectAssetAllocationInput(assetGuid, version, totalAssetSentValue, assetSenderAddrDesc, balances)
}
func (d *RocksDB) ConnectSyscoinOutputs(addrDesc bchain.AddressDescriptor, balances map[string]*bchain.AddrBalance, version int32, addresses bchain.AddressesMap, btxID []byte, outputIndex int32, txAddresses* bchain.TxAddresses) error {
	script, err := d.chainParser.GetScriptFromAddrDesc(addrDesc)
	if err != nil {
		return err
	}
	sptData := d.chainParser.TryGetOPReturn(script)
	if sptData == nil {
		return nil
	}
	if d.chainParser.IsAssetAllocationTx(version) {
		return d.ConnectAssetAllocationOutput(sptData, balances, version, addresses, btxID, outputIndex, txAddresses)
	} else if d.chainParser.IsAssetTx(version) {
		return d.ConnectAssetOutput(sptData, balances, version, addresses, btxID, outputIndex, txAddresses)
	} else if d.chainParser.IsSyscoinMintTx(version) {
		return d.ConnectMintAssetOutput(sptData, balances, version, addresses, btxID, outputIndex, txAddresses)
	}
	return nil
}

func (d *RocksDB) DisconnectSyscoinOutputs(addrDesc bchain.AddressDescriptor, balances map[string]*bchain.AddrBalance, version int32, addresses map[string]struct{}) error {
	script, err := d.chainParser.GetScriptFromAddrDesc(addrDesc)
	if err != nil {
		return err
	}
	sptData := d.chainParser.TryGetOPReturn(script)
	if sptData == nil {
		return nil
	}
	if d.chainParser.IsAssetAllocationTx(version) {
		return d.DisconnectAssetAllocationOutput(sptData, balances, version, addresses)
	} else if d.chainParser.IsAssetTx(version) {
		return d.DisconnectAssetOutput(sptData, balances, version, addresses)
	} else if d.chainParser.IsSyscoinMintTx(version) {
		return d.DisconnectMintAssetOutput(sptData, balances, version, addresses)
	}
	return nil
}