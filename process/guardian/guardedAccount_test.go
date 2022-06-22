package guardian

import (
	"errors"
	"testing"

	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-go-core/data/guardians"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/testscommon"
	"github.com/ElrondNetwork/elrond-go/testscommon/epochNotifier"
	stateMocks "github.com/ElrondNetwork/elrond-go/testscommon/state"
	"github.com/ElrondNetwork/elrond-go/testscommon/trie"
	"github.com/ElrondNetwork/elrond-go/testscommon/vmcommonMocks"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
	"github.com/stretchr/testify/require"
)

func TestNewGuardedAccount(t *testing.T) {
	marshaller := &testscommon.MarshalizerMock{}
	en := &epochNotifier.EpochNotifierStub{}
	ga, err := NewGuardedAccount(marshaller, en, 10)
	require.Nil(t, err)
	require.NotNil(t, ga)

	ga, err = NewGuardedAccount(nil, en, 10)
	require.Equal(t, process.ErrNilMarshalizer, err)
	require.Nil(t, ga)

	ga, err = NewGuardedAccount(marshaller, nil, 10)
	require.Equal(t, process.ErrNilEpochNotifier, err)
	require.Nil(t, ga)

	ga, err = NewGuardedAccount(marshaller, en, 0)
	require.Equal(t, process.ErrInvalidSetGuardianEpochsDelay, err)
	require.Nil(t, ga)
}

func TestGuardedAccount_getActiveGuardian(t *testing.T) {
	ga := createGuardedAccountWithEpoch(9)

	t.Run("no guardians", func(t *testing.T) {
		t.Parallel()

		configuredGuardians := &guardians.Guardians{}
		activeGuardian, err := ga.getActiveGuardian(configuredGuardians)
		require.Nil(t, activeGuardian)
		require.Equal(t, process.ErrAccountHasNoActiveGuardian, err)
	})
	t.Run("one pending guardian", func(t *testing.T) {
		t.Parallel()

		g1 := &guardians.Guardian{Address: []byte("addr1"), ActivationEpoch: 11}
		configuredGuardians := &guardians.Guardians{Slice: []*guardians.Guardian{g1}}
		activeGuardian, err := ga.getActiveGuardian(configuredGuardians)
		require.Nil(t, activeGuardian)
		require.Equal(t, process.ErrAccountHasNoActiveGuardian, err)
	})
	t.Run("one active guardian", func(t *testing.T) {
		t.Parallel()

		g1 := &guardians.Guardian{Address: []byte("addr1"), ActivationEpoch: 9}
		configuredGuardians := &guardians.Guardians{Slice: []*guardians.Guardian{g1}}
		activeGuardian, err := ga.getActiveGuardian(configuredGuardians)
		require.Nil(t, err)
		require.Equal(t, g1, activeGuardian)
	})
	t.Run("one active and one pending", func(t *testing.T) {
		t.Parallel()

		g1 := &guardians.Guardian{Address: []byte("addr1"), ActivationEpoch: 9}
		g2 := &guardians.Guardian{Address: []byte("addr2"), ActivationEpoch: 30}
		configuredGuardians := &guardians.Guardians{Slice: []*guardians.Guardian{g1, g2}}
		activeGuardian, err := ga.getActiveGuardian(configuredGuardians)
		require.Nil(t, err)
		require.Equal(t, g1, activeGuardian)
	})
	t.Run("one active and one too old", func(t *testing.T) {
		t.Parallel()

		g1 := &guardians.Guardian{Address: []byte("addr1"), ActivationEpoch: 8}
		g2 := &guardians.Guardian{Address: []byte("addr2"), ActivationEpoch: 9}
		configuredGuardians := &guardians.Guardians{Slice: []*guardians.Guardian{g1, g2}}
		activeGuardian, err := ga.getActiveGuardian(configuredGuardians)
		require.Nil(t, err)
		require.Equal(t, g2, activeGuardian)
	})
	t.Run("one active and one too old, saved in reverse order", func(t *testing.T) {
		t.Parallel()

		g1 := &guardians.Guardian{Address: []byte("addr1"), ActivationEpoch: 8}
		g2 := &guardians.Guardian{Address: []byte("addr2"), ActivationEpoch: 9}
		configuredGuardians := &guardians.Guardians{Slice: []*guardians.Guardian{g2, g1}}
		activeGuardian, err := ga.getActiveGuardian(configuredGuardians)
		require.Nil(t, err)
		require.Equal(t, g2, activeGuardian)
	})
}

