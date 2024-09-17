package bootstrap

import (
	"context"
	"fmt"
	"testing"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/data"
	"github.com/multiversx/mx-chain-core-go/data/block"
	"github.com/multiversx/mx-chain-go/process/mock"
	"github.com/multiversx/mx-chain-go/sharding/nodesCoordinator"
	"github.com/multiversx/mx-chain-go/storage"
	"github.com/multiversx/mx-chain-go/testscommon"
	epochStartMocks "github.com/multiversx/mx-chain-go/testscommon/bootstrapMocks/epochStart"
	dataRetrieverMock "github.com/multiversx/mx-chain-go/testscommon/dataRetriever"
	"github.com/multiversx/mx-chain-go/testscommon/shardingMocks"
	"github.com/stretchr/testify/require"
)

func createSovBootStrapProc() *sovereignBootStrapShardProcessor {
	args := createMockEpochStartBootstrapArgs(createComponentsForEpochStart())
	args.RunTypeComponents = mock.NewSovereignRunTypeComponentsStub()
	epochStartProvider, _ := NewEpochStartBootstrap(args)
	return &sovereignBootStrapShardProcessor{
		&sovereignChainEpochStartBootstrap{
			epochStartProvider,
		},
	}
}

func TestBootStrapSovereignShardProcessor_requestAndProcessForShard(t *testing.T) {
	t.Parallel()

	prevShardHeaderHash := []byte("prevShardHeaderHash")
	notarizedMetaHeaderHash := []byte("notarizedMetaHeaderHash")
	prevMetaHeaderHash := []byte("prevMetaHeaderHash")

	coreComp, cryptoComp := createComponentsForEpochStart()
	args := createMockEpochStartBootstrapArgs(coreComp, cryptoComp)
	args.RunTypeComponents = mock.NewSovereignRunTypeComponentsStub()

	prevShardHeader := &block.Header{}
	notarizedMetaHeader := &block.SovereignChainHeader{
		Header: &block.Header{
			PrevHash: prevMetaHeaderHash,
		},
	}
	metaBlockInstance := &block.SovereignChainHeader{
		Header: &block.Header{},
	}
	prevMetaBlock := &block.SovereignChainHeader{
		Header: &block.Header{},
	}

	epochStartProvider, _ := NewEpochStartBootstrap(args)
	epochStartProvider.syncedHeaders = make(map[string]data.HeaderHandler)
	epochStartProvider.epochStartMeta = metaBlockInstance
	epochStartProvider.prevEpochStartMeta = prevMetaBlock
	epochStartProvider.headersSyncer = &epochStartMocks.HeadersByHashSyncerStub{
		GetHeadersCalled: func() (m map[string]data.HeaderHandler, err error) {
			return map[string]data.HeaderHandler{
				string(notarizedMetaHeaderHash): notarizedMetaHeader,
				string(prevShardHeaderHash):     prevShardHeader,
			}, nil
		},
	}
	epochStartProvider.dataPool = &dataRetrieverMock.PoolsHolderStub{
		TrieNodesCalled: func() storage.Cacher {
			return &testscommon.CacherStub{
				GetCalled: func(key []byte) (value interface{}, ok bool) {
					return nil, true
				},
			}
		},
	}

	epochStartProvider.miniBlocksSyncer = &epochStartMocks.PendingMiniBlockSyncHandlerStub{}
	epochStartProvider.requestHandler = &testscommon.RequestHandlerStub{}
	epochStartProvider.nodesConfig = &nodesCoordinator.NodesCoordinatorRegistry{}
	sovProc := &sovereignBootStrapShardProcessor{
		&sovereignChainEpochStartBootstrap{
			epochStartProvider,
		},
	}

	err := sovProc.requestAndProcessForShard(make([]*block.MiniBlock, 0))
	require.Nil(t, err)

	// take this and use to test sov bootstrapper
	/*
		//epochStartBlock_0
		bootStorer, err := sovProc.storageService.GetStorer(dataRetriever.BootstrapUnit)
		require.Nil(t, err)

		roundToUseAsKey := int64(metaBlockInstance.GetRound())
		key := []byte(strconv.FormatInt(roundToUseAsKey, 10))
		bootStrapDataBytes, err := bootStorer.Get(key)
		require.Nil(t, err)

		bootStrapData := &bootstrapStorage.BootstrapData{}

		err = sovProc.coreComponentsHolder.InternalMarshalizer().Unmarshal(bootStrapData, bootStrapDataBytes)
		require.Nil(t, err)
	*/
}

func TestBootStrapSovereignShardProcessor_computeNumShards(t *testing.T) {
	t.Parallel()

	sovProc := createSovBootStrapProc()
	require.Equal(t, uint32(0x1), sovProc.computeNumShards(nil))
}

