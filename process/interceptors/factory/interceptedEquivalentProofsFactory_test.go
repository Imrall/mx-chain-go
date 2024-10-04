package factory

import (
	"testing"

	"github.com/multiversx/mx-chain-core-go/data"
	"github.com/multiversx/mx-chain-core-go/data/block"
	"github.com/multiversx/mx-chain-go/consensus/mock"
	processMock "github.com/multiversx/mx-chain-go/process/mock"
	"github.com/multiversx/mx-chain-go/testscommon/consensus"
	"github.com/stretchr/testify/require"
)

func createMockArgInterceptedDataFactory() ArgInterceptedDataFactory {
	return ArgInterceptedDataFactory{
		CoreComponents: &processMock.CoreComponentsMock{
			IntMarsh: &mock.MarshalizerMock{},
		},
		ShardCoordinator:  &mock.ShardCoordinatorMock{},
		HeaderSigVerifier: &consensus.HeaderSigVerifierMock{},
	}
}

func TestInterceptedEquivalentProofsFactory_IsInterfaceNil(t *testing.T) {
	t.Parallel()

	var factory *interceptedEquivalentProofsFactory
	require.True(t, factory.IsInterfaceNil())

	factory = NewInterceptedEquivalentProofsFactory(createMockArgInterceptedDataFactory())
	require.False(t, factory.IsInterfaceNil())
}

func TestNewInterceptedEquivalentProofsFactory(t *testing.T) {
	t.Parallel()

	factory := NewInterceptedEquivalentProofsFactory(createMockArgInterceptedDataFactory())
	require.NotNil(t, factory)
}

func TestInterceptedEquivalentProofsFactory_Create(t *testing.T) {
	t.Parallel()

	args := createMockArgInterceptedDataFactory()
	factory := NewInterceptedEquivalentProofsFactory(args)
	require.NotNil(t, factory)

	providedProof := &block.HeaderProof{
		PubKeysBitmap:       []byte("bitmap"),
		AggregatedSignature: []byte("sig"),
		HeaderHash:          []byte("hash"),
		HeaderEpoch:         123,
		HeaderNonce:         345,
		HeaderShardId:       0,
	}
	providedDataBuff, _ := args.CoreComponents.InternalMarshalizer().Marshal(providedProof)
	interceptedData, err := factory.Create(providedDataBuff)
	require.NoError(t, err)
	require.NotNil(t, interceptedData)

	type interceptedEquivalentProof interface {
		GetProof() data.HeaderProofHandler
	}
	interceptedHeaderProof, ok := interceptedData.(interceptedEquivalentProof)
	require.True(t, ok)

	proof := interceptedHeaderProof.GetProof()
	require.NotNil(t, proof)
	require.Equal(t, providedProof.GetPubKeysBitmap(), proof.GetPubKeysBitmap())
	require.Equal(t, providedProof.GetAggregatedSignature(), proof.GetAggregatedSignature())
	require.Equal(t, providedProof.GetHeaderHash(), proof.GetHeaderHash())
	require.Equal(t, providedProof.GetHeaderEpoch(), proof.GetHeaderEpoch())
	require.Equal(t, providedProof.GetHeaderNonce(), proof.GetHeaderNonce())
	require.Equal(t, providedProof.GetHeaderShardId(), proof.GetHeaderShardId())
}