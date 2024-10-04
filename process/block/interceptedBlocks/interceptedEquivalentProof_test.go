package interceptedBlocks

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/data/block"
	"github.com/multiversx/mx-chain-go/consensus/mock"
	"github.com/multiversx/mx-chain-go/process"
	"github.com/multiversx/mx-chain-go/testscommon/consensus"
	"github.com/multiversx/mx-chain-go/testscommon/marshallerMock"
	logger "github.com/multiversx/mx-chain-logger-go"
	"github.com/stretchr/testify/require"
)

var (
	expectedErr    = errors.New("expected error")
	testMarshaller = &marshallerMock.MarshalizerMock{}
)

func createMockDataBuff() []byte {
	proof := &block.HeaderProof{
		PubKeysBitmap:       []byte("bitmap"),
		AggregatedSignature: []byte("sig"),
		HeaderHash:          []byte("hash"),
		HeaderEpoch:         123,
		HeaderNonce:         345,
		HeaderShardId:       0,
	}

	dataBuff, _ := testMarshaller.Marshal(proof)
	return dataBuff
}

func createMockArgInterceptedEquivalentProof() ArgInterceptedEquivalentProof {
	return ArgInterceptedEquivalentProof{
		DataBuff:          createMockDataBuff(),
		Marshaller:        testMarshaller,
		ShardCoordinator:  &mock.ShardCoordinatorMock{},
		HeaderSigVerifier: &consensus.HeaderSigVerifierMock{},
	}
}

func TestInterceptedEquivalentProof_IsInterfaceNil(t *testing.T) {
	t.Parallel()

	var iep *interceptedEquivalentProof
	require.True(t, iep.IsInterfaceNil())

	iep, _ = NewInterceptedEquivalentProof(createMockArgInterceptedEquivalentProof())
	require.False(t, iep.IsInterfaceNil())
}

func TestNewInterceptedEquivalentProof(t *testing.T) {
	t.Parallel()

	t.Run("nil DataBuff should error", func(t *testing.T) {
		t.Parallel()

		args := createMockArgInterceptedEquivalentProof()
		args.DataBuff = nil
		iep, err := NewInterceptedEquivalentProof(args)
		require.Equal(t, process.ErrNilBuffer, err)
		require.Nil(t, iep)
	})
	t.Run("nil Marshaller should error", func(t *testing.T) {
		t.Parallel()

		args := createMockArgInterceptedEquivalentProof()
		args.Marshaller = nil
		iep, err := NewInterceptedEquivalentProof(args)
		require.Equal(t, process.ErrNilMarshalizer, err)
		require.Nil(t, iep)
	})
	t.Run("nil ShardCoordinator should error", func(t *testing.T) {
		t.Parallel()

		args := createMockArgInterceptedEquivalentProof()
		args.ShardCoordinator = nil
		iep, err := NewInterceptedEquivalentProof(args)
		require.Equal(t, process.ErrNilShardCoordinator, err)
		require.Nil(t, iep)
	})
	t.Run("nil HeaderSigVerifier should error", func(t *testing.T) {
		t.Parallel()

		args := createMockArgInterceptedEquivalentProof()
		args.HeaderSigVerifier = nil
		iep, err := NewInterceptedEquivalentProof(args)
		require.Equal(t, process.ErrNilHeaderSigVerifier, err)
		require.Nil(t, iep)
	})
	t.Run("unmarshal error should error", func(t *testing.T) {
		t.Parallel()

		args := createMockArgInterceptedEquivalentProof()
		args.Marshaller = &marshallerMock.MarshalizerStub{
			UnmarshalCalled: func(obj interface{}, buff []byte) error {
				return expectedErr
			},
		}
		iep, err := NewInterceptedEquivalentProof(args)
		require.Equal(t, expectedErr, err)
		require.Nil(t, iep)
	})
	t.Run("should work", func(t *testing.T) {
		t.Parallel()

		iep, err := NewInterceptedEquivalentProof(createMockArgInterceptedEquivalentProof())
		require.NoError(t, err)
		require.NotNil(t, iep)
	})
}

