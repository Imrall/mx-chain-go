package staking

import (
	"time"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/nodetype"
	"github.com/multiversx/mx-chain-core-go/data"
	"github.com/multiversx/mx-chain-core-go/data/block"
	"github.com/multiversx/mx-chain-core-go/data/endProcess"
	"github.com/multiversx/mx-chain-core-go/data/typeConverters/uint64ByteSlice"
	"github.com/multiversx/mx-chain-core-go/hashing/sha256"
	"github.com/multiversx/mx-chain-core-go/marshal"
	"github.com/multiversx/mx-chain-go/common"
	"github.com/multiversx/mx-chain-go/common/enablers"
	"github.com/multiversx/mx-chain-go/common/forking"
	"github.com/multiversx/mx-chain-go/config"
	"github.com/multiversx/mx-chain-go/dataRetriever"
	"github.com/multiversx/mx-chain-go/dataRetriever/blockchain"
	"github.com/multiversx/mx-chain-go/epochStart/notifier"
	"github.com/multiversx/mx-chain-go/factory"
	"github.com/multiversx/mx-chain-go/integrationTests"
	integrationMocks "github.com/multiversx/mx-chain-go/integrationTests/mock"
	mockFactory "github.com/multiversx/mx-chain-go/node/mock/factory"
	"github.com/multiversx/mx-chain-go/process/mock"
	"github.com/multiversx/mx-chain-go/sharding"
	"github.com/multiversx/mx-chain-go/sharding/nodesCoordinator"
	"github.com/multiversx/mx-chain-go/state"
	stateFactory "github.com/multiversx/mx-chain-go/state/factory"
	"github.com/multiversx/mx-chain-go/state/storagePruningManager"
	"github.com/multiversx/mx-chain-go/state/storagePruningManager/evictionWaitingList"
	"github.com/multiversx/mx-chain-go/statusHandler"
	"github.com/multiversx/mx-chain-go/testscommon"
	dataRetrieverMock "github.com/multiversx/mx-chain-go/testscommon/dataRetriever"
	"github.com/multiversx/mx-chain-go/testscommon/mainFactoryMocks"
	"github.com/multiversx/mx-chain-go/testscommon/outport"
	"github.com/multiversx/mx-chain-go/testscommon/stakingcommon"
	statusHandlerMock "github.com/multiversx/mx-chain-go/testscommon/statusHandler"
	"github.com/multiversx/mx-chain-go/trie"
	"github.com/multiversx/mx-chain-go/trie/hashesHolder"
)

const hashSize = 32

func createComponentHolders(numOfShards uint32) (
	factory.CoreComponentsHolder,
	factory.DataComponentsHolder,
	factory.BootstrapComponentsHolder,
	factory.StatusComponentsHolder,
	factory.StateComponentsHandler,
) {
	coreComponents := createCoreComponents()
	statusComponents := createStatusComponents()
	stateComponents := createStateComponents(coreComponents)
	dataComponents := createDataComponents(coreComponents, numOfShards)
	bootstrapComponents := createBootstrapComponents(coreComponents.InternalMarshalizer(), numOfShards)

	return coreComponents, dataComponents, bootstrapComponents, statusComponents, stateComponents
}

func createCoreComponents() factory.CoreComponentsHolder {
	epochNotifier := forking.NewGenericEpochNotifier()
	configEnableEpochs := config.EnableEpochs{
		StakingV4Step2EnableEpoch:          stakingV4Step2EnableEpoch,
		StakingV4Step1EnableEpoch:          stakingV4InitEpoch,
		StakingV4Step3EnableEpoch:          stakingV4Step3EnableEpoch,
		RefactorPeersMiniBlocksEnableEpoch: integrationTests.UnreachableEpoch,
	}

	enableEpochsHandler, _ := enablers.NewEnableEpochsHandler(configEnableEpochs, epochNotifier)

	return &integrationMocks.CoreComponentsStub{
		InternalMarshalizerField:           &marshal.GogoProtoMarshalizer{},
		HasherField:                        sha256.NewSha256(),
		Uint64ByteSliceConverterField:      uint64ByteSlice.NewBigEndianConverter(),
		StatusHandlerField:                 statusHandler.NewStatusMetrics(),
		RoundHandlerField:                  &mock.RoundHandlerMock{RoundTimeDuration: time.Second},
		EpochStartNotifierWithConfirmField: notifier.NewEpochStartSubscriptionHandler(),
		EpochNotifierField:                 epochNotifier,
		RaterField:                         &testscommon.RaterMock{Chance: 5},
		AddressPubKeyConverterField:        testscommon.NewPubkeyConverterMock(addressLength),
		EconomicsDataField:                 stakingcommon.CreateEconomicsData(),
		ChanStopNodeProcessField:           endProcess.GetDummyEndProcessChannel(),
		NodeTypeProviderField:              nodetype.NewNodeTypeProvider(core.NodeTypeValidator),
		ProcessStatusHandlerInternal:       statusHandler.NewProcessStatusHandler(),
		EnableEpochsHandlerField:           enableEpochsHandler,
		EnableRoundsHandlerField:           &testscommon.EnableRoundsHandlerStub{},
	}
}