func TestGuardedAccount_getConfiguredGuardians(t *testing.T) {
	ga := createGuardedAccountWithEpoch(10)

	t.Run("guardians key not found should err", func(t *testing.T) {
		t.Parallel()

		expectedErr := errors.New("expected error")
		acc := &stateMocks.UserAccountStub{
			RetrieveValueFromDataTrieTrackerCalled: func(key []byte) ([]byte, error) {
				return nil, expectedErr
			},
		}

		configuredGuardians, err := ga.getConfiguredGuardians(acc)
		require.Nil(t, configuredGuardians)
		require.Equal(t, expectedErr, err)
	})
	t.Run("key found but no guardians, should return empty", func(t *testing.T) {
		t.Parallel()

		acc := &stateMocks.UserAccountStub{
			RetrieveValueFromDataTrieTrackerCalled: func(key []byte) ([]byte, error) {
				return nil, nil
			},
		}

		configuredGuardians, err := ga.getConfiguredGuardians(acc)
		require.Nil(t, err)
		require.NotNil(t, configuredGuardians)
		require.True(t, len(configuredGuardians.Slice) == 0)
	})
	t.Run("unmarshal guardians error should return error", func(t *testing.T) {
		t.Parallel()

		expectedErr := errors.New("expected error")
		ga := createGuardedAccountWithEpoch(10)
		ga.marshaller = &testscommon.MarshalizerStub{
			UnmarshalCalled: func(obj interface{}, buff []byte) error {
				return expectedErr
			},
		}
		acc := &stateMocks.UserAccountStub{
			RetrieveValueFromDataTrieTrackerCalled: func(key []byte) ([]byte, error) {
				return []byte("wrongly marshalled guardians"), nil
			},
		}

		configuredGuardians, err := ga.getConfiguredGuardians(acc)
		require.Nil(t, configuredGuardians)
		require.Equal(t, expectedErr, err)
	})
	t.Run("unmarshal guardians error should return error", func(t *testing.T) {
		t.Parallel()

		g1 := &guardians.Guardian{Address: []byte("addr1"), ActivationEpoch: 9}
		expectedConfiguredGuardians := &guardians.Guardians{Slice: []*guardians.Guardian{g1}}

		acc := &stateMocks.UserAccountStub{
			RetrieveValueFromDataTrieTrackerCalled: func(key []byte) ([]byte, error) {
				return ga.marshaller.Marshal(expectedConfiguredGuardians)
			},
		}

		configuredGuardians, err := ga.getConfiguredGuardians(acc)
		require.Nil(t, err)
		require.Equal(t, expectedConfiguredGuardians, configuredGuardians)
	})
}

func TestGuardedAccount_saveAccountGuardians(t *testing.T) {
	userAccount := &vmcommonMocks.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &trie.DataTrieTrackerStub{
				SaveKeyValueCalled: func(key []byte, value []byte) error {
					return nil
				},
			}
		},
	}

	t.Run("marshaling error should return err", func(t *testing.T) {
		t.Parallel()

		expectedErr := errors.New("expected error")
		ga := createGuardedAccountWithEpoch(10)
		ga.marshaller = &testscommon.MarshalizerStub{
			MarshalCalled: func(obj interface{}) ([]byte, error) {
				return nil, expectedErr
			},
		}

		err := ga.saveAccountGuardians(userAccount, nil)
		require.Equal(t, expectedErr, err)
	})
	t.Run("save account guardians OK", func(t *testing.T) {
		t.Parallel()

		SaveKeyValueCalled := false
		userAccount.AccountDataHandlerCalled = func() vmcommon.AccountDataHandler {
			return &trie.DataTrieTrackerStub{
				SaveKeyValueCalled: func(key []byte, value []byte) error {
					SaveKeyValueCalled = true
					return nil
				},
			}
		}

		ga := createGuardedAccountWithEpoch(10)
		err := ga.saveAccountGuardians(userAccount, nil)
		require.Nil(t, err)
		require.True(t, SaveKeyValueCalled)
	})
}

