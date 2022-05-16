package metachain

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"sort"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/core/atomic"
	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-go-core/data"
	"github.com/ElrondNetwork/elrond-go-core/data/block"
	"github.com/ElrondNetwork/elrond-go-core/marshal"
	"github.com/ElrondNetwork/elrond-go/common"
	vInfo "github.com/ElrondNetwork/elrond-go/common/validatorInfo"
	"github.com/ElrondNetwork/elrond-go/epochStart"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/sharding"
	"github.com/ElrondNetwork/elrond-go/sharding/nodesCoordinator"
	"github.com/ElrondNetwork/elrond-go/state"
	"github.com/ElrondNetwork/elrond-go/vm"
	"github.com/ElrondNetwork/elrond-go/vm/systemSmartContracts"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
)

type legacySystemSCProcessor struct {
	systemVM                     vmcommon.VMExecutionHandler
	userAccountsDB               state.AccountsAdapter
	marshalizer                  marshal.Marshalizer
	peerAccountsDB               state.AccountsAdapter
	chanceComputer               nodesCoordinator.ChanceComputer
	shardCoordinator             sharding.Coordinator
	startRating                  uint32
	validatorInfoCreator         epochStart.ValidatorInfoCreator
	genesisNodesConfig           sharding.GenesisNodesSetupHandler
	nodesConfigProvider          epochStart.NodesConfigProvider
	stakingDataProvider          epochStart.StakingDataProvider
	maxNodesChangeConfigProvider epochStart.MaxNodesChangeConfigProvider
	endOfEpochCallerAddress      []byte
	stakingSCAddress             []byte
	esdtOwnerAddressBytes        []byte
	mapNumSwitchedPerShard       map[uint32]uint32
	mapNumSwitchablePerShard     map[uint32]uint32
	maxNodes                     uint32

	switchEnableEpoch           uint32
	hystNodesEnableEpoch        uint32
	delegationEnableEpoch       uint32
	stakingV2EnableEpoch        uint32
	correctLastUnJailEpoch      uint32
	esdtEnableEpoch             uint32
	saveJailedAlwaysEnableEpoch uint32
	stakingV4InitEnableEpoch    uint32

	flagSwitchJailedWaiting        atomic.Flag
	flagHystNodesEnabled           atomic.Flag
	flagDelegationEnabled          atomic.Flag
	flagSetOwnerEnabled            atomic.Flag
	flagChangeMaxNodesEnabled      atomic.Flag
	flagStakingV2Enabled           atomic.Flag
	flagCorrectLastUnjailedEnabled atomic.Flag
	flagCorrectNumNodesToStake     atomic.Flag
	flagESDTEnabled                atomic.Flag
	flagSaveJailedAlwaysEnabled    atomic.Flag
	flagStakingQueueEnabled        atomic.Flag
}

func newLegacySystemSCProcessor(args ArgsNewEpochStartSystemSCProcessing) (*legacySystemSCProcessor, error) {
	err := checkLegacyArgs(args)
	if err != nil {
		return nil, err
	}

	legacy := &legacySystemSCProcessor{
		systemVM:                     args.SystemVM,
		userAccountsDB:               args.UserAccountsDB,
		peerAccountsDB:               args.PeerAccountsDB,
		marshalizer:                  args.Marshalizer,
		startRating:                  args.StartRating,
		validatorInfoCreator:         args.ValidatorInfoCreator,
		genesisNodesConfig:           args.GenesisNodesConfig,
		endOfEpochCallerAddress:      args.EndOfEpochCallerAddress,
		stakingSCAddress:             args.StakingSCAddress,
		chanceComputer:               args.ChanceComputer,
		mapNumSwitchedPerShard:       make(map[uint32]uint32),
		mapNumSwitchablePerShard:     make(map[uint32]uint32),
		switchEnableEpoch:            args.EpochConfig.EnableEpochs.SwitchJailWaitingEnableEpoch,
		hystNodesEnableEpoch:         args.EpochConfig.EnableEpochs.SwitchHysteresisForMinNodesEnableEpoch,
		delegationEnableEpoch:        args.EpochConfig.EnableEpochs.DelegationSmartContractEnableEpoch,
		stakingV2EnableEpoch:         args.EpochConfig.EnableEpochs.StakingV2EnableEpoch,
		esdtEnableEpoch:              args.EpochConfig.EnableEpochs.ESDTEnableEpoch,
		stakingDataProvider:          args.StakingDataProvider,
		nodesConfigProvider:          args.NodesConfigProvider,
		shardCoordinator:             args.ShardCoordinator,
		correctLastUnJailEpoch:       args.EpochConfig.EnableEpochs.CorrectLastUnjailedEnableEpoch,
		esdtOwnerAddressBytes:        args.ESDTOwnerAddressBytes,
		saveJailedAlwaysEnableEpoch:  args.EpochConfig.EnableEpochs.SaveJailedAlwaysEnableEpoch,
		stakingV4InitEnableEpoch:     args.EpochConfig.EnableEpochs.StakingV4InitEnableEpoch,
		maxNodesChangeConfigProvider: args.MaxNodesChangeConfigProvider,
	}

	log.Debug("legacySystemSC: enable epoch for switch jail waiting", "epoch", legacy.switchEnableEpoch)
	log.Debug("legacySystemSC: enable epoch for switch hysteresis for min nodes", "epoch", legacy.hystNodesEnableEpoch)
	log.Debug("legacySystemSC: enable epoch for delegation manager", "epoch", legacy.delegationEnableEpoch)
	log.Debug("legacySystemSC: enable epoch for staking v2", "epoch", legacy.stakingV2EnableEpoch)
	log.Debug("legacySystemSC: enable epoch for ESDT", "epoch", legacy.esdtEnableEpoch)
	log.Debug("legacySystemSC: enable epoch for correct last unjailed", "epoch", legacy.correctLastUnJailEpoch)
	log.Debug("legacySystemSC: enable epoch for save jailed always", "epoch", legacy.saveJailedAlwaysEnableEpoch)
	log.Debug("legacySystemSC: enable epoch for initializing staking v4", "epoch", legacy.stakingV4InitEnableEpoch)

	return legacy, nil
}

