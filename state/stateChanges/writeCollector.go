package stateChanges

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"

	"github.com/multiversx/mx-chain-core-go/data/transaction"
	logger "github.com/multiversx/mx-chain-logger-go"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
)

var log = logger.GetOrCreate("state/stateChanges")

// DataTrieChange represents a change in the data trie
type DataTrieChange struct {
	Type string `json:"type"`
	Key  []byte `json:"key"`
	Val  []byte `json:"-"`
}

// ErrStateChangesIndexOutOfBounds signals that the state changes index is out of bounds
var ErrStateChangesIndexOutOfBounds = errors.New("state changes index out of bounds")

type StateChange interface {
	GetTxHash() []byte
	SetTxHash(txHash []byte)
	GetIndex() int
	SetIndex(index int)
}

// StateChangeDTO is used to collect state changes
// TODO: change to use proto structs
type StateChangeDTO struct {
	Type            string           `json:"type"`
	Index           int              `json:"-"`
	TxHash          []byte           `json:"-"`
	MainTrieKey     []byte           `json:"mainTrieKey"`
	MainTrieVal     []byte           `json:"-"`
	Operation       string           `json:"operation"`
	DataTrieChanges []DataTrieChange `json:"dataTrieChanges"`
}

func (sc *StateChangeDTO) GetIndex() int {
	return sc.Index
}

func (sc *StateChangeDTO) SetIndex(index int) {
	sc.Index = index
}

func (sc *StateChangeDTO) GetTxHash() []byte {
	return sc.TxHash
}

func (sc *StateChangeDTO) SetTxHash(txHash []byte) {
	sc.TxHash = txHash
}

// StateChangesForTx is used to collect state changes for a transaction hash
type StateChangesForTx struct {
	TxHash       []byte        `json:"txHash"`
	StateChanges []StateChange `json:"stateChanges"`
}

type stateChangesCollector struct {
	stateChanges    []StateChange
	stateChangesMut sync.RWMutex
}

// NewStateChangesCollector creates a new StateChangesCollector
func NewStateChangesCollector() *stateChangesCollector {
	// TODO: add outport driver

	return &stateChangesCollector{
		stateChanges: make([]StateChange, 0),
	}
}

// AddSaveAccountStateChange adds a new state change for the save account operation
func (scc *stateChangesCollector) AddSaveAccountStateChange(_, _ vmcommon.AccountHandler, stateChange StateChange) {
	scc.AddStateChange(stateChange)
}

// AddStateChange adds a new state change to the collector
func (scc *stateChangesCollector) AddStateChange(stateChange StateChange) {
	scc.stateChangesMut.Lock()
	scc.stateChanges = append(scc.stateChanges, stateChange)
	scc.stateChangesMut.Unlock()
}

func (scc *stateChangesCollector) getStateChangesForTxs() ([]StateChangesForTx, error) {
	scc.stateChangesMut.Lock()
	defer scc.stateChangesMut.Unlock()

	stateChangesForTxs := make([]StateChangesForTx, 0)

	for i := 0; i < len(scc.stateChanges); i++ {
		txHash := scc.stateChanges[i].GetTxHash()

		if len(txHash) == 0 {
			log.Warn("empty tx hash, state change event not associated to a transaction")
			break
		}

		innerStateChangesForTx := make([]StateChange, 0)
		for j := i; j < len(scc.stateChanges); j++ {
			txHash2 := scc.stateChanges[j].GetTxHash()
			if !bytes.Equal(txHash, txHash2) {
				i = j
				break
			}

			innerStateChangesForTx = append(innerStateChangesForTx, scc.stateChanges[j])
			i = j
		}

		stateChangesForTx := StateChangesForTx{
			TxHash:       txHash,
			StateChanges: innerStateChangesForTx,
		}
		stateChangesForTxs = append(stateChangesForTxs, stateChangesForTx)
	}

	return stateChangesForTxs, nil
}

// Reset resets the state changes collector
func (scc *stateChangesCollector) Reset() {
	scc.stateChangesMut.Lock()
	defer scc.stateChangesMut.Unlock()

	scc.stateChanges = make([]StateChange, 0)
}

// AddTxHashToCollectedStateChanges will try to set txHash field to each state change
// if the field is not already set
func (scc *stateChangesCollector) AddTxHashToCollectedStateChanges(txHash []byte, tx *transaction.Transaction) {
	scc.stateChangesMut.Lock()
	defer scc.stateChangesMut.Unlock()

	for i := len(scc.stateChanges) - 1; i >= 0; i-- {
		if len(scc.stateChanges[i].GetTxHash()) > 0 {
			break
		}

		scc.stateChanges[i].SetTxHash(txHash)
	}
}

// SetIndexToLastStateChange will set index to the last state change
func (scc *stateChangesCollector) SetIndexToLastStateChange(index int) error {
	if index > len(scc.stateChanges) || index < 0 {
		return ErrStateChangesIndexOutOfBounds
	}

	scc.stateChangesMut.Lock()
	defer scc.stateChangesMut.Unlock()

	scc.stateChanges[len(scc.stateChanges)-1].SetIndex(index)

	return nil
}

// RevertToIndex will revert to index
func (scc *stateChangesCollector) RevertToIndex(index int) error {
	if index > len(scc.stateChanges) || index < 0 {
		return ErrStateChangesIndexOutOfBounds
	}

	if index == 0 {
		scc.Reset()
		return nil
	}

	scc.stateChangesMut.Lock()
	defer scc.stateChangesMut.Unlock()

	for i := len(scc.stateChanges) - 1; i >= 0; i-- {
		if scc.stateChanges[i].GetIndex() == index {
			scc.stateChanges = scc.stateChanges[:i]
			break
		}
	}

	return nil
}

// Publish will export state changes
func (scc *stateChangesCollector) Publish() error {
	stateChangesForTx, err := scc.getStateChangesForTxs()
	if err != nil {
		return err
	}

	printStateChanges(stateChangesForTx)

	return nil
}

func (scc *stateChangesCollector) RetrieveStateChanges() []StateChange {
	return scc.stateChanges
}

func printStateChanges(stateChanges []StateChangesForTx) {
	for _, stateChange := range stateChanges {

		if stateChange.TxHash != nil {
			fmt.Println(hex.EncodeToString(stateChange.TxHash))
		}

		for _, st := range stateChange.StateChanges {
			fmt.Println(st)
		}
	}
}

// IsInterfaceNil returns true if there is no value under the interface
func (scc *stateChangesCollector) IsInterfaceNil() bool {
	return scc == nil
}