func TestGuardedAccount_updateGuardians(t *testing.T) {
	ga := createGuardedAccountWithEpoch(10)
	newGuardian := &guardians.Guardian{
		Address:         []byte("new guardian address"),
		ActivationEpoch: 20,
	}

	t.Run("update empty guardian list with new guardian", func(t *testing.T) {
		configuredGuardians := &guardians.Guardians{Slice: []*guardians.Guardian{}}
		expectedGuardians := append(configuredGuardians.Slice, newGuardian)
		updatedGuardians, err := ga.updateGuardians(newGuardian, configuredGuardians)
		require.Nil(t, err)
		require.Equal(t, expectedGuardians, updatedGuardians.Slice)
	})
	t.Run("updating when there is an existing pending guardian and no active should error", func(t *testing.T) {
		existingGuardian := &guardians.Guardian{
			Address:         []byte("guardian address"),
			ActivationEpoch: 11,
		}
		configuredGuardians := &guardians.Guardians{Slice: []*guardians.Guardian{existingGuardian}}

		updatedGuardians, err := ga.updateGuardians(newGuardian, configuredGuardians)
		require.Nil(t, updatedGuardians)
		require.True(t, errors.Is(err, process.ErrAccountHasNoActiveGuardian))
	})
	t.Run("updating the existing same active guardian should leave the active guardian unchanged", func(t *testing.T) {
		existingGuardian := &guardians.Guardian{
			Address:         []byte("guardian address"),
			ActivationEpoch: 9,
		}

		newGuardian := newGuardian
		newGuardian.Address = existingGuardian.Address
		configuredGuardians := &guardians.Guardians{Slice: []*guardians.Guardian{existingGuardian}}

		updatedGuardians, err := ga.updateGuardians(newGuardian, configuredGuardians)
		require.Nil(t, err)
		require.Equal(t, configuredGuardians, updatedGuardians)
	})
	t.Run("updating the existing same active guardian, when there is also a pending guardian configured, should clean up pending and leave active unchanged", func(t *testing.T) {
		existingActiveGuardian := &guardians.Guardian{
			Address:         []byte("guardian address"),
			ActivationEpoch: 9,
		}
		existingPendingGuardian := &guardians.Guardian{
			Address:         []byte("pending guardian address"),
			ActivationEpoch: 13,
		}

		newGuardian := newGuardian
		newGuardian.Address = existingPendingGuardian.Address
		configuredGuardians := &guardians.Guardians{Slice: []*guardians.Guardian{existingActiveGuardian, existingPendingGuardian}}
		expectedUpdatedGuardians := &guardians.Guardians{Slice: []*guardians.Guardian{existingActiveGuardian, newGuardian}}

		updatedGuardians, err := ga.updateGuardians(newGuardian, configuredGuardians)
		require.Nil(t, err)
		require.Equal(t, expectedUpdatedGuardians, updatedGuardians)
	})
	t.Run("updating the existing same pending guardian while there is an active one should leave the active guardian unchanged but update the pending", func(t *testing.T) {
		existingGuardian := &guardians.Guardian{
			Address:         []byte("guardian address"),
			ActivationEpoch: 9,
		}

		newGuardian := newGuardian
		newGuardian.Address = existingGuardian.Address
		configuredGuardians := &guardians.Guardians{Slice: []*guardians.Guardian{existingGuardian}}

		updatedGuardians, err := ga.updateGuardians(newGuardian, configuredGuardians)
		require.Nil(t, err)
		require.Equal(t, configuredGuardians, updatedGuardians)
	})
}

func TestGuardedAccount_setAccountGuardian(t *testing.T) {
	ga := createGuardedAccountWithEpoch(10)
	newGuardian := &guardians.Guardian{
		Address:         []byte("new guardian address"),
		ActivationEpoch: 20,
	}

	t.Run("getConfiguredGuardians with err", func(t *testing.T) {
		expectedErr := errors.New("expected error")
		ua := &stateMocks.UserAccountStub{
			RetrieveValueFromDataTrieTrackerCalled: func(key []byte) ([]byte, error) {
				return nil, expectedErr
			},
		}

		err := ga.setAccountGuardian(ua, newGuardian)
		require.Equal(t, expectedErr, err)
	})
	t.Run("if updateGuardians returns err, the err should be propagated", func(t *testing.T) {
		existingGuardian := &guardians.Guardian{
			Address:         []byte("guardian address"),
			ActivationEpoch: 11,
		}
		configuredGuardians := &guardians.Guardians{Slice: []*guardians.Guardian{existingGuardian}}
		ua := &stateMocks.UserAccountStub{
			RetrieveValueFromDataTrieTrackerCalled: func(key []byte) ([]byte, error) {
				return ga.marshaller.Marshal(configuredGuardians)
			},
		}

		err := ga.setAccountGuardian(ua, newGuardian)
		require.True(t, errors.Is(err, process.ErrAccountHasNoActiveGuardian))
	})
	t.Run("setGuardian same guardian ok, not changing existing config", func(t *testing.T) {
		existingGuardian := &guardians.Guardian{
			Address:         []byte("guardian address"),
			ActivationEpoch: 9,
		}
		newGuardian := newGuardian
		newGuardian.Address = existingGuardian.Address
		configuredGuardians := &guardians.Guardians{Slice: []*guardians.Guardian{existingGuardian}}

		expectedValue := []byte(nil)
		ua := &stateMocks.UserAccountStub{
			RetrieveValueFromDataTrieTrackerCalled: func(key []byte) ([]byte, error) {
				expectedValue, _ = ga.marshaller.Marshal(configuredGuardians)
				return expectedValue, nil
			},
			AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
				return &trie.DataTrieTrackerStub{
					SaveKeyValueCalled: func(key []byte, value []byte) error {
						require.Equal(t, guardianKey, key)
						require.Equal(t, expectedValue, value)
						return nil
					},
				}
			},
		}

		err := ga.setAccountGuardian(ua, newGuardian)
		require.Nil(t, err)
	})
}