func checkLegacyArgs(args ArgsNewEpochStartSystemSCProcessing) error {
	if check.IfNilReflect(args.SystemVM) {
		return epochStart.ErrNilSystemVM
	}
	if check.IfNil(args.UserAccountsDB) {
		return epochStart.ErrNilAccountsDB
	}
	if check.IfNil(args.PeerAccountsDB) {
		return epochStart.ErrNilAccountsDB
	}
	if check.IfNil(args.Marshalizer) {
		return epochStart.ErrNilMarshalizer
	}
	if check.IfNil(args.ValidatorInfoCreator) {
		return epochStart.ErrNilValidatorInfoProcessor
	}
	if len(args.EndOfEpochCallerAddress) == 0 {
		return epochStart.ErrNilEndOfEpochCallerAddress
	}
	if len(args.StakingSCAddress) == 0 {
		return epochStart.ErrNilStakingSCAddress
	}
	if check.IfNil(args.ChanceComputer) {
		return epochStart.ErrNilChanceComputer
	}
	if check.IfNil(args.GenesisNodesConfig) {
		return epochStart.ErrNilGenesisNodesConfig
	}
	if check.IfNil(args.NodesConfigProvider) {
		return epochStart.ErrNilNodesConfigProvider
	}
	if check.IfNil(args.StakingDataProvider) {
		return epochStart.ErrNilStakingDataProvider
	}
	if check.IfNil(args.ShardCoordinator) {
		return epochStart.ErrNilShardCoordinator
	}
	if check.IfNil(args.MaxNodesChangeConfigProvider) {
		return epochStart.ErrNilMaxNodesChangeConfigProvider
	}
	if len(args.ESDTOwnerAddressBytes) == 0 {
		return epochStart.ErrEmptyESDTOwnerAddress
	}

	return nil
}

func (s *legacySystemSCProcessor) processLegacy(
	validatorsInfoMap state.ShardValidatorsInfoMapHandler,
	nonce uint64,
	epoch uint32,
) error {
	if s.flagHystNodesEnabled.IsSet() {
		err := s.updateSystemSCConfigMinNodes()
		if err != nil {
			return err
		}
	}

	if s.flagSetOwnerEnabled.IsSet() {
		err := s.updateOwnersForBlsKeys()
		if err != nil {
			return err
		}
	}

	if s.flagChangeMaxNodesEnabled.IsSet() {
		err := s.updateMaxNodes(validatorsInfoMap, nonce)
		if err != nil {
			return err
		}
	}

	if s.flagCorrectLastUnjailedEnabled.IsSet() {
		err := s.resetLastUnJailed()
		if err != nil {
			return err
		}
	}

	if s.flagDelegationEnabled.IsSet() {
		err := s.initDelegationSystemSC()
		if err != nil {
			return err
		}
	}

	if s.flagCorrectNumNodesToStake.IsSet() {
		err := s.cleanAdditionalQueue()
		if err != nil {
			return err
		}
	}

	if s.flagSwitchJailedWaiting.IsSet() {
		err := s.computeNumWaitingPerShard(validatorsInfoMap)
		if err != nil {
			return err
		}

		err = s.swapJailedWithWaiting(validatorsInfoMap)
		if err != nil {
			return err
		}
	}

	if s.flagStakingV2Enabled.IsSet() {
		err := s.prepareStakingDataForEligibleNodes(validatorsInfoMap)
		if err != nil {
			return err
		}

		err = s.fillStakingDataForNonEligible(validatorsInfoMap)
		if err != nil {
			return err
		}

		numUnStaked, err := s.unStakeNodesWithNotEnoughFunds(validatorsInfoMap, epoch)
		if err != nil {
			return err
		}

		if s.flagStakingQueueEnabled.IsSet() {
			err = s.stakeNodesFromQueue(validatorsInfoMap, numUnStaked, nonce, common.NewList)
			if err != nil {
				return err
			}
		}
	}

	if s.flagESDTEnabled.IsSet() {
		err := s.initESDT()
		if err != nil {
			// not a critical error
			log.Error("error while initializing ESDT", "err", err)
		}
	}

	return nil
}

// ToggleUnStakeUnBond will pause/unPause the unStake/unBond functions on the validator system sc
func (s *legacySystemSCProcessor) ToggleUnStakeUnBond(value bool) error {
	if !s.flagStakingV2Enabled.IsSet() {
		return nil
	}

	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr: s.endOfEpochCallerAddress,
			Arguments:  nil,
			CallValue:  big.NewInt(0),
		},
		RecipientAddr: vm.ValidatorSCAddress,
		Function:      "unPauseUnStakeUnBond",
	}

	if value {
		vmInput.Function = "pauseUnStakeUnBond"
	}

	vmOutput, err := s.systemVM.RunSmartContractCall(vmInput)
	if err != nil {
		return err
	}

	if vmOutput.ReturnCode != vmcommon.Ok {
		return epochStart.ErrSystemValidatorSCCall
	}

	err = s.processSCOutputAccounts(vmOutput)
	if err != nil {
		return err
	}

	return nil
}

func (s *legacySystemSCProcessor) unStakeNodesWithNotEnoughFunds(
	validatorsInfoMap state.ShardValidatorsInfoMapHandler,
	epoch uint32,
) (uint32, error) {
	nodesToUnStake, mapOwnersKeys, err := s.stakingDataProvider.ComputeUnQualifiedNodes(validatorsInfoMap)
	if err != nil {
		return 0, err
	}

	nodesUnStakedFromAdditionalQueue := uint32(0)

	log.Debug("unStake nodes with not enough funds", "num", len(nodesToUnStake))
	for _, blsKey := range nodesToUnStake {
		log.Debug("unStake at end of epoch for node", "blsKey", blsKey)
		err = s.unStakeOneNode(blsKey, epoch)
		if err != nil {
			return 0, err
		}

		validatorInfo := validatorsInfoMap.GetValidator(blsKey)
		if validatorInfo == nil {
			nodesUnStakedFromAdditionalQueue++
			log.Debug("unStaked node which was in additional queue", "blsKey", blsKey)
			continue
		}

		validatorLeaving := validatorInfo.ShallowClone()
		validatorLeaving.SetList(string(common.LeavingList))
		err = validatorsInfoMap.Replace(validatorInfo, validatorLeaving)
		if err != nil {
			return 0, err
		}
	}

	err = s.updateDelegationContracts(mapOwnersKeys)
	if err != nil {
		return 0, err
	}

	nodesToStakeFromQueue := uint32(len(nodesToUnStake))
	nodesToStakeFromQueue -= nodesUnStakedFromAdditionalQueue

	log.Debug("stake nodes from waiting list", "num", nodesToStakeFromQueue)
	return nodesToStakeFromQueue, nil
}

