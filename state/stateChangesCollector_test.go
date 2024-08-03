package state

import (
	"strconv"
	"testing"

	"github.com/multiversx/mx-chain-core-go/data/transaction"
	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
)

func TestNewStateChangesCollector(t *testing.T) {
	t.Parallel()

	stateChangesCollector := NewStateChangesCollector()
	require.False(t, stateChangesCollector.IsInterfaceNil())
}

func TestStateChangesCollector_AddStateChange(t *testing.T) {
	t.Parallel()

	scc := NewStateChangesCollector()
	assert.Equal(t, 0, len(scc.stateChanges))

	numStateChanges := 10
	for i := 0; i < numStateChanges; i++ {
		scc.AddStateChange(StateChangeDTO{})
	}
	assert.Equal(t, numStateChanges, len(scc.stateChanges))
}

func TestStateChangesCollector_GetStateChanges(t *testing.T) {
	t.Parallel()

	t.Run("getStateChanges with tx hash", func(t *testing.T) {
		t.Parallel()

		scc := NewStateChangesCollector()
		assert.Equal(t, 0, len(scc.stateChanges))
		assert.Equal(t, 0, len(scc.GetStateChanges()))

		numStateChanges := 10
		for i := 0; i < numStateChanges; i++ {
			scc.AddStateChange(StateChangeDTO{
				MainTrieKey: []byte(strconv.Itoa(i)),
			})
		}
		assert.Equal(t, numStateChanges, len(scc.stateChanges))
		assert.Equal(t, 0, len(scc.GetStateChanges()))
		scc.AddTxHashToCollectedStateChanges([]byte("txHash"), &transaction.Transaction{})
		assert.Equal(t, numStateChanges, len(scc.stateChanges))
		assert.Equal(t, 1, len(scc.GetStateChanges()))
		assert.Equal(t, []byte("txHash"), scc.GetStateChanges()[0].TxHash)
		assert.Equal(t, numStateChanges, len(scc.GetStateChanges()[0].StateChanges))

		stateChangesForTx := scc.GetStateChanges()
		assert.Equal(t, 1, len(stateChangesForTx))
		assert.Equal(t, []byte("txHash"), stateChangesForTx[0].TxHash)
		for i := 0; i < len(stateChangesForTx[0].StateChanges); i++ {
			assert.Equal(t, []byte(strconv.Itoa(i)), stateChangesForTx[0].StateChanges[i].MainTrieKey)
		}
	})

	t.Run("getStateChanges without tx hash", func(t *testing.T) {
		t.Parallel()

		scc := NewStateChangesCollector()
		assert.Equal(t, 0, len(scc.stateChanges))
		assert.Equal(t, 0, len(scc.GetStateChanges()))

		numStateChanges := 10
		for i := 0; i < numStateChanges; i++ {
			scc.AddStateChange(StateChangeDTO{
				MainTrieKey: []byte(strconv.Itoa(i)),
			})
		}
		assert.Equal(t, numStateChanges, len(scc.stateChanges))
		assert.Equal(t, 0, len(scc.GetStateChanges()))

		stateChangesForTx := scc.GetStateChanges()
		assert.Equal(t, 0, len(stateChangesForTx))
		// assert.Equal(t, []byte{}, stateChangesForTx[0].TxHash)
		// for i := 0; i < len(stateChangesForTx[0].StateChanges); i++ {
		// 	assert.Equal(t, []byte(strconv.Itoa(i)), stateChangesForTx[0].StateChanges[i].MainTrieKey)
		// }
	})
}

func TestStateChangesCollector_AddTxHashToCollectedStateChanges(t *testing.T) {
	t.Parallel()

	scc := NewStateChangesCollector()
	assert.Equal(t, 0, len(scc.stateChanges))
	assert.Equal(t, 0, len(scc.GetStateChanges()))

	stateChange := StateChangeDTO{
		MainTrieKey:     []byte("mainTrieKey"),
		MainTrieVal:     []byte("mainTrieVal"),
		DataTrieChanges: []DataTrieChange{{Key: []byte("dataTrieKey"), Val: []byte("dataTrieVal")}},
	}
	scc.AddStateChange(stateChange)

	assert.Equal(t, 1, len(scc.stateChanges))
	assert.Equal(t, 0, len(scc.GetStateChanges()))
	scc.AddTxHashToCollectedStateChanges([]byte("txHash"), &transaction.Transaction{})
	assert.Equal(t, 1, len(scc.stateChanges))
	assert.Equal(t, 1, len(scc.GetStateChanges()))

	stateChangesForTx := scc.GetStateChanges()
	assert.Equal(t, 1, len(stateChangesForTx))
	assert.Equal(t, []byte("txHash"), stateChangesForTx[0].TxHash)
	assert.Equal(t, 1, len(stateChangesForTx[0].StateChanges))
	assert.Equal(t, []byte("mainTrieKey"), stateChangesForTx[0].StateChanges[0].MainTrieKey)
	assert.Equal(t, []byte("mainTrieVal"), stateChangesForTx[0].StateChanges[0].MainTrieVal)
	assert.Equal(t, 1, len(stateChangesForTx[0].StateChanges[0].DataTrieChanges))
}

func TestStateChangesCollector_Reset(t *testing.T) {
	t.Parallel()

	scc := NewStateChangesCollector()
	assert.Equal(t, 0, len(scc.stateChanges))

	numStateChanges := 10
	for i := 0; i < numStateChanges; i++ {
		scc.AddStateChange(StateChangeDTO{})
	}
	scc.AddTxHashToCollectedStateChanges([]byte("txHash"), &transaction.Transaction{})
	for i := numStateChanges; i < numStateChanges*2; i++ {
		scc.AddStateChange(StateChangeDTO{})
	}
	assert.Equal(t, numStateChanges*2, len(scc.stateChanges))

	assert.Equal(t, 1, len(scc.GetStateChanges()))

	scc.Reset()
	assert.Equal(t, 0, len(scc.stateChanges))

	assert.Equal(t, 0, len(scc.GetStateChanges()))
}