func TestGuardedAccount_instantSetGuardian(t *testing.T) {
	currentEpoch := uint32(10)
	ga := createGuardedAccountWithEpoch(currentEpoch)
	newGuardian := &guardians.Guardian{
		Address:         []byte("new guardian address"),
		ActivationEpoch: 20,
	}
	txGuardianAddress := []byte("guardian address")

	t.Run("getConfiguredGuardians with err", func(t *testing.T) {
		expectedErr := errors.New("expected error")
		ua := &stateMocks.UserAccountStub{
			RetrieveValueFromDataTrieTrackerCalled: func(key []byte) ([]byte, error) {
				return nil, expectedErr
			},
		}

		err := ga.instantSetGuardian(ua, newGuardian.Address, txGuardianAddress)
		require.Equal(t, expectedErr, err)
	})
	t.Run("getActiveGuardianErr with err (no active guardian) should error", func(t *testing.T) {
		configuredGuardians := &guardians.Guardians{Slice: []*guardians.Guardian{}}

		ua := &stateMocks.UserAccountStub{
			RetrieveValueFromDataTrieTrackerCalled: func(key []byte) ([]byte, error) {
				return ga.marshaller.Marshal(configuredGuardians)
			},
		}

		err := ga.instantSetGuardian(ua, newGuardian.Address, txGuardianAddress)
		require.Equal(t, process.ErrAccountHasNoActiveGuardian, err)
	})
	t.Run("tx signed by different than active guardian should err", func(t *testing.T) {
		activeGuardian := &guardians.Guardian{
			Address:         []byte("active guardian address"),
			ActivationEpoch: 1,
		}

		configuredGuardians := &guardians.Guardians{Slice: []*guardians.Guardian{activeGuardian}}

		ua := &stateMocks.UserAccountStub{
			RetrieveValueFromDataTrieTrackerCalled: func(key []byte) ([]byte, error) {
				return ga.marshaller.Marshal(configuredGuardians)
			},
		}

		err := ga.instantSetGuardian(ua, newGuardian.Address, txGuardianAddress)
		require.Equal(t, process.ErrTransactionAndAccountGuardianMismatch, err)
	})
	t.Run("immediately set the guardian if setGuardian tx is signed by active guardian", func(t *testing.T) {
		activeGuardian := &guardians.Guardian{
			Address:         txGuardianAddress,
			ActivationEpoch: 1,
		}
		newGuardian := &guardians.Guardian{
			Address:         []byte("new guardian address"),
			ActivationEpoch: currentEpoch,
		}

		configuredGuardians := &guardians.Guardians{Slice: []*guardians.Guardian{activeGuardian}}
		expectedValue, _ := ga.marshaller.Marshal(&guardians.Guardians{Slice: []*guardians.Guardian{newGuardian}})

		ua := &stateMocks.UserAccountStub{
			RetrieveValueFromDataTrieTrackerCalled: func(key []byte) ([]byte, error) {
				return ga.marshaller.Marshal(configuredGuardians)
			},
			AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
				return &trie.DataTrieTrackerStub{
					SaveKeyValueCalled: func(key []byte, value []byte) error {
						require.Equal(t, guardianKey, key)
						require.Equal(t, expectedValue, value)
						return nil
					},
				}
			}}

		err := ga.instantSetGuardian(ua, newGuardian.Address, txGuardianAddress)
		require.Nil(t, err)
	})
}