func (s *legacySystemSCProcessor) unStakeOneNode(blsKey []byte, epoch uint32) error {
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr: s.endOfEpochCallerAddress,
			Arguments:  [][]byte{blsKey},
			CallValue:  big.NewInt(0),
		},
		RecipientAddr: s.stakingSCAddress,
		Function:      "unStakeAtEndOfEpoch",
	}

	vmOutput, err := s.systemVM.RunSmartContractCall(vmInput)
	if err != nil {
		return err
	}
	if vmOutput.ReturnCode != vmcommon.Ok {
		log.Debug("unStakeOneNode", "returnMessage", vmOutput.ReturnMessage, "returnCode", vmOutput.ReturnCode.String())
		return epochStart.ErrUnStakeExecuteError
	}

	err = s.processSCOutputAccounts(vmOutput)
	if err != nil {
		return err
	}

	account, errExists := s.peerAccountsDB.GetExistingAccount(blsKey)
	if errExists != nil {
		return nil
	}

	peerAccount, ok := account.(state.PeerAccountHandler)
	if !ok {
		return epochStart.ErrWrongTypeAssertion
	}

	peerAccount.SetListAndIndex(peerAccount.GetShardId(), string(common.LeavingList), peerAccount.GetIndexInList())
	peerAccount.SetUnStakedEpoch(epoch)
	err = s.peerAccountsDB.SaveAccount(peerAccount)
	if err != nil {
		return err
	}

	return nil
}

func (s *legacySystemSCProcessor) updateDelegationContracts(mapOwnerKeys map[string][][]byte) error {
	sortedDelegationsSCs := make([]string, 0, len(mapOwnerKeys))
	for address := range mapOwnerKeys {
		shardId := s.shardCoordinator.ComputeId([]byte(address))
		if shardId != core.MetachainShardId {
			continue
		}
		sortedDelegationsSCs = append(sortedDelegationsSCs, address)
	}

	sort.Slice(sortedDelegationsSCs, func(i, j int) bool {
		return sortedDelegationsSCs[i] < sortedDelegationsSCs[j]
	})

	for _, address := range sortedDelegationsSCs {
		vmInput := &vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallerAddr: s.endOfEpochCallerAddress,
				Arguments:  mapOwnerKeys[address],
				CallValue:  big.NewInt(0),
			},
			RecipientAddr: []byte(address),
			Function:      "unStakeAtEndOfEpoch",
		}

		vmOutput, err := s.systemVM.RunSmartContractCall(vmInput)
		if err != nil {
			return err
		}
		if vmOutput.ReturnCode != vmcommon.Ok {
			log.Debug("unStakeAtEndOfEpoch", "returnMessage", vmOutput.ReturnMessage, "returnCode", vmOutput.ReturnCode.String())
			return epochStart.ErrUnStakeExecuteError
		}

		err = s.processSCOutputAccounts(vmOutput)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *legacySystemSCProcessor) fillStakingDataForNonEligible(validatorsInfoMap state.ShardValidatorsInfoMapHandler) error {
	for shId, validatorsInfoSlice := range validatorsInfoMap.GetShardValidatorsInfoMap() {
		newList := make([]state.ValidatorInfoHandler, 0, len(validatorsInfoSlice))
		deleteCalled := false

		for _, validatorInfo := range validatorsInfoSlice {
			if vInfo.WasEligibleInCurrentEpoch(validatorInfo) {
				newList = append(newList, validatorInfo)
				continue
			}

			err := s.stakingDataProvider.FillValidatorInfo(validatorInfo.GetPublicKey())
			if err != nil {
				deleteCalled = true

				log.Error("fillStakingDataForNonEligible", "error", err)
				if len(validatorInfo.GetList()) > 0 {
					return err
				}

				err = s.peerAccountsDB.RemoveAccount(validatorInfo.GetPublicKey())
				if err != nil {
					log.Error("fillStakingDataForNonEligible removeAccount", "error", err)
				}

				continue
			}

			newList = append(newList, validatorInfo)
		}

		if deleteCalled {
			err := validatorsInfoMap.SetValidatorsInShard(shId, newList)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *legacySystemSCProcessor) prepareStakingDataForEligibleNodes(validatorsInfoMap state.ShardValidatorsInfoMapHandler) error {
	eligibleNodes := s.getEligibleNodeKeys(validatorsInfoMap)
	return s.prepareStakingData(eligibleNodes)
}

func (s *legacySystemSCProcessor) prepareStakingData(nodeKeys map[uint32][][]byte) error {
	sw := core.NewStopWatch()
	sw.Start("prepareStakingDataForRewards")
	defer func() {
		sw.Stop("prepareStakingDataForRewards")
		log.Debug("systemSCProcessor.prepareStakingDataForRewards time measurements", sw.GetMeasurements()...)
	}()

	return s.stakingDataProvider.PrepareStakingData(nodeKeys)
}

func (s *legacySystemSCProcessor) getEligibleNodeKeys(
	validatorsInfoMap state.ShardValidatorsInfoMapHandler,
) map[uint32][][]byte {
	eligibleNodesKeys := make(map[uint32][][]byte)
	for shardID, validatorsInfoSlice := range validatorsInfoMap.GetShardValidatorsInfoMap() {
		eligibleNodesKeys[shardID] = make([][]byte, 0, s.nodesConfigProvider.ConsensusGroupSize(shardID))
		for _, validatorInfo := range validatorsInfoSlice {
			if vInfo.WasEligibleInCurrentEpoch(validatorInfo) {
				eligibleNodesKeys[shardID] = append(eligibleNodesKeys[shardID], validatorInfo.GetPublicKey())
			}
		}
	}

	return eligibleNodesKeys
}

// ProcessDelegationRewards will process the rewards which are directed towards the delegation system smart contracts
func (s *legacySystemSCProcessor) ProcessDelegationRewards(
	miniBlocks block.MiniBlockSlice,
	txCache epochStart.TransactionCacher,
) error {
	if txCache == nil {
		return epochStart.ErrNilLocalTxCache
	}

	rwdMb := getRewardsMiniBlockForMeta(miniBlocks)
	if rwdMb == nil {
		return nil
	}

	for _, txHash := range rwdMb.TxHashes {
		rwdTx, err := txCache.GetTx(txHash)
		if err != nil {
			return err
		}

		err = s.executeRewardTx(rwdTx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *legacySystemSCProcessor) executeRewardTx(rwdTx data.TransactionHandler) error {
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr: s.endOfEpochCallerAddress,
			Arguments:  nil,
			CallValue:  rwdTx.GetValue(),
		},
		RecipientAddr: rwdTx.GetRcvAddr(),
		Function:      "updateRewards",
	}

	vmOutput, err := s.systemVM.RunSmartContractCall(vmInput)
	if err != nil {
		return err
	}

	if vmOutput.ReturnCode != vmcommon.Ok {
		return epochStart.ErrSystemDelegationCall
	}

	err = s.processSCOutputAccounts(vmOutput)
	if err != nil {
		return err
	}

	return nil
}

// updates the configuration of the system SC if the flags permit
func (s *legacySystemSCProcessor) updateSystemSCConfigMinNodes() error {
	minNumberOfNodesWithHysteresis := s.genesisNodesConfig.MinNumberOfNodesWithHysteresis()
	err := s.setMinNumberOfNodes(minNumberOfNodesWithHysteresis)

	return err
}

func (s *legacySystemSCProcessor) resetLastUnJailed() error {
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr: s.endOfEpochCallerAddress,
			Arguments:  [][]byte{},
			CallValue:  big.NewInt(0),
		},
		RecipientAddr: s.stakingSCAddress,
		Function:      "resetLastUnJailedFromQueue",
	}

	vmOutput, err := s.systemVM.RunSmartContractCall(vmInput)
	if err != nil {
		return err
	}

	if vmOutput.ReturnCode != vmcommon.Ok {
		return epochStart.ErrResetLastUnJailedFromQueue
	}

	err = s.processSCOutputAccounts(vmOutput)
	if err != nil {
		return err
	}

	return nil
}