func createDataComponents(coreComponents factory.CoreComponentsHolder, numOfShards uint32) factory.DataComponentsHolder {
	genesisBlock := createGenesisMetaBlock()
	genesisBlockHash, _ := coreComponents.InternalMarshalizer().Marshal(genesisBlock)
	genesisBlockHash = coreComponents.Hasher().Compute(string(genesisBlockHash))

	blockChain, _ := blockchain.NewMetaChain(&statusHandlerMock.AppStatusHandlerStub{})
	_ = blockChain.SetGenesisHeader(createGenesisMetaBlock())
	blockChain.SetGenesisHeaderHash(genesisBlockHash)

	chainStorer := dataRetriever.NewChainStorer()
	chainStorer.AddStorer(dataRetriever.BootstrapUnit, integrationTests.CreateMemUnit())
	chainStorer.AddStorer(dataRetriever.MetaHdrNonceHashDataUnit, integrationTests.CreateMemUnit())
	chainStorer.AddStorer(dataRetriever.MetaBlockUnit, integrationTests.CreateMemUnit())
	chainStorer.AddStorer(dataRetriever.MiniBlockUnit, integrationTests.CreateMemUnit())
	chainStorer.AddStorer(dataRetriever.BlockHeaderUnit, integrationTests.CreateMemUnit())
	for i := uint32(0); i < numOfShards; i++ {
		unit := dataRetriever.ShardHdrNonceHashDataUnit + dataRetriever.UnitType(i)
		chainStorer.AddStorer(unit, integrationTests.CreateMemUnit())
	}

	return &mockFactory.DataComponentsMock{
		Store:         chainStorer,
		DataPool:      dataRetrieverMock.NewPoolsHolderMock(),
		BlockChain:    blockChain,
		EconomicsData: coreComponents.EconomicsData(),
	}
}

func createBootstrapComponents(
	marshaller marshal.Marshalizer,
	numOfShards uint32,
) factory.BootstrapComponentsHolder {
	shardCoordinator, _ := sharding.NewMultiShardCoordinator(numOfShards, core.MetachainShardId)
	ncr, _ := nodesCoordinator.NewNodesCoordinatorRegistryFactory(
		marshaller,
		stakingV4Step2EnableEpoch,
	)

	return &mainFactoryMocks.BootstrapComponentsStub{
		ShCoordinator:        shardCoordinator,
		HdrIntegrityVerifier: &mock.HeaderIntegrityVerifierStub{},
		VersionedHdrFactory: &testscommon.VersionedHeaderFactoryStub{
			CreateCalled: func(epoch uint32) data.HeaderHandler {
				return &block.MetaBlock{Epoch: epoch}
			},
		},
		NodesCoordinatorRegistryFactoryField: ncr,
	}
}

func createStatusComponents() factory.StatusComponentsHolder {
	return &integrationMocks.StatusComponentsStub{
		Outport:          &outport.OutportStub{},
		AppStatusHandler: &statusHandlerMock.AppStatusHandlerStub{},
	}
}

func createStateComponents(coreComponents factory.CoreComponentsHolder) factory.StateComponentsHandler {
	tsmArgs := getNewTrieStorageManagerArgs(coreComponents)
	tsm, _ := trie.CreateTrieStorageManager(tsmArgs, trie.StorageManagerOptions{})
	trieFactoryManager, _ := trie.NewTrieStorageManagerWithoutPruning(tsm)
	userAccountsDB := createAccountsDB(coreComponents, stateFactory.NewAccountCreator(), trieFactoryManager)
	peerAccountsDB := createAccountsDB(coreComponents, stateFactory.NewPeerAccountCreator(), trieFactoryManager)

	_ = userAccountsDB.SetSyncer(&mock.AccountsDBSyncerStub{})
	_ = peerAccountsDB.SetSyncer(&mock.AccountsDBSyncerStub{})

	return &testscommon.StateComponentsMock{
		PeersAcc: peerAccountsDB,
		Accounts: userAccountsDB,
	}
}

func getNewTrieStorageManagerArgs(coreComponents factory.CoreComponentsHolder) trie.NewTrieStorageManagerArgs {
	return trie.NewTrieStorageManagerArgs{
		MainStorer:             testscommon.CreateMemUnit(),
		CheckpointsStorer:      testscommon.CreateMemUnit(),
		Marshalizer:            coreComponents.InternalMarshalizer(),
		Hasher:                 coreComponents.Hasher(),
		GeneralConfig:          config.TrieStorageManagerConfig{SnapshotsGoroutineNum: 1},
		CheckpointHashesHolder: hashesHolder.NewCheckpointHashesHolder(10, hashSize),
		IdleProvider:           &testscommon.ProcessStatusHandlerStub{},
	}
}

func createAccountsDB(
	coreComponents factory.CoreComponentsHolder,
	accountFactory state.AccountFactory,
	trieStorageManager common.StorageManager,
) *state.AccountsDB {
	tr, _ := trie.NewTrie(trieStorageManager, coreComponents.InternalMarshalizer(), coreComponents.Hasher(), 5)

	argsEvictionWaitingList := evictionWaitingList.MemoryEvictionWaitingListArgs{
		RootHashesSize: 10,
		HashesSize:     hashSize,
	}
	ewl, _ := evictionWaitingList.NewMemoryEvictionWaitingList(argsEvictionWaitingList)
	spm, _ := storagePruningManager.NewStoragePruningManager(ewl, 10)
	argsAccountsDb := state.ArgsAccountsDB{
		Trie:                  tr,
		Hasher:                coreComponents.Hasher(),
		Marshaller:            coreComponents.InternalMarshalizer(),
		AccountFactory:        accountFactory,
		StoragePruningManager: spm,
		ProcessingMode:        common.Normal,
		ProcessStatusHandler:  coreComponents.ProcessStatusHandler(),
		AppStatusHandler:      &statusHandlerMock.AppStatusHandlerStub{},
		AddressConverter:      coreComponents.AddressPubKeyConverter(),
	}
	adb, _ := state.NewAccountsDB(argsAccountsDb)
	return adb
}