func TestGuardedAccount_GetActiveGuardian(t *testing.T) {
	currentEpoch := uint32(10)
	ga := createGuardedAccountWithEpoch(currentEpoch)

	t.Run("wrong account type should err", func(t *testing.T) {
		var uah *vmcommonMocks.UserAccountStub
		activeGuardian, err := ga.GetActiveGuardian(uah)
		require.Nil(t, activeGuardian)
		require.Equal(t, process.ErrWrongTypeAssertion, err)
	})
	t.Run("getConfiguredGuardians with err should propagate the err", func(t *testing.T) {
		expectedErr := errors.New("expected error")
		uah := &stateMocks.UserAccountStub{
			RetrieveValueFromDataTrieTrackerCalled: func(key []byte) ([]byte, error) {
				return nil, expectedErr
			},
		}
		activeGuardian, err := ga.GetActiveGuardian(uah)
		require.Nil(t, activeGuardian)
		require.Equal(t, expectedErr, err)
	})
	t.Run("no guardian should return err", func(t *testing.T) {
		configuredGuardians := &guardians.Guardians{}
		uah := &stateMocks.UserAccountStub{
			RetrieveValueFromDataTrieTrackerCalled: func(key []byte) ([]byte, error) {
				return ga.marshaller.Marshal(configuredGuardians)
			},
		}

		activeGuardian, err := ga.GetActiveGuardian(uah)
		require.Nil(t, activeGuardian)
		require.Equal(t, process.ErrAccountHasNoGuardianSet, err)
	})
	t.Run("one pending guardian should return err", func(t *testing.T) {
		pendingGuardian := &guardians.Guardian{
			Address:         []byte("guardian address"),
			ActivationEpoch: currentEpoch + 1,
		}
		configuredGuardians := &guardians.Guardians{Slice: []*guardians.Guardian{pendingGuardian}}
		uah := &stateMocks.UserAccountStub{
			RetrieveValueFromDataTrieTrackerCalled: func(key []byte) ([]byte, error) {
				return ga.marshaller.Marshal(configuredGuardians)
			},
		}

		activeGuardian, err := ga.GetActiveGuardian(uah)
		require.Nil(t, activeGuardian)
		require.Equal(t, process.ErrAccountHasNoActiveGuardian, err)
	})
	t.Run("one active guardian should return the active", func(t *testing.T) {
		activeGuardian := &guardians.Guardian{
			Address:         []byte("guardian address"),
			ActivationEpoch: currentEpoch - 1,
		}
		configuredGuardians := &guardians.Guardians{Slice: []*guardians.Guardian{activeGuardian}}
		uah := &stateMocks.UserAccountStub{
			RetrieveValueFromDataTrieTrackerCalled: func(key []byte) ([]byte, error) {
				return ga.marshaller.Marshal(configuredGuardians)
			},
		}

		guardian, err := ga.GetActiveGuardian(uah)
		require.Equal(t, activeGuardian.Address, guardian)
		require.Nil(t, err)
	})
	t.Run("one active guardian and one pending new guardian", func(t *testing.T) {
		activeGuardian := &guardians.Guardian{
			Address:         []byte("guardian address"),
			ActivationEpoch: currentEpoch - 1,
		}
		pendingGuardian := &guardians.Guardian{
			Address:         []byte("pending guardian address"),
			ActivationEpoch: currentEpoch + 1,
		}

		configuredGuardians := &guardians.Guardians{Slice: []*guardians.Guardian{activeGuardian, pendingGuardian}}
		uah := &stateMocks.UserAccountStub{
			RetrieveValueFromDataTrieTrackerCalled: func(key []byte) ([]byte, error) {
				return ga.marshaller.Marshal(configuredGuardians)
			},
		}

		guardian, err := ga.GetActiveGuardian(uah)
		require.Equal(t, activeGuardian.Address, guardian)
		require.Nil(t, err)
	})
	t.Run("one active guardian and one disabled (old) guardian", func(t *testing.T) {
		activeGuardian := &guardians.Guardian{
			Address:         []byte("guardian address"),
			ActivationEpoch: currentEpoch - 1,
		}
		oldGuardian := &guardians.Guardian{
			Address:         []byte("pending guardian address"),
			ActivationEpoch: currentEpoch - 5,
		}

		configuredGuardians := &guardians.Guardians{Slice: []*guardians.Guardian{activeGuardian, oldGuardian}}
		uah := &stateMocks.UserAccountStub{
			RetrieveValueFromDataTrieTrackerCalled: func(key []byte) ([]byte, error) {
				return ga.marshaller.Marshal(configuredGuardians)
			},
		}

		guardian, err := ga.GetActiveGuardian(uah)
		require.Equal(t, activeGuardian.Address, guardian)
		require.Nil(t, err)
	})
}