// updates the configuration of the system SC if the flags permit
func (s *legacySystemSCProcessor) updateMaxNodes(validatorsInfoMap state.ShardValidatorsInfoMapHandler, nonce uint64) error {
	sw := core.NewStopWatch()
	sw.Start("total")
	defer func() {
		sw.Stop("total")
		log.Debug("systemSCProcessor.updateMaxNodes", sw.GetMeasurements()...)
	}()

	maxNumberOfNodes := s.maxNodes
	sw.Start("setMaxNumberOfNodes")
	prevMaxNumberOfNodes, err := s.setMaxNumberOfNodes(maxNumberOfNodes)
	sw.Stop("setMaxNumberOfNodes")
	if err != nil {
		return err
	}

	if s.flagStakingQueueEnabled.IsSet() {
		sw.Start("stakeNodesFromQueue")
		err = s.stakeNodesFromQueue(validatorsInfoMap, maxNumberOfNodes-prevMaxNumberOfNodes, nonce, common.NewList)
		sw.Stop("stakeNodesFromQueue")
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *legacySystemSCProcessor) computeNumWaitingPerShard(validatorsInfoMap state.ShardValidatorsInfoMapHandler) error {
	for shardID, validatorInfoList := range validatorsInfoMap.GetShardValidatorsInfoMap() {
		totalInWaiting := uint32(0)
		for _, validatorInfo := range validatorInfoList {
			switch validatorInfo.GetList() {
			case string(common.WaitingList):
				totalInWaiting++
			}
		}
		s.mapNumSwitchablePerShard[shardID] = totalInWaiting
		s.mapNumSwitchedPerShard[shardID] = 0
	}
	return nil
}

func (s *legacySystemSCProcessor) swapJailedWithWaiting(validatorsInfoMap state.ShardValidatorsInfoMapHandler) error {
	jailedValidators := s.getSortedJailedNodes(validatorsInfoMap)

	log.Debug("number of jailed validators", "num", len(jailedValidators))

	newValidators := make(map[string]struct{})
	for _, jailedValidator := range jailedValidators {
		if _, ok := newValidators[string(jailedValidator.GetPublicKey())]; ok {
			continue
		}
		if isValidator(jailedValidator) && s.mapNumSwitchablePerShard[jailedValidator.GetShardId()] <= s.mapNumSwitchedPerShard[jailedValidator.GetShardId()] {
			log.Debug("cannot switch in this epoch anymore for this shard as switched num waiting",
				"shardID", jailedValidator.GetShardId(),
				"numSwitched", s.mapNumSwitchedPerShard[jailedValidator.GetShardId()])
			continue
		}

		vmInput := &vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallerAddr: s.endOfEpochCallerAddress,
				Arguments:  [][]byte{jailedValidator.GetPublicKey()},
				CallValue:  big.NewInt(0),
			},
			RecipientAddr: s.stakingSCAddress,
			Function:      "switchJailedWithWaiting",
		}

		vmOutput, err := s.systemVM.RunSmartContractCall(vmInput)
		if err != nil {
			return err
		}

		log.Debug("switchJailedWithWaiting called for",
			"key", jailedValidator.GetPublicKey(),
			"returnMessage", vmOutput.ReturnMessage)
		if vmOutput.ReturnCode != vmcommon.Ok {
			continue
		}

		newValidator, err := s.stakingToValidatorStatistics(validatorsInfoMap, jailedValidator, vmOutput)
		if err != nil {
			return err
		}

		if len(newValidator) != 0 {
			newValidators[string(newValidator)] = struct{}{}
		}
	}

	return nil
}