func TestInterceptedEquivalentProof_CheckValidity(t *testing.T) {
	t.Parallel()

	t.Run("invalid proof should error", func(t *testing.T) {
		t.Parallel()

		// no header hash
		proof := &block.HeaderProof{
			PubKeysBitmap:       []byte("bitmap"),
			AggregatedSignature: []byte("sig"),
		}
		args := createMockArgInterceptedEquivalentProof()
		args.DataBuff, _ = args.Marshaller.Marshal(proof)
		iep, err := NewInterceptedEquivalentProof(args)
		require.NoError(t, err)

		err = iep.CheckValidity()
		require.Equal(t, ErrInvalidProof, err)
	})
	t.Run("should work", func(t *testing.T) {
		t.Parallel()

		iep, err := NewInterceptedEquivalentProof(createMockArgInterceptedEquivalentProof())
		require.NoError(t, err)

		err = iep.CheckValidity()
		require.NoError(t, err)
	})
}

func TestInterceptedEquivalentProof_IsForCurrentShard(t *testing.T) {
	t.Parallel()

	t.Run("meta should return true", func(t *testing.T) {
		t.Parallel()

		proof := &block.HeaderProof{
			PubKeysBitmap:       []byte("bitmap"),
			AggregatedSignature: []byte("sig"),
			HeaderHash:          []byte("hash"),
			HeaderShardId:       core.MetachainShardId,
		}
		args := createMockArgInterceptedEquivalentProof()
		args.DataBuff, _ = args.Marshaller.Marshal(proof)
		args.ShardCoordinator = &mock.ShardCoordinatorMock{ShardID: core.MetachainShardId}
		iep, err := NewInterceptedEquivalentProof(args)
		require.NoError(t, err)

		require.True(t, iep.IsForCurrentShard())
	})
	t.Run("self shard id return true", func(t *testing.T) {
		t.Parallel()

		selfShardId := uint32(1234)
		proof := &block.HeaderProof{
			PubKeysBitmap:       []byte("bitmap"),
			AggregatedSignature: []byte("sig"),
			HeaderHash:          []byte("hash"),
			HeaderShardId:       selfShardId,
		}
		args := createMockArgInterceptedEquivalentProof()
		args.DataBuff, _ = args.Marshaller.Marshal(proof)
		args.ShardCoordinator = &mock.ShardCoordinatorMock{ShardID: selfShardId}
		iep, err := NewInterceptedEquivalentProof(args)
		require.NoError(t, err)

		require.True(t, iep.IsForCurrentShard())
	})
	t.Run("other shard id return true", func(t *testing.T) {
		t.Parallel()

		selfShardId := uint32(1234)
		proof := &block.HeaderProof{
			PubKeysBitmap:       []byte("bitmap"),
			AggregatedSignature: []byte("sig"),
			HeaderHash:          []byte("hash"),
			HeaderShardId:       selfShardId,
		}
		args := createMockArgInterceptedEquivalentProof()
		args.DataBuff, _ = args.Marshaller.Marshal(proof)
		iep, err := NewInterceptedEquivalentProof(args)
		require.NoError(t, err)

		require.False(t, iep.IsForCurrentShard())
	})
}

func TestInterceptedEquivalentProof_Getters(t *testing.T) {
	t.Parallel()

	proof := &block.HeaderProof{
		PubKeysBitmap:       []byte("bitmap"),
		AggregatedSignature: []byte("sig"),
		HeaderHash:          []byte("hash"),
		HeaderEpoch:         123,
		HeaderNonce:         345,
		HeaderShardId:       0,
	}
	args := createMockArgInterceptedEquivalentProof()
	args.DataBuff, _ = args.Marshaller.Marshal(proof)
	iep, err := NewInterceptedEquivalentProof(args)
	require.NoError(t, err)

	require.Equal(t, proof, iep.GetProof()) // pointer testing
	require.True(t, bytes.Equal(proof.HeaderHash, iep.Hash()))
	require.Equal(t, [][]byte{proof.HeaderHash}, iep.Identifiers())
	require.Equal(t, interceptedEquivalentProofType, iep.Type())
	expectedStr := fmt.Sprintf("bitmap=%s, signature=%s, hash=%s, epoch=123, shard=0, nonce=345",
		logger.DisplayByteSlice(proof.PubKeysBitmap),
		logger.DisplayByteSlice(proof.AggregatedSignature),
		logger.DisplayByteSlice(proof.HeaderHash))
	require.Equal(t, expectedStr, iep.String())
}