func TestGuardedAccount_SetGuardian(t *testing.T) {
	currentEpoch := uint32(10)
	ga := createGuardedAccountWithEpoch(currentEpoch)
	g1 := &guardians.Guardian{
		Address:         []byte("guardian address 1"),
		ActivationEpoch: currentEpoch - 2,
	}
	g2 := &guardians.Guardian{
		Address:         []byte("guardian address 2"),
		ActivationEpoch: currentEpoch - 1,
	}
	newGuardianAddress := []byte("new guardian address")

	t.Run("invalid user account handler should err", func(t *testing.T) {
		err := ga.SetGuardian(nil, newGuardianAddress, g1.Address)
		require.Equal(t, process.ErrWrongTypeAssertion, err)
	})
	t.Run("transaction signed by current active guardian but instantSetGuardian returns error", func(t *testing.T) {
		configuredGuardians := &guardians.Guardians{Slice: []*guardians.Guardian{g1}}
		uah := &stateMocks.UserAccountStub{
			RetrieveValueFromDataTrieTrackerCalled: func(key []byte) ([]byte, error) {
				return ga.marshaller.Marshal(configuredGuardians)
			},
		}
		err := ga.SetGuardian(uah, newGuardianAddress, g2.Address)
		require.Equal(t, process.ErrTransactionAndAccountGuardianMismatch, err)
	})
	t.Run("instantly set guardian if tx signed by current active guardian", func(t *testing.T) {
		configuredGuardians := &guardians.Guardians{Slice: []*guardians.Guardian{g1}}
		newGuardian := &guardians.Guardian{
			Address:         newGuardianAddress,
			ActivationEpoch: currentEpoch,
		}
		expectedNewGuardians, _ := ga.marshaller.Marshal(&guardians.Guardians{Slice: []*guardians.Guardian{newGuardian}})

		uah := &stateMocks.UserAccountStub{
			RetrieveValueFromDataTrieTrackerCalled: func(key []byte) ([]byte, error) {
				return ga.marshaller.Marshal(configuredGuardians)
			},
			AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
				return &trie.DataTrieTrackerStub{
					SaveKeyValueCalled: func(key []byte, value []byte) error {
						require.Equal(t, guardianKey, key)
						require.Equal(t, expectedNewGuardians, value)
						return nil
					},
				}
			},
		}
		err := ga.SetGuardian(uah, newGuardianAddress, g1.Address)
		require.Nil(t, err)
	})
	t.Run("tx not signed by active guardian sets guardian with delay", func(t *testing.T) {
		configuredGuardians := &guardians.Guardians{Slice: []*guardians.Guardian{g1}}
		newGuardian := &guardians.Guardian{
			Address:         newGuardianAddress,
			ActivationEpoch: currentEpoch + ga.guardianActivationEpochsDelay,
		}
		expectedNewGuardians, _ := ga.marshaller.Marshal(&guardians.Guardians{Slice: []*guardians.Guardian{g1, newGuardian}})

		uah := &stateMocks.UserAccountStub{
			RetrieveValueFromDataTrieTrackerCalled: func(key []byte) ([]byte, error) {
				return ga.marshaller.Marshal(configuredGuardians)
			},
			AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
				return &trie.DataTrieTrackerStub{
					SaveKeyValueCalled: func(key []byte, value []byte) error {
						require.Equal(t, guardianKey, key)
						require.Equal(t, expectedNewGuardians, value)
						return nil
					},
				}
			},
		}
		err := ga.SetGuardian(uah, newGuardianAddress, nil)
		require.Nil(t, err)
	})
}