func (s *legacySystemSCProcessor) stakingToValidatorStatistics(
	validatorsInfoMap state.ShardValidatorsInfoMapHandler,
	jailedValidator state.ValidatorInfoHandler,
	vmOutput *vmcommon.VMOutput,
) ([]byte, error) {
	stakingSCOutput, ok := vmOutput.OutputAccounts[string(s.stakingSCAddress)]
	if !ok {
		return nil, epochStart.ErrStakingSCOutputAccountNotFound
	}

	var activeStorageUpdate *vmcommon.StorageUpdate
	for _, storageUpdate := range stakingSCOutput.StorageUpdates {
		isNewValidatorKey := len(storageUpdate.Offset) == len(jailedValidator.GetPublicKey()) &&
			!bytes.Equal(storageUpdate.Offset, jailedValidator.GetPublicKey())
		if isNewValidatorKey {
			activeStorageUpdate = storageUpdate
			break
		}
	}
	if activeStorageUpdate == nil {
		log.Debug("no one in waiting suitable for switch")
		if s.flagSaveJailedAlwaysEnabled.IsSet() {
			err := s.processSCOutputAccounts(vmOutput)
			if err != nil {
				return nil, err
			}
		}

		return nil, nil
	}

	err := s.processSCOutputAccounts(vmOutput)
	if err != nil {
		return nil, err
	}

	var stakingData systemSmartContracts.StakedDataV2_0
	err = s.marshalizer.Unmarshal(&stakingData, activeStorageUpdate.Data)
	if err != nil {
		return nil, err
	}

	blsPubKey := activeStorageUpdate.Offset
	log.Debug("staking validator key who switches with the jailed one", "blsKey", blsPubKey)
	account, err := s.getPeerAccount(blsPubKey)
	if err != nil {
		return nil, err
	}

	if !bytes.Equal(account.GetRewardAddress(), stakingData.RewardAddress) {
		err = account.SetRewardAddress(stakingData.RewardAddress)
		if err != nil {
			return nil, err
		}
	}

	if !bytes.Equal(account.GetBLSPublicKey(), blsPubKey) {
		err = account.SetBLSPublicKey(blsPubKey)
		if err != nil {
			return nil, err
		}
	} else {
		// old jailed validator getting switched back after unJail with stake - must remove first from exported map
		err = validatorsInfoMap.Delete(jailedValidator)
		if err != nil {
			return nil, err
		}
	}

	account.SetListAndIndex(jailedValidator.GetShardId(), string(common.NewList), uint32(stakingData.StakedNonce))
	account.SetTempRating(s.startRating)
	account.SetUnStakedEpoch(common.DefaultUnstakedEpoch)

	err = s.peerAccountsDB.SaveAccount(account)
	if err != nil {
		return nil, err
	}

	jailedAccount, err := s.getPeerAccount(jailedValidator.GetPublicKey())
	if err != nil {
		return nil, err
	}

	jailedAccount.SetListAndIndex(jailedValidator.GetShardId(), string(common.JailedList), jailedValidator.GetIndex())
	jailedAccount.ResetAtNewEpoch()
	err = s.peerAccountsDB.SaveAccount(jailedAccount)
	if err != nil {
		return nil, err
	}

	if isValidator(jailedValidator) {
		s.mapNumSwitchedPerShard[jailedValidator.GetShardId()]++
	}

	newValidatorInfo := s.validatorInfoCreator.PeerAccountToValidatorInfo(account)
	err = validatorsInfoMap.Replace(jailedValidator, newValidatorInfo)
	if err != nil {
		return nil, err
	}

	return blsPubKey, nil
}

func isValidator(validator state.ValidatorInfoHandler) bool {
	return validator.GetList() == string(common.WaitingList) || validator.GetList() == string(common.EligibleList)
}

func (s *legacySystemSCProcessor) getUserAccount(address []byte) (state.UserAccountHandler, error) {
	acnt, err := s.userAccountsDB.LoadAccount(address)
	if err != nil {
		return nil, err
	}

	stAcc, ok := acnt.(state.UserAccountHandler)
	if !ok {
		return nil, process.ErrWrongTypeAssertion
	}

	return stAcc, nil
}

// save account changes in state from vmOutput - protected by VM - every output can be treated as is.
func (s *legacySystemSCProcessor) processSCOutputAccounts(
	vmOutput *vmcommon.VMOutput,
) error {

	outputAccounts := process.SortVMOutputInsideData(vmOutput)
	for _, outAcc := range outputAccounts {
		acc, err := s.getUserAccount(outAcc.Address)
		if err != nil {
			return err
		}

		storageUpdates := process.GetSortedStorageUpdates(outAcc)
		for _, storeUpdate := range storageUpdates {
			err = acc.DataTrieTracker().SaveKeyValue(storeUpdate.Offset, storeUpdate.Data)
			if err != nil {
				return err
			}
		}

		if outAcc.BalanceDelta != nil && outAcc.BalanceDelta.Cmp(zero) != 0 {
			err = acc.AddToBalance(outAcc.BalanceDelta)
			if err != nil {
				return err
			}
		}

		err = s.userAccountsDB.SaveAccount(acc)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *legacySystemSCProcessor) getSortedJailedNodes(validatorsInfoMap state.ShardValidatorsInfoMapHandler) []state.ValidatorInfoHandler {
	newJailedValidators := make([]state.ValidatorInfoHandler, 0)
	oldJailedValidators := make([]state.ValidatorInfoHandler, 0)

	minChance := s.chanceComputer.GetChance(0)
	for _, validatorInfo := range validatorsInfoMap.GetAllValidatorsInfo() {
		if validatorInfo.GetList() == string(common.JailedList) {
			oldJailedValidators = append(oldJailedValidators, validatorInfo)
		} else if s.chanceComputer.GetChance(validatorInfo.GetTempRating()) < minChance {
			newJailedValidators = append(newJailedValidators, validatorInfo)
		}

	}

	sort.Sort(validatorList(oldJailedValidators))
	sort.Sort(validatorList(newJailedValidators))

	return append(oldJailedValidators, newJailedValidators...)
}

func (s *legacySystemSCProcessor) getPeerAccount(key []byte) (state.PeerAccountHandler, error) {
	account, err := s.peerAccountsDB.LoadAccount(key)
	if err != nil {
		return nil, err
	}

	peerAcc, ok := account.(state.PeerAccountHandler)
	if !ok {
		return nil, epochStart.ErrWrongTypeAssertion
	}

	return peerAcc, nil
}

func (s *legacySystemSCProcessor) setMinNumberOfNodes(minNumNodes uint32) error {
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr: s.endOfEpochCallerAddress,
			Arguments:  [][]byte{big.NewInt(int64(minNumNodes)).Bytes()},
			CallValue:  big.NewInt(0),
		},
		RecipientAddr: s.stakingSCAddress,
		Function:      "updateConfigMinNodes",
	}

	vmOutput, err := s.systemVM.RunSmartContractCall(vmInput)
	if err != nil {
		return err
	}

	log.Debug("setMinNumberOfNodes called with",
		"minNumNodes", minNumNodes,
		"returnMessage", vmOutput.ReturnMessage)

	if vmOutput.ReturnCode != vmcommon.Ok {
		return epochStart.ErrInvalidMinNumberOfNodes
	}

	err = s.processSCOutputAccounts(vmOutput)
	if err != nil {
		return err
	}

	return nil
}