func TestBootStrapSovereignShardProcessor_createRequestHandler(t *testing.T) {
	t.Parallel()

	sovProc := createSovBootStrapProc()
	reqHandler, err := sovProc.createRequestHandler()
	require.Nil(t, err)
	require.Equal(t, "*requestHandlers.sovereignResolverRequestHandler", fmt.Sprintf("%T", reqHandler))
}

func TestBootStrapSovereignShardProcessor_createResolversContainer(t *testing.T) {
	t.Parallel()

	sovProc := createSovBootStrapProc()
	require.Nil(t, sovProc.createResolversContainer())
}

func TestBootStrapSovereignShardProcessor_syncHeadersFrom(t *testing.T) {
	t.Parallel()

	sovProc := createSovBootStrapProc()

	prevEpochStartHash := []byte("prevEpochStartHash")
	sovHdr := &block.SovereignChainHeader{
		Header: &block.Header{
			Epoch: 4,
		},
		EpochStart: block.EpochStartSovereign{
			Economics: block.Economics{
				PrevEpochStartHash: prevEpochStartHash,
			},
		},
	}

	syncedHeaders := map[string]data.HeaderHandler{
		"hash": &block.SovereignChainHeader{},
	}
	headersSyncedCt := 0
	sovProc.headersSyncer = &epochStartMocks.HeadersByHashSyncerStub{
		SyncMissingHeadersByHashCalled: func(shardIDs []uint32, headersHashes [][]byte, ctx context.Context) error {
			require.Equal(t, []uint32{core.SovereignChainShardId}, shardIDs)
			require.Equal(t, [][]byte{prevEpochStartHash}, headersHashes)
			headersSyncedCt++
			return nil
		},
		GetHeadersCalled: func() (map[string]data.HeaderHandler, error) {
			return syncedHeaders, nil
		},
	}

	res, err := sovProc.syncHeadersFrom(sovHdr)
	require.Nil(t, err)
	require.Equal(t, res, syncedHeaders)
	require.Equal(t, 1, headersSyncedCt)

	res, err = sovProc.syncHeadersFromStorage(sovHdr, 0, 0, DefaultTimeToWaitForRequestedData)
	require.Nil(t, err)
	require.Equal(t, res, syncedHeaders)
	require.Equal(t, 2, headersSyncedCt)
}

func TestBootStrapSovereignShardProcessor_processNodesConfigFromStorage(t *testing.T) {
	t.Parallel()

	sovBlock := &block.SovereignChainHeader{
		Header: &block.Header{
			Epoch: 0,
		},
	}

	pksBytes := createPkBytes(1)
	delete(pksBytes, core.MetachainShardId)

	expectedNodesConfig := &nodesCoordinator.NodesCoordinatorRegistry{
		EpochsConfig: map[string]*nodesCoordinator.EpochValidators{
			"0": {
				EligibleValidators: map[string][]*nodesCoordinator.SerializableValidator{
					"0": {
						&nodesCoordinator.SerializableValidator{
							PubKey:  pksBytes[0],
							Chances: 1,
							Index:   0,
						},
					},
				},
				WaitingValidators: map[string][]*nodesCoordinator.SerializableValidator{},
				LeavingValidators: map[string][]*nodesCoordinator.SerializableValidator{},
			},
		},
		CurrentEpoch: 0,
	}

	sovProc := createSovBootStrapProc()
	sovProc.runTypeComponents = &mock.RunTypeComponentsStub{
		NodesCoordinatorWithRaterFactory: &testscommon.NodesCoordinatorFactoryMock{
			CreateNodesCoordinatorWithRaterCalled: func(args *nodesCoordinator.NodesCoordinatorWithRaterArgs) (nodesCoordinator.NodesCoordinator, error) {
				return &shardingMocks.NodesCoordinatorMock{
					NodesCoordinatorToRegistryCalled: func(epoch uint32) nodesCoordinator.NodesCoordinatorRegistryHandler {
						return expectedNodesConfig
					},
				}, nil
			},
		},
	}

	sovProc.dataPool = dataRetrieverMock.NewPoolsHolderMock()
	sovProc.requestHandler = &testscommon.RequestHandlerStub{}
	sovProc.epochStartMeta = sovBlock
	sovProc.prevEpochStartMeta = sovBlock

	nodesConfig, shardId, err := sovProc.processNodesConfigFromStorage([]byte("pubkey"), 0)
	require.Nil(t, err)
	require.Equal(t, core.SovereignChainShardId, shardId)
	require.Equal(t, nodesConfig, expectedNodesConfig)
}
