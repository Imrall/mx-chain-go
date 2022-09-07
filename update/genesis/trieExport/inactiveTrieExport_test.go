package trieExport

import (
	"context"
	"fmt"
	"testing"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-go-core/core/keyValStorage"
	"github.com/ElrondNetwork/elrond-go-core/data"
	"github.com/ElrondNetwork/elrond-go/process/mock"
	"github.com/ElrondNetwork/elrond-go/state"
	"github.com/ElrondNetwork/elrond-go/testscommon/trie"
	"github.com/stretchr/testify/assert"
)

func TestNewInactiveTrieExporter_NilMarshalizerShouldErr(t *testing.T) {
	t.Parallel()

	ite, err := NewInactiveTrieExporter(nil)
	assert.True(t, check.IfNil(ite))
	assert.Equal(t, data.ErrNilMarshalizer, err)
}

func TestNewInactiveTrieExporter(t *testing.T) {
	t.Parallel()

	ite, err := NewInactiveTrieExporter(&mock.MarshalizerMock{})
	assert.Nil(t, err)
	assert.NotNil(t, ite)
}

func TestInactiveTrieExport_ExportValidatorTrieDoesNothing(t *testing.T) {
	t.Parallel()

	ite, _ := NewInactiveTrieExporter(&mock.MarshalizerMock{})
	err := ite.ExportValidatorTrie(nil)
	assert.Nil(t, err)
}

func TestInactiveTrieExport_ExportDataTrieDoesNothing(t *testing.T) {
	t.Parallel()

	ite, _ := NewInactiveTrieExporter(&mock.MarshalizerMock{})
	err := ite.ExportDataTrie("", nil)
	assert.Nil(t, err)
}

func TestInactiveTrieExport_ExportMainTrieInvalidTrieRootHashShouldErr(t *testing.T) {
	t.Parallel()

	ite, _ := NewInactiveTrieExporter(&mock.MarshalizerMock{})

	expectedErr := fmt.Errorf("rootHash err")
	tr := &trie.TrieStub{
		RootCalled: func() ([]byte, error) {
			return nil, expectedErr
		},
	}

	rootHashes, err := ite.ExportMainTrie("", tr)
	assert.Equal(t, expectedErr, err)
	assert.Nil(t, rootHashes)
}

func TestInactiveTrieExport_ExportMainTrieGetAllLeavesOnChannelErrShouldErr(t *testing.T) {
	t.Parallel()

	ite, _ := NewInactiveTrieExporter(&mock.MarshalizerMock{})

	expectedErr := fmt.Errorf("getAllLeavesOnChannel err")
	tr := &trie.TrieStub{
		RootCalled: func() ([]byte, error) {
			return nil, nil
		},
		GetAllLeavesOnChannelCalled: func(leavesChannel chan core.KeyValueHolder, ctx context.Context, rootHash []byte) error {
			return expectedErr
		},
	}

	rootHashes, err := ite.ExportMainTrie("", tr)
	assert.Equal(t, expectedErr, err)
	assert.Nil(t, rootHashes)
}

func TestInactiveTrieExport_ExportMainTrieShouldReturnDataTrieRootHashes(t *testing.T) {
	t.Parallel()

	marshalizer := &mock.MarshalizerMock{}
	trieExporter, _ := NewInactiveTrieExporter(marshalizer)

	expectedRootHash := []byte("rootHash")
	account1 := state.NewEmptyUserAccount()
	account2 := state.NewEmptyUserAccount()
	account2.RootHash = expectedRootHash

	serializedAcc1, err := marshalizer.Marshal(account1)
	assert.Nil(t, err)
	serializedAcc2, err := marshalizer.Marshal(account2)
	assert.Nil(t, err)

	tr := &trie.TrieStub{
		RootCalled: func() ([]byte, error) {
			return nil, nil
		},
		GetAllLeavesOnChannelCalled: func(leavesChannel chan core.KeyValueHolder, ctx context.Context, rootHash []byte) error {
			leavesChannel <- keyValStorage.NewKeyValStorage([]byte("key1"), []byte("val1"))
			leavesChannel <- keyValStorage.NewKeyValStorage([]byte("key2"), serializedAcc1)
			leavesChannel <- keyValStorage.NewKeyValStorage([]byte("key3"), serializedAcc2)
			close(leavesChannel)
			return nil
		},
	}

	rootHashes, err := trieExporter.ExportMainTrie("", tr)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(rootHashes))
	assert.Equal(t, expectedRootHash, rootHashes[0])
}