func (s *legacySystemSCProcessor) setMaxNumberOfNodes(maxNumNodes uint32) (uint32, error) {
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr: s.endOfEpochCallerAddress,
			Arguments:  [][]byte{big.NewInt(int64(maxNumNodes)).Bytes()},
			CallValue:  big.NewInt(0),
		},
		RecipientAddr: s.stakingSCAddress,
		Function:      "updateConfigMaxNodes",
	}

	vmOutput, err := s.systemVM.RunSmartContractCall(vmInput)
	if err != nil {
		return 0, err
	}

	log.Debug("setMaxNumberOfNodes called with",
		"maxNumNodes", maxNumNodes,
		"current maxNumNodes in legacySystemSCProcessor", s.maxNodes,
		"returnMessage", vmOutput.ReturnMessage)

	if vmOutput.ReturnCode != vmcommon.Ok {
		return 0, epochStart.ErrInvalidMaxNumberOfNodes
	}
	if len(vmOutput.ReturnData) != 1 {
		return 0, epochStart.ErrInvalidSystemSCReturn
	}

	err = s.processSCOutputAccounts(vmOutput)
	if err != nil {
		return 0, err
	}

	prevMaxNumNodes := big.NewInt(0).SetBytes(vmOutput.ReturnData[0]).Uint64()
	return uint32(prevMaxNumNodes), nil
}

func (s *legacySystemSCProcessor) updateOwnersForBlsKeys() error {
	sw := core.NewStopWatch()
	sw.Start("systemSCProcessor")
	defer func() {
		sw.Stop("systemSCProcessor")
		log.Debug("systemSCProcessor.updateOwnersForBlsKeys time measurements", sw.GetMeasurements()...)
	}()

	sw.Start("getValidatorSystemAccount")
	userValidatorAccount, err := s.getValidatorSystemAccount()
	sw.Stop("getValidatorSystemAccount")
	if err != nil {
		return err
	}

	sw.Start("getArgumentsForSetOwnerFunctionality")
	arguments, err := s.getArgumentsForSetOwnerFunctionality(userValidatorAccount)
	sw.Stop("getArgumentsForSetOwnerFunctionality")
	if err != nil {
		return err
	}

	sw.Start("callSetOwnersOnAddresses")
	err = s.callSetOwnersOnAddresses(arguments)
	sw.Stop("callSetOwnersOnAddresses")
	if err != nil {
		return err
	}

	return nil
}

func (s *legacySystemSCProcessor) getValidatorSystemAccount() (state.UserAccountHandler, error) {
	validatorAccount, err := s.userAccountsDB.LoadAccount(vm.ValidatorSCAddress)
	if err != nil {
		return nil, fmt.Errorf("%w when loading validator account", err)
	}

	userValidatorAccount, ok := validatorAccount.(state.UserAccountHandler)
	if !ok {
		return nil, fmt.Errorf("%w when loading validator account", epochStart.ErrWrongTypeAssertion)
	}

	if check.IfNil(userValidatorAccount.DataTrie()) {
		return nil, epochStart.ErrNilDataTrie
	}

	return userValidatorAccount, nil
}

func (s *legacySystemSCProcessor) getArgumentsForSetOwnerFunctionality(userValidatorAccount state.UserAccountHandler) ([][]byte, error) {
	arguments := make([][]byte, 0)

	rootHash, err := userValidatorAccount.DataTrie().RootHash()
	if err != nil {
		return nil, err
	}

	chLeaves := make(chan core.KeyValueHolder, common.TrieLeavesChannelDefaultCapacity)
	err = userValidatorAccount.DataTrie().GetAllLeavesOnChannel(chLeaves, context.Background(), rootHash)
	if err != nil {
		return nil, err
	}
	for leaf := range chLeaves {
		validatorData := &systemSmartContracts.ValidatorDataV2{}
		value, errTrim := leaf.ValueWithoutSuffix(append(leaf.Key(), vm.ValidatorSCAddress...))
		if errTrim != nil {
			return nil, fmt.Errorf("%w for validator key %s", errTrim, hex.EncodeToString(leaf.Key()))
		}

		err = s.marshalizer.Unmarshal(validatorData, value)
		if err != nil {
			continue
		}
		for _, blsKey := range validatorData.BlsPubKeys {
			arguments = append(arguments, blsKey)
			arguments = append(arguments, leaf.Key())
		}
	}

	return arguments, nil
}

func (s *legacySystemSCProcessor) callSetOwnersOnAddresses(arguments [][]byte) error {
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr: vm.EndOfEpochAddress,
			CallValue:  big.NewInt(0),
			Arguments:  arguments,
		},
		RecipientAddr: vm.StakingSCAddress,
		Function:      "setOwnersOnAddresses",
	}

	vmOutput, errRun := s.systemVM.RunSmartContractCall(vmInput)
	if errRun != nil {
		return fmt.Errorf("%w when calling setOwnersOnAddresses function", errRun)
	}
	if vmOutput.ReturnCode != vmcommon.Ok {
		return fmt.Errorf("got return code %s when calling setOwnersOnAddresses", vmOutput.ReturnCode)
	}

	return s.processSCOutputAccounts(vmOutput)
}

