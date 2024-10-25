package storageBootstrap

import (
	"testing"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/data"
	"github.com/multiversx/mx-chain-core-go/data/block"
	"github.com/multiversx/mx-chain-go/dataRetriever"
	"github.com/multiversx/mx-chain-go/process"
	"github.com/multiversx/mx-chain-go/process/block/bootstrapStorage"
	"github.com/multiversx/mx-chain-go/storage"
	"github.com/multiversx/mx-chain-go/testscommon"
	storageStubs "github.com/multiversx/mx-chain-go/testscommon/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSovereignChainShardStorageBootstrapper(t *testing.T) {
	t.Parallel()

	baseArgs := createMockShardStorageBoostrapperArgs()
	args := ArgsShardStorageBootstrapper{
		ArgsBaseStorageBootstrapper: baseArgs,
	}

	t.Run("should error when shard storage bootstrapper is nil", func(t *testing.T) {
		t.Parallel()

		scesb, err := NewSovereignChainShardStorageBootstrapper(nil)

		assert.Nil(t, scesb)
		assert.Equal(t, process.ErrNilShardStorageBootstrapper, err)
	})

	t.Run("should work", func(t *testing.T) {
		t.Parallel()

		ssb, _ := NewShardStorageBootstrapper(args)
		scssb, err := NewSovereignChainShardStorageBootstrapper(ssb)

		assert.NotNil(t, scssb)
		assert.Nil(t, err)
	})
}

func TestSovereignShardBootstrapFactory_applyCrossNotarizedHeaders(t *testing.T) {
	t.Parallel()

	baseArgs := createMockShardStorageBoostrapperArgs()

	extendedHdrhash := []byte("hash")
	extendedHdr := &block.ShardHeaderExtended{
		Header: &block.HeaderV2{
			Header: &block.Header{
				Nonce: 4,
			},
		},
	}

	wasCrossChainHdrNotarized := false
	wasCrossChainHdrTracked := false
	baseArgs.BlockTracker = &testscommon.BlockTrackerStub{
		AddCrossNotarizedHeaderCalled: func(shardID uint32, crossNotarizedHeader data.HeaderHandler, crossNotarizedHeaderHash []byte) {
			require.Equal(t, core.MainChainShardId, shardID)
			require.Equal(t, extendedHdr, crossNotarizedHeader)
			require.Equal(t, extendedHdrhash, crossNotarizedHeaderHash)
			wasCrossChainHdrNotarized = true
		},
		AddTrackedHeaderCalled: func(header data.HeaderHandler, hash []byte) {
			require.Equal(t, extendedHdr, header)
			require.Equal(t, extendedHdrhash, hash)
			wasCrossChainHdrTracked = true
		},
	}

	storer := &storageStubs.StorerStub{
		GetCalled: func(key []byte) ([]byte, error) {
			return baseArgs.Marshalizer.Marshal(extendedHdr)
		},
	}

	baseArgs.Store = &storageStubs.ChainStorerStub{
		GetStorerCalled: func(unitType dataRetriever.UnitType) (storage.Storer, error) {
			switch unitType {
			case dataRetriever.ExtendedShardHeadersUnit:
				return storer, nil

			}
			return &storageStubs.StorerStub{}, nil
		},
	}
	args := ArgsShardStorageBootstrapper{
		ArgsBaseStorageBootstrapper: baseArgs,
	}
	ssb, _ := NewShardStorageBootstrapper(args)
	scssb, _ := NewSovereignChainShardStorageBootstrapper(ssb)

	crossNotarizedHeaders := []bootstrapStorage.BootstrapHeaderInfo{
		{
			Hash:    extendedHdrhash,
			Nonce:   4,
			ShardId: core.MainChainShardId,
		},
	}
	err := scssb.applyCrossNotarizedHeaders(crossNotarizedHeaders)
	require.Nil(t, err)
	require.True(t, wasCrossChainHdrNotarized)
	require.True(t, wasCrossChainHdrTracked)
}

func TestSovereignShardBootstrapFactory_cleanupNotarizedStorageForHigherNoncesIfExist(t *testing.T) {
	extendedHeader := &block.ShardHeaderExtended{
		Header: &block.HeaderV2{
			Header: &block.Header{},
		},
	}
	testCleanupNotarizedStorageForHigherNoncesIfExist(t, core.MainChainShardId, extendedHeader, NewSovereignShardStorageBootstrapperFactory())
}
