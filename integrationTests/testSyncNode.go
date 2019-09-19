package integrationTests

import (
	"context"
	"fmt"

	"github.com/ElrondNetwork/elrond-go/consensus/spos/sposFactory"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/integrationTests/mock"
	"github.com/ElrondNetwork/elrond-go/node"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/block"
	"github.com/ElrondNetwork/elrond-go/process/smartContract"
	"github.com/ElrondNetwork/elrond-go/process/sync"
	"github.com/ElrondNetwork/elrond-go/sharding"
)

// NewTestSyncNode returns a new TestProcessorNode instance with sync capabilities
func NewTestSyncNode(
	maxShards uint32,
	nodeShardId uint32,
	txSignPrivKeyShardId uint32,
	initialNodeAddr string,
) *TestProcessorNode {

	shardCoordinator, _ := sharding.NewMultiShardCoordinator(maxShards, nodeShardId)

	messenger := CreateMessengerWithKadDht(context.Background(), initialNodeAddr)
	tpn := &TestProcessorNode{
		ShardCoordinator: shardCoordinator,
		Messenger:        messenger,
	}

	tpn.OwnAccount = CreateTestWalletAccount(shardCoordinator, txSignPrivKeyShardId)
	tpn.initDataPools()
	tpn.initTestNodeWithSync()

	return tpn
}

func (tpn *TestProcessorNode) initTestNodeWithSync() {
	tpn.initRounder()
	tpn.initStorage()
	tpn.AccntState, _, _ = CreateAccountsDB(tpn.ShardCoordinator)
	tpn.initChainHandler()
	tpn.GenesisBlocks = CreateGenesisBlocks(tpn.ShardCoordinator)
	tpn.initInterceptors()
	tpn.initResolvers()
	tpn.initInnerProcessors()
	tpn.initBlockProcessorWithSync()
	tpn.BroadcastMessenger, _ = sposFactory.GetBroadcastMessenger(
		TestMarshalizer,
		tpn.Messenger,
		tpn.ShardCoordinator,
		tpn.OwnAccount.SkTxSign,
		tpn.OwnAccount.SingleSigner,
	)
	tpn.initBootstrapper()
	tpn.setGenesisBlock()
	tpn.initNode()
	tpn.ScDataGetter, _ = smartContract.NewSCDataGetter(tpn.VmDataGetter)
	tpn.addHandlersForCounters()
}

func (tpn *TestProcessorNode) initBlockProcessorWithSync() {
	var err error

	tpn.ForkDetector, _ = sync.NewBasicForkDetector(tpn.Rounder)

	tpn.BlockTracker = &mock.BlocksTrackerMock{
		AddBlockCalled: func(headerHandler data.HeaderHandler) {
		},
		RemoveNotarisedBlocksCalled: func(headerHandler data.HeaderHandler) error {
			return nil
		},
		UnnotarisedBlocksCalled: func() []data.HeaderHandler {
			return make([]data.HeaderHandler, 0)
		},
	}

	if tpn.ShardCoordinator.SelfId() == sharding.MetachainShardId {
		tpn.BlockProcessor, err = block.NewMetaProcessor(
			&mock.ServiceContainerMock{},
			tpn.AccntState,
			tpn.MetaDataPool,
			tpn.ForkDetector,
			tpn.ShardCoordinator,
			TestHasher,
			TestMarshalizer,
			tpn.Storage,
			tpn.GenesisBlocks,
			tpn.RequestHandler,
			TestUint64Converter,
		)
	} else {
		arguments := block.ArgShardProcessor{
			ArgBaseProcessor: &block.ArgBaseProcessor{
				Accounts:         tpn.AccntState,
				ForkDetector:     tpn.ForkDetector,
				Hasher:           TestHasher,
				Marshalizer:      TestMarshalizer,
				Store:            tpn.Storage,
				ShardCoordinator: tpn.ShardCoordinator,
				Uint64Converter:  TestUint64Converter,
				StartHeaders:     tpn.GenesisBlocks,
				RequestHandler:   tpn.RequestHandler,
				Core:             nil,
			},
			DataPool:        tpn.ShardDataPool,
			BlocksTracker:   tpn.BlockTracker,
			TxCoordinator:   tpn.TxCoordinator,
			TxsPoolsCleaner: &mock.TxPoolsCleanerMock{},
		}

		tpn.BlockProcessor, err = block.NewShardProcessor(arguments)
	}

	if err != nil {
		fmt.Printf("Error creating blockprocessor: %s\n", err.Error())
	}
}

func (tpn *TestProcessorNode) createShardBootstrapper() (process.Bootstrapper, error) {
	bootstrap, err := sync.NewShardBootstrap(
		tpn.ShardDataPool,
		tpn.Storage,
		tpn.BlockChain,
		tpn.Rounder,
		tpn.BlockProcessor,
		node.WaitTime,
		TestHasher,
		TestMarshalizer,
		tpn.ForkDetector,
		tpn.ResolverFinder,
		tpn.ShardCoordinator,
		tpn.AccntState,
		1,
	)
	if err != nil {
		return nil, err
	}

	return bootstrap, nil
}

func (tpn *TestProcessorNode) createMetaChainBootstrapper() (process.Bootstrapper, error) {
	bootstrap, err := sync.NewMetaBootstrap(
		tpn.MetaDataPool,
		tpn.Storage,
		tpn.BlockChain,
		tpn.Rounder,
		tpn.BlockProcessor,
		node.WaitTime,
		TestHasher,
		TestMarshalizer,
		tpn.ForkDetector,
		tpn.ResolverFinder,
		tpn.ShardCoordinator,
		tpn.AccntState,
		1,
	)

	if err != nil {
		return nil, err
	}

	return bootstrap, nil
}

func (tpn *TestProcessorNode) initBootstrapper() {
	if tpn.ShardCoordinator.SelfId() < tpn.ShardCoordinator.NumberOfShards() {
		tpn.Bootstrapper, _ = tpn.createShardBootstrapper()
	} else {
		tpn.Bootstrapper, _ = tpn.createMetaChainBootstrapper()
	}
}