func (s *legacySystemSCProcessor) initDelegationSystemSC() error {
	codeMetaData := &vmcommon.CodeMetadata{
		Upgradeable: false,
		Payable:     false,
		Readable:    true,
	}

	vmInput := &vmcommon.ContractCreateInput{
		VMInput: vmcommon.VMInput{
			CallerAddr: vm.DelegationManagerSCAddress,
			Arguments:  [][]byte{},
			CallValue:  big.NewInt(0),
		},
		ContractCode:         vm.DelegationManagerSCAddress,
		ContractCodeMetadata: codeMetaData.ToBytes(),
	}

	vmOutput, err := s.systemVM.RunSmartContractCreate(vmInput)
	if err != nil {
		return err
	}
	if vmOutput.ReturnCode != vmcommon.Ok {
		return epochStart.ErrCouldNotInitDelegationSystemSC
	}

	err = s.processSCOutputAccounts(vmOutput)
	if err != nil {
		return err
	}

	err = s.updateSystemSCContractsCode(vmInput.ContractCodeMetadata)
	if err != nil {
		return err
	}

	return nil
}

func (s *legacySystemSCProcessor) updateSystemSCContractsCode(contractMetadata []byte) error {
	contractsToUpdate := make([][]byte, 0)
	contractsToUpdate = append(contractsToUpdate, vm.StakingSCAddress)
	contractsToUpdate = append(contractsToUpdate, vm.ValidatorSCAddress)
	contractsToUpdate = append(contractsToUpdate, vm.GovernanceSCAddress)
	contractsToUpdate = append(contractsToUpdate, vm.ESDTSCAddress)
	contractsToUpdate = append(contractsToUpdate, vm.DelegationManagerSCAddress)
	contractsToUpdate = append(contractsToUpdate, vm.FirstDelegationSCAddress)

	for _, address := range contractsToUpdate {
		userAcc, err := s.getUserAccount(address)
		if err != nil {
			return err
		}

		userAcc.SetOwnerAddress(address)
		userAcc.SetCodeMetadata(contractMetadata)
		userAcc.SetCode(address)

		err = s.userAccountsDB.SaveAccount(userAcc)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *legacySystemSCProcessor) cleanAdditionalQueue() error {
	sw := core.NewStopWatch()
	sw.Start("systemSCProcessor")
	defer func() {
		sw.Stop("systemSCProcessor")
		log.Debug("systemSCProcessor.cleanAdditionalQueue time measurements", sw.GetMeasurements()...)
	}()

	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr: vm.EndOfEpochAddress,
			CallValue:  big.NewInt(0),
			Arguments:  [][]byte{},
		},
		RecipientAddr: vm.StakingSCAddress,
		Function:      "cleanAdditionalQueue",
	}
	vmOutput, errRun := s.systemVM.RunSmartContractCall(vmInput)
	if errRun != nil {
		return fmt.Errorf("%w when cleaning additional queue", errRun)
	}
	if vmOutput.ReturnCode != vmcommon.Ok {
		return fmt.Errorf("got return code %s, return message %s when cleaning additional queue", vmOutput.ReturnCode, vmOutput.ReturnMessage)
	}

	err := s.processSCOutputAccounts(vmOutput)
	if err != nil {
		return err
	}

	// returnData format is list(address - all blsKeys which were unstaked for that)
	addressLength := len(s.endOfEpochCallerAddress)
	mapOwnersKeys := make(map[string][][]byte)
	currentOwner := ""
	for _, returnData := range vmOutput.ReturnData {
		if len(returnData) == addressLength {
			currentOwner = string(returnData)
			continue
		}

		if len(currentOwner) != addressLength {
			continue
		}

		mapOwnersKeys[currentOwner] = append(mapOwnersKeys[currentOwner], returnData)
	}

	err = s.updateDelegationContracts(mapOwnersKeys)
	if err != nil {
		log.Error("update delegation contracts failed after cleaning additional queue", "error", err.Error())
		return err
	}

	return nil
}

func (s *legacySystemSCProcessor) stakeNodesFromQueue(
	validatorsInfoMap state.ShardValidatorsInfoMapHandler,
	nodesToStake uint32,
	nonce uint64,
	list common.PeerType,
) error {
	if nodesToStake == 0 {
		return nil
	}

	nodesToStakeAsBigInt := big.NewInt(0).SetUint64(uint64(nodesToStake))
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr: vm.EndOfEpochAddress,
			CallValue:  big.NewInt(0),
			Arguments:  [][]byte{nodesToStakeAsBigInt.Bytes()},
		},
		RecipientAddr: vm.StakingSCAddress,
		Function:      "stakeNodesFromQueue",
	}
	vmOutput, errRun := s.systemVM.RunSmartContractCall(vmInput)
	if errRun != nil {
		return fmt.Errorf("%w when staking nodes from waiting list", errRun)
	}
	if vmOutput.ReturnCode != vmcommon.Ok {
		return fmt.Errorf("got return code %s when staking nodes from waiting list", vmOutput.ReturnCode)
	}
	if len(vmOutput.ReturnData)%2 != 0 {
		return fmt.Errorf("%w return data must be divisible by 2 when staking nodes from waiting list", epochStart.ErrInvalidSystemSCReturn)
	}

	err := s.processSCOutputAccounts(vmOutput)
	if err != nil {
		return err
	}

	err = s.addNewlyStakedNodesToValidatorTrie(validatorsInfoMap, vmOutput.ReturnData, nonce, list)
	if err != nil {
		return err
	}

	return nil
}