func TestGuardedAccount_HasActiveGuardian(t *testing.T) {
	currentEpoch := uint32(10)
	ga := createGuardedAccountWithEpoch(currentEpoch)

	t.Run("nil account type should return false", func(t *testing.T) {
		var uah *stateMocks.UserAccountStub
		require.False(t, ga.HasActiveGuardian(uah))
	})
	t.Run("getConfiguredGuardians with err should return false", func(t *testing.T) {
		expectedErr := errors.New("expected error")
		uah := &stateMocks.UserAccountStub{
			RetrieveValueFromDataTrieTrackerCalled: func(key []byte) ([]byte, error) {
				return nil, expectedErr
			},
		}
		require.False(t, ga.HasActiveGuardian(uah))
	})
	t.Run("no guardian should return false", func(t *testing.T) {
		configuredGuardians := &guardians.Guardians{}
		uah := &stateMocks.UserAccountStub{
			RetrieveValueFromDataTrieTrackerCalled: func(key []byte) ([]byte, error) {
				return ga.marshaller.Marshal(configuredGuardians)
			},
		}

		require.False(t, ga.HasActiveGuardian(uah))
	})
	t.Run("one pending guardian should return false", func(t *testing.T) {
		pendingGuardian := &guardians.Guardian{
			Address:         []byte("guardian address"),
			ActivationEpoch: currentEpoch + 1,
		}
		configuredGuardians := &guardians.Guardians{Slice: []*guardians.Guardian{pendingGuardian}}
		uah := &stateMocks.UserAccountStub{
			RetrieveValueFromDataTrieTrackerCalled: func(key []byte) ([]byte, error) {
				return ga.marshaller.Marshal(configuredGuardians)
			},
		}

		require.False(t, ga.HasActiveGuardian(uah))
	})
	t.Run("one active guardian should return true", func(t *testing.T) {
		activeGuardian := &guardians.Guardian{
			Address:         []byte("guardian address"),
			ActivationEpoch: currentEpoch - 1,
		}
		configuredGuardians := &guardians.Guardians{Slice: []*guardians.Guardian{activeGuardian}}
		uah := &stateMocks.UserAccountStub{
			RetrieveValueFromDataTrieTrackerCalled: func(key []byte) ([]byte, error) {
				return ga.marshaller.Marshal(configuredGuardians)
			},
		}

		require.True(t, ga.HasActiveGuardian(uah))
	})
	t.Run("one active guardian and one pending new guardian should return true", func(t *testing.T) {
		activeGuardian := &guardians.Guardian{
			Address:         []byte("guardian address"),
			ActivationEpoch: currentEpoch - 1,
		}
		pendingGuardian := &guardians.Guardian{
			Address:         []byte("pending guardian address"),
			ActivationEpoch: currentEpoch + 1,
		}

		configuredGuardians := &guardians.Guardians{Slice: []*guardians.Guardian{activeGuardian, pendingGuardian}}
		uah := &stateMocks.UserAccountStub{
			RetrieveValueFromDataTrieTrackerCalled: func(key []byte) ([]byte, error) {
				return ga.marshaller.Marshal(configuredGuardians)
			},
		}

		require.True(t, ga.HasActiveGuardian(uah))
	})
	t.Run("one active guardian and one disabled (old) guardian should return true", func(t *testing.T) {
		activeGuardian := &guardians.Guardian{
			Address:         []byte("guardian address"),
			ActivationEpoch: currentEpoch - 1,
		}
		oldGuardian := &guardians.Guardian{
			Address:         []byte("pending guardian address"),
			ActivationEpoch: currentEpoch - 5,
		}

		configuredGuardians := &guardians.Guardians{Slice: []*guardians.Guardian{activeGuardian, oldGuardian}}
		uah := &stateMocks.UserAccountStub{
			RetrieveValueFromDataTrieTrackerCalled: func(key []byte) ([]byte, error) {
				return ga.marshaller.Marshal(configuredGuardians)
			},
		}

		require.True(t, ga.HasActiveGuardian(uah))
	})
}

