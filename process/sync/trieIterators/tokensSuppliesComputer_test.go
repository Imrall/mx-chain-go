package trieIterators

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/multiversx/mx-chain-core-go/core/keyValStorage"
	"github.com/multiversx/mx-chain-core-go/data/esdt"
	"github.com/multiversx/mx-chain-go/common"
	"github.com/multiversx/mx-chain-go/dataRetriever"
	coreEsdt "github.com/multiversx/mx-chain-go/dblookupext/esdtSupply"
	"github.com/multiversx/mx-chain-go/state"
	chainStorage "github.com/multiversx/mx-chain-go/storage"
	"github.com/multiversx/mx-chain-go/testscommon"
	"github.com/multiversx/mx-chain-go/testscommon/genericMocks"
	stateMock "github.com/multiversx/mx-chain-go/testscommon/state"
	"github.com/multiversx/mx-chain-go/testscommon/storage"
	"github.com/multiversx/mx-chain-go/testscommon/trie"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
	"github.com/stretchr/testify/require"
)

func getTokensSuppliesProcessorArgs() ArgsTokensSuppliesProcessor {
	return ArgsTokensSuppliesProcessor{
		StorageService: &genericMocks.ChainStorerMock{},
		Marshaller:     &testscommon.MarshalizerMock{},
	}
}

func TestNewTokensSuppliesProcessor(t *testing.T) {
	t.Parallel()

	t.Run("nil storage service", func(t *testing.T) {
		t.Parallel()

		args := getTokensSuppliesProcessorArgs()
		args.StorageService = nil

		tsp, err := NewTokensSuppliesProcessor(args)
		require.Nil(t, tsp)
		require.Equal(t, errNilStorageService, err)
	})

	t.Run("nil marshaller", func(t *testing.T) {
		t.Parallel()

		args := getTokensSuppliesProcessorArgs()
		args.Marshaller = nil

		tsp, err := NewTokensSuppliesProcessor(args)
		require.Nil(t, tsp)
		require.Equal(t, errNilMarshaller, err)
	})

	t.Run("should work", func(t *testing.T) {
		t.Parallel()

		args := getTokensSuppliesProcessorArgs()

		tsp, err := NewTokensSuppliesProcessor(args)
		require.NotNil(t, tsp)
		require.NoError(t, err)
	})
}

func TestTokensSuppliesProcessor_HandleTrieAccountIteration(t *testing.T) {
	t.Parallel()

	t.Run("nil user account", func(t *testing.T) {
		t.Parallel()

		tsp, _ := NewTokensSuppliesProcessor(getTokensSuppliesProcessorArgs())
		err := tsp.HandleTrieAccountIteration(nil)
		require.Equal(t, errNilUserAccount, err)
	})

	t.Run("should skip system account", func(t *testing.T) {
		t.Parallel()

		tsp, _ := NewTokensSuppliesProcessor(getTokensSuppliesProcessorArgs())

		userAcc := stateMock.NewAccountWrapMock(vmcommon.SystemAccountAddress)
		err := tsp.HandleTrieAccountIteration(userAcc)
		require.NoError(t, err)
	})

	t.Run("empty root hash of account", func(t *testing.T) {
		t.Parallel()

		tsp, _ := NewTokensSuppliesProcessor(getTokensSuppliesProcessorArgs())

		userAcc := stateMock.NewAccountWrapMock([]byte("addr"))
		err := tsp.HandleTrieAccountIteration(userAcc)
		require.NoError(t, err)
	})

	t.Run("should work", func(t *testing.T) {
		t.Parallel()

		args := getTokensSuppliesProcessorArgs()
		tsp, _ := NewTokensSuppliesProcessor(args)

		userAcc, _ := state.NewUserAccount([]byte("addr"))
		userAcc.SetRootHash([]byte("rootHash"))
		userAcc.SetDataTrie(&trie.TrieStub{
			GetAllLeavesOnChannelCalled: func(leavesChannels *common.TrieIteratorChannels, ctx context.Context, rootHash []byte, keyBuilder common.KeyBuilder) error {
				esToken := &esdt.ESDigitalToken{
					Value: big.NewInt(37),
				}
				esBytes, _ := args.Marshaller.Marshal(esToken)
				tknKey := []byte("ELRONDesdtTKN-00aacc")
				value := append(esBytes, tknKey...)
				value = append(value, []byte("addr")...)
				leavesChannels.LeavesChan <- keyValStorage.NewKeyValStorage(tknKey, value)

				sft := &esdt.ESDigitalToken{
					Value: big.NewInt(1),
				}
				sftBytes, _ := args.Marshaller.Marshal(sft)
				sftKey := []byte("ELRONDesdtSFT-00aabb")
				sftKey = append(sftKey, big.NewInt(37).Bytes()...)
				value = append(sftBytes, sftKey...)
				value = append(value, []byte("addr")...)
				leavesChannels.LeavesChan <- keyValStorage.NewKeyValStorage(sftKey, value)

				close(leavesChannels.LeavesChan)
				return nil
			},
		})

		err := tsp.HandleTrieAccountIteration(userAcc)
		require.NoError(t, err)

		err = tsp.HandleTrieAccountIteration(userAcc)
		require.NoError(t, err)

		expectedSupplies := map[string]*big.Int{
			"SFT-00aabb-37": big.NewInt(2),
			"SFT-00aabb":    big.NewInt(2),
			"TKN-00aacc":    big.NewInt(74),
		}
		require.Equal(t, expectedSupplies, tsp.tokensSupplies)
	})
}

func TestTokensSuppliesProcessor_SaveSupplies(t *testing.T) {
	t.Parallel()

	t.Run("cannot find esdt supplies storer", func(t *testing.T) {
		t.Parallel()

		errStorerNotFound := errors.New("storer not found")
		args := getTokensSuppliesProcessorArgs()
		args.StorageService = &storage.ChainStorerStub{
			GetStorerCalled: func(unitType dataRetriever.UnitType) (chainStorage.Storer, error) {
				return nil, errStorerNotFound
			},
		}
		tsp, _ := NewTokensSuppliesProcessor(args)
		err := tsp.SaveSupplies()
		require.Equal(t, errStorerNotFound, err)
	})

	t.Run("should work", func(t *testing.T) {
		t.Parallel()

		savedItems := make(map[string][]byte)
		args := getTokensSuppliesProcessorArgs()
		args.StorageService = &storage.ChainStorerStub{
			GetStorerCalled: func(unitType dataRetriever.UnitType) (chainStorage.Storer, error) {
				return &storage.StorerStub{
					PutCalled: func(key, data []byte) error {
						savedItems[string(key)] = data
						return nil
					},
				}, nil
			},
		}
		tsp, _ := NewTokensSuppliesProcessor(args)

		supplies := map[string]*big.Int{
			"SFT-00aabb-37": big.NewInt(2),
			"SFT-00aabb":    big.NewInt(2),
			"TKN-00aacc":    big.NewInt(74),
		}
		tsp.tokensSupplies = supplies

		err := tsp.SaveSupplies()
		require.NoError(t, err)

		checkStoredSupply := func(t *testing.T, key string, storedValue []byte, expectedSupply *big.Int) {
			supply := coreEsdt.SupplyESDT{}
			_ = args.Marshaller.Unmarshal(&supply, storedValue)
			require.Equal(t, expectedSupply, supply.Supply)
		}

		require.Len(t, savedItems, 3)
		for key, value := range savedItems {
			checkStoredSupply(t, key, value, supplies[key])
		}
	})
}