func (s *legacySystemSCProcessor) addNewlyStakedNodesToValidatorTrie(
	validatorsInfoMap state.ShardValidatorsInfoMapHandler,
	returnData [][]byte,
	nonce uint64,
	list common.PeerType,
) error {
	for i := 0; i < len(returnData); i += 2 {
		blsKey := returnData[i]
		rewardAddress := returnData[i+1]

		peerAcc, err := s.getPeerAccount(blsKey)
		if err != nil {
			return err
		}

		err = peerAcc.SetRewardAddress(rewardAddress)
		if err != nil {
			return err
		}

		err = peerAcc.SetBLSPublicKey(blsKey)
		if err != nil {
			return err
		}

		peerAcc.SetListAndIndex(peerAcc.GetShardId(), string(list), uint32(nonce))
		peerAcc.SetTempRating(s.startRating)
		peerAcc.SetUnStakedEpoch(common.DefaultUnstakedEpoch)

		err = s.peerAccountsDB.SaveAccount(peerAcc)
		if err != nil {
			return err
		}

		validatorInfo := &state.ValidatorInfo{
			PublicKey:       blsKey,
			ShardId:         peerAcc.GetShardId(),
			List:            string(list),
			Index:           uint32(nonce),
			TempRating:      s.startRating,
			Rating:          s.startRating,
			RewardAddress:   rewardAddress,
			AccumulatedFees: big.NewInt(0),
		}
		err = validatorsInfoMap.Add(validatorInfo)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *legacySystemSCProcessor) initESDT() error {
	currentConfigValues, err := s.extractConfigFromESDTContract()
	if err != nil {
		return err
	}

	return s.changeESDTOwner(currentConfigValues)
}

func (s *legacySystemSCProcessor) extractConfigFromESDTContract() ([][]byte, error) {
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr:  s.endOfEpochCallerAddress,
			Arguments:   [][]byte{},
			CallValue:   big.NewInt(0),
			GasProvided: math.MaxUint64,
		},
		Function:      "getContractConfig",
		RecipientAddr: vm.ESDTSCAddress,
	}

	output, err := s.systemVM.RunSmartContractCall(vmInput)
	if err != nil {
		return nil, err
	}
	if len(output.ReturnData) != 4 {
		return nil, fmt.Errorf("%w getContractConfig should have returned 4 values", epochStart.ErrInvalidSystemSCReturn)
	}

	return output.ReturnData, nil
}

func (s *legacySystemSCProcessor) changeESDTOwner(currentConfigValues [][]byte) error {
	baseIssuingCost := currentConfigValues[1]
	minTokenNameLength := currentConfigValues[2]
	maxTokenNameLength := currentConfigValues[3]

	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr:  s.endOfEpochCallerAddress,
			Arguments:   [][]byte{s.esdtOwnerAddressBytes, baseIssuingCost, minTokenNameLength, maxTokenNameLength},
			CallValue:   big.NewInt(0),
			GasProvided: math.MaxUint64,
		},
		Function:      "configChange",
		RecipientAddr: vm.ESDTSCAddress,
	}

	output, err := s.systemVM.RunSmartContractCall(vmInput)
	if err != nil {
		return err
	}
	if output.ReturnCode != vmcommon.Ok {
		return fmt.Errorf("%w changeESDTOwner should have returned Ok", epochStart.ErrInvalidSystemSCReturn)
	}

	return s.processSCOutputAccounts(output)
}

func getRewardsMiniBlockForMeta(miniBlocks block.MiniBlockSlice) *block.MiniBlock {
	for _, miniBlock := range miniBlocks {
		if miniBlock.Type != block.RewardsBlock {
			continue
		}
		if miniBlock.ReceiverShardID != core.MetachainShardId {
			continue
		}
		return miniBlock
	}
	return nil
}

func (s *legacySystemSCProcessor) legacyEpochConfirmed(epoch uint32) {
	s.flagSwitchJailedWaiting.SetValue(epoch >= s.switchEnableEpoch && epoch <= s.stakingV4InitEnableEpoch)
	log.Debug("legacySystemSC: switch jail with waiting", "enabled", s.flagSwitchJailedWaiting.IsSet())

	// only toggle on exact epoch. In future epochs the config should have already been synchronized from peers
	s.flagHystNodesEnabled.SetValue(epoch == s.hystNodesEnableEpoch)

	s.flagChangeMaxNodesEnabled.SetValue(false)
	for _, maxNodesConfig := range s.maxNodesChangeConfigProvider.GetAllNodesConfig() {
		if epoch == maxNodesConfig.EpochEnable {
			s.flagChangeMaxNodesEnabled.SetValue(true)
		}
	}
	s.maxNodes = s.maxNodesChangeConfigProvider.GetCurrentNodesConfig().MaxNumNodes

	log.Debug("legacySystemSC: consider also (minimum) hysteresis nodes for minimum number of nodes",
		"enabled", epoch >= s.hystNodesEnableEpoch)

	// only toggle on exact epoch as init should be called only once
	s.flagDelegationEnabled.SetValue(epoch == s.delegationEnableEpoch)
	log.Debug("systemSCProcessor: delegation", "enabled", epoch >= s.delegationEnableEpoch)

	s.flagSetOwnerEnabled.SetValue(epoch == s.stakingV2EnableEpoch)
	s.flagStakingV2Enabled.SetValue(epoch >= s.stakingV2EnableEpoch && epoch <= s.stakingV4InitEnableEpoch)
	log.Debug("legacySystemSC: stakingV2", "enabled", epoch >= s.stakingV2EnableEpoch)
	log.Debug("legacySystemSC: change of maximum number of nodes and/or shuffling percentage",
		"enabled", s.flagChangeMaxNodesEnabled.IsSet(),
		"epoch", epoch,
		"maxNodes", s.maxNodes,
	)

	s.flagCorrectLastUnjailedEnabled.SetValue(epoch == s.correctLastUnJailEpoch)
	log.Debug("legacySystemSC: correct last unjailed", "enabled", s.flagCorrectLastUnjailedEnabled.IsSet())

	s.flagCorrectNumNodesToStake.SetValue(epoch >= s.correctLastUnJailEpoch && epoch <= s.stakingV4InitEnableEpoch)
	log.Debug("legacySystemSC: correct last unjailed", "enabled", s.flagCorrectNumNodesToStake.IsSet())

	s.flagESDTEnabled.SetValue(epoch == s.esdtEnableEpoch)
	log.Debug("legacySystemSC: ESDT initialization", "enabled", s.flagESDTEnabled.IsSet())

	s.flagSaveJailedAlwaysEnabled.SetValue(epoch >= s.saveJailedAlwaysEnableEpoch)
	log.Debug("legacySystemSC: save jailed always", "enabled", s.flagSaveJailedAlwaysEnabled.IsSet())

	s.flagStakingQueueEnabled.SetValue(epoch < s.stakingV4InitEnableEpoch)
	log.Debug("legacySystemSC: staking queue on meta", "enabled", s.flagStakingQueueEnabled.IsSet())
}