func TestGuardedAccount_HasPendingGuardian(t *testing.T) {
	currentEpoch := uint32(10)
	ga := createGuardedAccountWithEpoch(currentEpoch)

	t.Run("nil account type should return false", func(t *testing.T) {
		var uah *stateMocks.UserAccountStub
		require.False(t, ga.HasPendingGuardian(uah))
	})
	t.Run("getConfiguredGuardians with err should return false", func(t *testing.T) {
		expectedErr := errors.New("expected error")
		uah := &stateMocks.UserAccountStub{
			RetrieveValueFromDataTrieTrackerCalled: func(key []byte) ([]byte, error) {
				return nil, expectedErr
			},
		}
		require.False(t, ga.HasPendingGuardian(uah))
	})
	t.Run("no guardian should return false", func(t *testing.T) {
		configuredGuardians := &guardians.Guardians{}
		uah := &stateMocks.UserAccountStub{
			RetrieveValueFromDataTrieTrackerCalled: func(key []byte) ([]byte, error) {
				return ga.marshaller.Marshal(configuredGuardians)
			},
		}

		require.False(t, ga.HasPendingGuardian(uah))
	})
	t.Run("one pending guardian should return true", func(t *testing.T) {
		pendingGuardian := &guardians.Guardian{
			Address:         []byte("guardian address"),
			ActivationEpoch: currentEpoch + 1,
		}
		configuredGuardians := &guardians.Guardians{Slice: []*guardians.Guardian{pendingGuardian}}
		uah := &stateMocks.UserAccountStub{
			RetrieveValueFromDataTrieTrackerCalled: func(key []byte) ([]byte, error) {
				return ga.marshaller.Marshal(configuredGuardians)
			},
		}

		require.True(t, ga.HasPendingGuardian(uah))
	})
	t.Run("one active guardian should return false", func(t *testing.T) {
		activeGuardian := &guardians.Guardian{
			Address:         []byte("guardian address"),
			ActivationEpoch: currentEpoch - 1,
		}
		configuredGuardians := &guardians.Guardians{Slice: []*guardians.Guardian{activeGuardian}}
		uah := &stateMocks.UserAccountStub{
			RetrieveValueFromDataTrieTrackerCalled: func(key []byte) ([]byte, error) {
				return ga.marshaller.Marshal(configuredGuardians)
			},
		}

		require.False(t, ga.HasPendingGuardian(uah))
	})
	t.Run("one active guardian and one pending new guardian should return true", func(t *testing.T) {
		activeGuardian := &guardians.Guardian{
			Address:         []byte("guardian address"),
			ActivationEpoch: currentEpoch - 1,
		}
		pendingGuardian := &guardians.Guardian{
			Address:         []byte("pending guardian address"),
			ActivationEpoch: currentEpoch + 1,
		}

		configuredGuardians := &guardians.Guardians{Slice: []*guardians.Guardian{activeGuardian, pendingGuardian}}
		uah := &stateMocks.UserAccountStub{
			RetrieveValueFromDataTrieTrackerCalled: func(key []byte) ([]byte, error) {
				return ga.marshaller.Marshal(configuredGuardians)
			},
		}

		require.True(t, ga.HasPendingGuardian(uah))
	})
	t.Run("one active guardian and one disabled (old) guardian should return false", func(t *testing.T) {
		activeGuardian := &guardians.Guardian{
			Address:         []byte("guardian address"),
			ActivationEpoch: currentEpoch - 1,
		}
		oldGuardian := &guardians.Guardian{
			Address:         []byte("pending guardian address"),
			ActivationEpoch: currentEpoch - 5,
		}

		configuredGuardians := &guardians.Guardians{Slice: []*guardians.Guardian{activeGuardian, oldGuardian}}
		uah := &stateMocks.UserAccountStub{
			RetrieveValueFromDataTrieTrackerCalled: func(key []byte) ([]byte, error) {
				return ga.marshaller.Marshal(configuredGuardians)
			},
		}

		require.False(t, ga.HasPendingGuardian(uah))
	})
}

func TestGuardedAccount_EpochConfirmed(t *testing.T) {
	ga := createGuardedAccountWithEpoch(0)
	ga.EpochConfirmed(1, 0)
	require.Equal(t, uint32(1), ga.currentEpoch)

	ga.EpochConfirmed(111, 0)
	require.Equal(t, uint32(111), ga.currentEpoch)
}

func TestGuardedAccount_IsInterfaceNil(t *testing.T) {
	var gah process.GuardedAccountHandler
	require.True(t, check.IfNil(gah))

	var ga *guardedAccount
	require.True(t, check.IfNil(ga))

	ga, _ = NewGuardedAccount(&testscommon.MarshalizerMock{}, &epochNotifier.EpochNotifierStub{}, 10)
	require.False(t, check.IfNil(ga))
}

func createGuardedAccountWithEpoch(epoch uint32) *guardedAccount {
	marshaller := &testscommon.MarshalizerMock{}
	en := &epochNotifier.EpochNotifierStub{
		RegisterNotifyHandlerCalled: func(handler vmcommon.EpochSubscriberHandler) {
			handler.EpochConfirmed(epoch, 0)
		},
	}

	ga, _ := NewGuardedAccount(marshaller, en, 10)
	return ga
}
