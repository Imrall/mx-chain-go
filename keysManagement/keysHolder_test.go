package keysManagement

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-go-crypto"
	"github.com/ElrondNetwork/elrond-go/keysManagement/mock"
	"github.com/ElrondNetwork/elrond-go/testscommon/cryptoMocks"
	"github.com/stretchr/testify/assert"
)

var (
	p2pPrivateKey = []byte("p2p private key")
	pid           = core.PeerID("pid")
	skBytes0      = []byte("private key 0")
	skBytes1      = []byte("private key 1")
	pkBytes0      = []byte("public key 0")
	pkBytes1      = []byte("public key 1")
)

func createMockArgsVirtualPeersHolder() ArgsVirtualPeersHolder {
	return ArgsVirtualPeersHolder{
		KeyGenerator: createMockKeyGenerator(),
		P2PIdentityGenerator: &mock.IdentityGeneratorStub{
			CreateRandomP2PIdentityCalled: func() ([]byte, core.PeerID, error) {
				return p2pPrivateKey, pid, nil
			},
		},
		IsMainMachine:                    true,
		MaxRoundsWithoutReceivedMessages: 1,
	}
}

func createMockKeyGenerator() crypto.KeyGenerator {
	return &cryptoMocks.KeyGenStub{
		PrivateKeyFromByteArrayStub: func(b []byte) (crypto.PrivateKey, error) {
			return &cryptoMocks.PrivateKeyStub{
				GeneratePublicStub: func() crypto.PublicKey {
					pk := &cryptoMocks.PublicKeyStub{
						ToByteArrayStub: func() ([]byte, error) {
							pkBytes := bytes.Replace(b, []byte("private"), []byte("public"), -1)

							return pkBytes, nil
						},
					}

					return pk
				},
				ToByteArrayStub: func() ([]byte, error) {
					return b, nil
				},
			}, nil
		},
	}
}

func testManagedKeys(tb testing.TB, result map[string]crypto.PrivateKey, pkBytes ...[]byte) {
	assert.Equal(tb, len(pkBytes), len(result))

	for _, pk := range pkBytes {
		_, found := result[string(pk)]
		assert.True(tb, found)
	}
}

func TestNewVirtualPeersHolder(t *testing.T) {
	t.Parallel()

	t.Run("nil key generator should error", func(t *testing.T) {
		t.Parallel()

		args := createMockArgsVirtualPeersHolder()
		args.KeyGenerator = nil
		holder, err := NewVirtualPeersHolder(args)

		assert.Equal(t, errNilKeyGenerator, err)
		assert.True(t, check.IfNil(holder))
	})
	t.Run("nil identity generator should error", func(t *testing.T) {
		t.Parallel()

		args := createMockArgsVirtualPeersHolder()
		args.P2PIdentityGenerator = nil
		holder, err := NewVirtualPeersHolder(args)

		assert.Equal(t, errNilP2PIdentityGenerator, err)
		assert.True(t, check.IfNil(holder))
	})
	t.Run("invalid MaxRoundsWithoutReceivedMessages should error", func(t *testing.T) {
		t.Parallel()

		args := createMockArgsVirtualPeersHolder()
		args.MaxRoundsWithoutReceivedMessages = 0
		holder, err := NewVirtualPeersHolder(args)

		assert.True(t, errors.Is(err, errInvalidValue))
		assert.True(t, strings.Contains(err.Error(), "MaxRoundsWithoutReceivedMessages"))
		assert.True(t, check.IfNil(holder))
	})
	t.Run("valid arguments should work", func(t *testing.T) {
		t.Parallel()

		args := createMockArgsVirtualPeersHolder()
		holder, err := NewVirtualPeersHolder(args)

		assert.Nil(t, err)
		assert.False(t, check.IfNil(holder))
	})
}

func TestVirtualPeersHolder_AddVirtualPeer(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("expected error")
	t.Run("private key from byte array errors", func(t *testing.T) {
		args := createMockArgsVirtualPeersHolder()
		args.KeyGenerator = &cryptoMocks.KeyGenStub{
			PrivateKeyFromByteArrayStub: func(b []byte) (crypto.PrivateKey, error) {
				return nil, expectedErr
			},
		}

		holder, _ := NewVirtualPeersHolder(args)
		err := holder.AddVirtualPeer([]byte("private key"))

		assert.True(t, errors.Is(err, expectedErr))
	})
	t.Run("public key from byte array errors", func(t *testing.T) {
		args := createMockArgsVirtualPeersHolder()

		pk := &cryptoMocks.PublicKeyStub{
			ToByteArrayStub: func() ([]byte, error) {
				return nil, expectedErr
			},
		}

		sk := &cryptoMocks.PrivateKeyStub{
			GeneratePublicStub: func() crypto.PublicKey {
				return pk
			},
		}

		args.KeyGenerator = &cryptoMocks.KeyGenStub{
			PrivateKeyFromByteArrayStub: func(b []byte) (crypto.PrivateKey, error) {
				return sk, nil
			},
		}

		holder, _ := NewVirtualPeersHolder(args)
		err := holder.AddVirtualPeer([]byte("private key"))

		assert.True(t, errors.Is(err, expectedErr))
	})
	t.Run("identity creation errors should errors", func(t *testing.T) {
		args := createMockArgsVirtualPeersHolder()
		args.P2PIdentityGenerator = &mock.IdentityGeneratorStub{
			CreateRandomP2PIdentityCalled: func() ([]byte, core.PeerID, error) {
				return nil, "", expectedErr
			},
		}

		holder, _ := NewVirtualPeersHolder(args)
		err := holder.AddVirtualPeer([]byte("private key"))

		assert.True(t, errors.Is(err, expectedErr))
	})
	t.Run("should work for a new pk", func(t *testing.T) {
		args := createMockArgsVirtualPeersHolder()

		holder, _ := NewVirtualPeersHolder(args)
		err := holder.AddVirtualPeer(skBytes0)
		assert.Nil(t, err)

		pInfo := holder.getPeerInfo(pkBytes0)
		assert.NotNil(t, pInfo)
		assert.Equal(t, pid, pInfo.pid)
		assert.Equal(t, p2pPrivateKey, pInfo.p2pPrivateKeyBytes)
		skBytesRecovered, _ := pInfo.privateKey.ToByteArray()
		assert.Equal(t, skBytes0, skBytesRecovered)
		assert.Equal(t, 10, len(pInfo.machineID))
	})
	t.Run("should error when trying to add the same pk", func(t *testing.T) {
		args := createMockArgsVirtualPeersHolder()

		holder, _ := NewVirtualPeersHolder(args)
		err := holder.AddVirtualPeer(skBytes0)
		assert.Nil(t, err)

		err = holder.AddVirtualPeer(skBytes0)
		assert.True(t, errors.Is(err, errDuplicatedKey))
	})
	t.Run("should work for 2 new public keys", func(t *testing.T) {
		args := createMockArgsVirtualPeersHolder()

		holder, _ := NewVirtualPeersHolder(args)
		err := holder.AddVirtualPeer(skBytes0)
		assert.Nil(t, err)

		err = holder.AddVirtualPeer(skBytes1)
		assert.Nil(t, err)

		pInfo0 := holder.getPeerInfo(pkBytes0)
		assert.NotNil(t, pInfo0)

		pInfo1 := holder.getPeerInfo(pkBytes1)
		assert.NotNil(t, pInfo1)

		assert.Equal(t, p2pPrivateKey, pInfo0.p2pPrivateKeyBytes)
		assert.Equal(t, p2pPrivateKey, pInfo1.p2pPrivateKeyBytes)

		assert.Equal(t, pid, pInfo0.pid)
		assert.Equal(t, pid, pInfo1.pid)

		skBytesRecovered0, _ := pInfo0.privateKey.ToByteArray()
		assert.Equal(t, skBytes0, skBytesRecovered0)

		skBytesRecovered1, _ := pInfo1.privateKey.ToByteArray()
		assert.Equal(t, skBytes1, skBytesRecovered1)

		assert.NotEqual(t, pInfo0.machineID, pInfo1.machineID)
		assert.Equal(t, 10, len(pInfo0.machineID))
		assert.Equal(t, 10, len(pInfo1.machineID))
	})
}

func TestVirtualPeersHolder_GetPrivateKey(t *testing.T) {
	t.Parallel()

	args := createMockArgsVirtualPeersHolder()

	holder, _ := NewVirtualPeersHolder(args)
	_ = holder.AddVirtualPeer(skBytes0)
	t.Run("public key not added should error", func(t *testing.T) {
		skRecovered, err := holder.GetPrivateKey(pkBytes1)
		assert.Nil(t, skRecovered)
		assert.True(t, errors.Is(err, errMissingPublicKeyDefinition))
	})
	t.Run("public key exists should return the private key", func(t *testing.T) {
		skRecovered, err := holder.GetPrivateKey(pkBytes0)
		assert.Nil(t, err)

		skBytesRecovered, _ := skRecovered.ToByteArray()
		assert.Equal(t, skBytes0, skBytesRecovered)
	})
}

func TestVirtualPeersHolder_GetP2PIdentity(t *testing.T) {
	t.Parallel()

	args := createMockArgsVirtualPeersHolder()

	holder, _ := NewVirtualPeersHolder(args)
	_ = holder.AddVirtualPeer(skBytes0)
	t.Run("public key not added should error", func(t *testing.T) {
		p2pPrivateKeyRecovered, pidRecovered, err := holder.GetP2PIdentity(pkBytes1)
		assert.Nil(t, p2pPrivateKeyRecovered)
		assert.Empty(t, pidRecovered)
		assert.True(t, errors.Is(err, errMissingPublicKeyDefinition))
	})
	t.Run("public key exists should return the p2p identity", func(t *testing.T) {
		p2pPrivateKeyRecovered, pidRecovered, err := holder.GetP2PIdentity(pkBytes0)
		assert.Nil(t, err)
		assert.Equal(t, p2pPrivateKey, p2pPrivateKeyRecovered)
		assert.Equal(t, pid, pidRecovered)
	})
}

func TestVirtualPeersHolder_GetMachineID(t *testing.T) {
	t.Parallel()

	args := createMockArgsVirtualPeersHolder()
	holder, _ := NewVirtualPeersHolder(args)
	_ = holder.AddVirtualPeer(skBytes0)
	t.Run("public key not added should error", func(t *testing.T) {
		machineIDRecovered, err := holder.GetMachineID(pkBytes1)
		assert.Empty(t, machineIDRecovered)
		assert.True(t, errors.Is(err, errMissingPublicKeyDefinition))
	})
	t.Run("public key exists should return machine ID", func(t *testing.T) {
		machineIDRecovered, err := holder.GetMachineID(pkBytes0)
		assert.Nil(t, err)
		assert.Equal(t, 10, len(machineIDRecovered))
	})
}

func TestVirtualPeersHolder_IncrementRoundsWithoutReceivedMessages(t *testing.T) {
	t.Parallel()

	t.Run("is main machine should ignore the call", func(t *testing.T) {
		args := createMockArgsVirtualPeersHolder()
		args.IsMainMachine = true
		holder, _ := NewVirtualPeersHolder(args)
		_ = holder.AddVirtualPeer(skBytes0)

		t.Run("missing public key", func(t *testing.T) {
			err := holder.IncrementRoundsWithoutReceivedMessages(pkBytes1)
			assert.Nil(t, err)
		})
		t.Run("existing public key", func(t *testing.T) {
			err := holder.IncrementRoundsWithoutReceivedMessages(pkBytes0)
			assert.Nil(t, err)

			pInfoRecovered := holder.getPeerInfo(pkBytes0)
			assert.Zero(t, pInfoRecovered.getRoundsWithoutReceivedMessages())
		})
	})
	t.Run("is secondary machine should increment, if existing", func(t *testing.T) {
		args := createMockArgsVirtualPeersHolder()
		args.IsMainMachine = false
		holder, _ := NewVirtualPeersHolder(args)
		_ = holder.AddVirtualPeer(skBytes0)

		t.Run("missing public key should error", func(t *testing.T) {
			err := holder.IncrementRoundsWithoutReceivedMessages(pkBytes1)
			assert.True(t, errors.Is(err, errMissingPublicKeyDefinition))
		})
		t.Run("existing public key should increment", func(t *testing.T) {
			pInfoRecovered := holder.getPeerInfo(pkBytes0)
			assert.Zero(t, pInfoRecovered.getRoundsWithoutReceivedMessages())

			err := holder.IncrementRoundsWithoutReceivedMessages(pkBytes0)
			assert.Nil(t, err)

			pInfoRecovered = holder.getPeerInfo(pkBytes0)
			assert.Equal(t, 1, pInfoRecovered.getRoundsWithoutReceivedMessages())

			err = holder.IncrementRoundsWithoutReceivedMessages(pkBytes0)
			assert.Nil(t, err)

			pInfoRecovered = holder.getPeerInfo(pkBytes0)
			assert.Equal(t, 2, pInfoRecovered.getRoundsWithoutReceivedMessages())
		})
	})
}

func TestVirtualPeersHolder_ResetRoundsWithoutReceivedMessages(t *testing.T) {
	t.Parallel()

	t.Run("is main machine should ignore the call", func(t *testing.T) {
		args := createMockArgsVirtualPeersHolder()
		args.IsMainMachine = true
		holder, _ := NewVirtualPeersHolder(args)
		_ = holder.AddVirtualPeer(skBytes0)

		t.Run("missing public key", func(t *testing.T) {
			err := holder.ResetRoundsWithoutReceivedMessages(pkBytes1)
			assert.Nil(t, err)
		})
		t.Run("existing public key", func(t *testing.T) {
			err := holder.ResetRoundsWithoutReceivedMessages(pkBytes0)
			assert.Nil(t, err)

			pInfoRecovered := holder.getPeerInfo(pkBytes0)
			assert.Zero(t, pInfoRecovered.getRoundsWithoutReceivedMessages())
		})
	})
	t.Run("is secondary machine should reset, if existing", func(t *testing.T) {
		args := createMockArgsVirtualPeersHolder()
		args.IsMainMachine = false
		holder, _ := NewVirtualPeersHolder(args)
		_ = holder.AddVirtualPeer(skBytes0)

		t.Run("missing public key should error", func(t *testing.T) {
			err := holder.ResetRoundsWithoutReceivedMessages(pkBytes1)
			assert.True(t, errors.Is(err, errMissingPublicKeyDefinition))
		})
		t.Run("existing public key should reset", func(t *testing.T) {
			pInfoRecovered := holder.getPeerInfo(pkBytes0)
			assert.Zero(t, pInfoRecovered.getRoundsWithoutReceivedMessages())

			_ = holder.IncrementRoundsWithoutReceivedMessages(pkBytes0)
			pInfoRecovered = holder.getPeerInfo(pkBytes0)
			assert.Equal(t, 1, pInfoRecovered.getRoundsWithoutReceivedMessages())

			err := holder.ResetRoundsWithoutReceivedMessages(pkBytes0)
			assert.Nil(t, err)

			pInfoRecovered = holder.getPeerInfo(pkBytes0)
			assert.Equal(t, 0, pInfoRecovered.getRoundsWithoutReceivedMessages())

			_ = holder.IncrementRoundsWithoutReceivedMessages(pkBytes0)
			_ = holder.IncrementRoundsWithoutReceivedMessages(pkBytes0)
			_ = holder.IncrementRoundsWithoutReceivedMessages(pkBytes0)
			pInfoRecovered = holder.getPeerInfo(pkBytes0)
			assert.Equal(t, 3, pInfoRecovered.getRoundsWithoutReceivedMessages())

			err = holder.ResetRoundsWithoutReceivedMessages(pkBytes0)
			assert.Nil(t, err)

			pInfoRecovered = holder.getPeerInfo(pkBytes0)
			assert.Equal(t, 0, pInfoRecovered.getRoundsWithoutReceivedMessages())
		})
	})
}

func TestVirtualPeersHolder_GetAllManagedKeys(t *testing.T) {
	t.Parallel()

	t.Run("main machine should return all keys, always", func(t *testing.T) {
		args := createMockArgsVirtualPeersHolder()
		args.IsMainMachine = true
		holder, _ := NewVirtualPeersHolder(args)
		_ = holder.AddVirtualPeer(skBytes0)
		_ = holder.AddVirtualPeer(skBytes1)

		for i := 0; i < 10; i++ {
			_ = holder.IncrementRoundsWithoutReceivedMessages(pkBytes0)
		}

		result := holder.GetAllManagedKeys()
		testManagedKeys(t, result, pkBytes0, pkBytes1)
	})
	t.Run("is secondary machine should return managed keys", func(t *testing.T) {
		args := createMockArgsVirtualPeersHolder()
		args.IsMainMachine = false
		args.MaxRoundsWithoutReceivedMessages = 2
		holder, _ := NewVirtualPeersHolder(args)
		_ = holder.AddVirtualPeer(skBytes0)
		_ = holder.AddVirtualPeer(skBytes1)

		t.Run("MaxRoundsWithoutReceivedMessages not reached should return none", func(t *testing.T) {
			result := holder.GetAllManagedKeys()
			testManagedKeys(t, result)

			_ = holder.IncrementRoundsWithoutReceivedMessages(pkBytes0)

			result = holder.GetAllManagedKeys()
			testManagedKeys(t, result)
		})
		t.Run("MaxRoundsWithoutReceivedMessages reached, should return failed pk", func(t *testing.T) {
			_ = holder.IncrementRoundsWithoutReceivedMessages(pkBytes0)

			result := holder.GetAllManagedKeys()
			testManagedKeys(t, result, pkBytes0)

			_ = holder.IncrementRoundsWithoutReceivedMessages(pkBytes0)
			result = holder.GetAllManagedKeys()
			testManagedKeys(t, result, pkBytes0)
		})
	})
}

func TestVirtualPeersHolder_IsKeyManagedByCurrentNode(t *testing.T) {
	t.Parallel()

	t.Run("main machine", func(t *testing.T) {
		args := createMockArgsVirtualPeersHolder()
		args.IsMainMachine = true
		holder, _ := NewVirtualPeersHolder(args)
		_ = holder.AddVirtualPeer(skBytes0)

		t.Run("foreign public key should return false", func(t *testing.T) {
			isManaged := holder.IsKeyManagedByCurrentNode(pkBytes1)
			assert.False(t, isManaged)
		})
		t.Run("managed key should return true", func(t *testing.T) {
			isManaged := holder.IsKeyManagedByCurrentNode(pkBytes0)
			assert.True(t, isManaged)
		})
	})
	t.Run("secondary machine", func(t *testing.T) {
		args := createMockArgsVirtualPeersHolder()
		args.IsMainMachine = false
		args.MaxRoundsWithoutReceivedMessages = 2
		holder, _ := NewVirtualPeersHolder(args)
		_ = holder.AddVirtualPeer(skBytes0)

		t.Run("foreign public key should return false", func(t *testing.T) {
			isManaged := holder.IsKeyManagedByCurrentNode(pkBytes1)
			assert.False(t, isManaged)
		})
		t.Run("managed key should return false while MaxRoundsWithoutReceivedMessages is not reached", func(t *testing.T) {
			isManaged := holder.IsKeyManagedByCurrentNode(pkBytes0)
			assert.False(t, isManaged)

			_ = holder.IncrementRoundsWithoutReceivedMessages(pkBytes0)
			isManaged = holder.IsKeyManagedByCurrentNode(pkBytes0)
			assert.False(t, isManaged)

			_ = holder.IncrementRoundsWithoutReceivedMessages(pkBytes0)
			isManaged = holder.IsKeyManagedByCurrentNode(pkBytes0)
			assert.True(t, isManaged)

			_ = holder.ResetRoundsWithoutReceivedMessages(pkBytes0)
			isManaged = holder.IsKeyManagedByCurrentNode(pkBytes0)
			assert.False(t, isManaged)
		})
	})
}

func TestVirtualPeersHolder_IsKeyRegistered(t *testing.T) {
	t.Parallel()

	t.Run("main machine", func(t *testing.T) {
		args := createMockArgsVirtualPeersHolder()
		args.IsMainMachine = true
		holder, _ := NewVirtualPeersHolder(args)
		_ = holder.AddVirtualPeer(skBytes0)

		t.Run("foreign public key should return false", func(t *testing.T) {
			isManaged := holder.IsKeyRegistered(pkBytes1)
			assert.False(t, isManaged)
		})
		t.Run("registered key should return true", func(t *testing.T) {
			isManaged := holder.IsKeyRegistered(pkBytes0)
			assert.True(t, isManaged)
		})
	})
	t.Run("secondary machine", func(t *testing.T) {
		args := createMockArgsVirtualPeersHolder()
		args.IsMainMachine = false
		holder, _ := NewVirtualPeersHolder(args)
		_ = holder.AddVirtualPeer(skBytes0)

		t.Run("foreign public key should return false", func(t *testing.T) {
			isManaged := holder.IsKeyRegistered(pkBytes1)
			assert.False(t, isManaged)
		})
		t.Run("registered key should return true", func(t *testing.T) {
			isManaged := holder.IsKeyRegistered(pkBytes0)
			assert.True(t, isManaged)
		})
	})
}

func TestVirtualPeersHolder_IsPidManagedByCurrentNode(t *testing.T) {
	t.Parallel()

	args := createMockArgsVirtualPeersHolder()
	args.IsMainMachine = true
	holder, _ := NewVirtualPeersHolder(args)

	t.Run("empty holder should return false", func(t *testing.T) {
		isManaged := holder.IsPidManagedByCurrentNode(pid)
		assert.False(t, isManaged)
	})

	_ = holder.AddVirtualPeer(skBytes0)

	t.Run("pid not managed by current should return false", func(t *testing.T) {
		isManaged := holder.IsPidManagedByCurrentNode("other pid")
		assert.False(t, isManaged)
	})
	t.Run("pid managed by current should return true", func(t *testing.T) {
		isManaged := holder.IsPidManagedByCurrentNode(pid)
		assert.True(t, isManaged)
	})
}

func TestVirtualPeersHolder_ParallelOperationsShouldNotPanic(t *testing.T) {
	defer func() {
		r := recover()
		if r != nil {
			assert.Fail(t, fmt.Sprintf("should have not panicked %v", r))
		}
	}()

	args := createMockArgsVirtualPeersHolder()
	holder, _ := NewVirtualPeersHolder(args)

	numOperations := 1000
	wg := sync.WaitGroup{}
	wg.Add(numOperations)
	for i := 0; i < numOperations; i++ {
		go func(idOperation int) {
			time.Sleep(time.Millisecond * 10) // increase the chance of concurrent operations

			switch idOperation {
			case 0:
				randomBytes := make([]byte, 32)
				_, _ = rand.Read(randomBytes)
				_ = holder.AddVirtualPeer(randomBytes)
			case 1:
				_, _ = holder.GetMachineID(pkBytes1)
			case 2:
				_, _, _ = holder.GetP2PIdentity(pkBytes1)
			case 3:
				_, _ = holder.GetPrivateKey(pkBytes1)
			case 4:
				_ = holder.IncrementRoundsWithoutReceivedMessages(pkBytes0)
			case 5:
				_ = holder.ResetRoundsWithoutReceivedMessages(pkBytes0)
			case 6:
				_ = holder.GetAllManagedKeys()
			case 7:
				_ = holder.IsKeyManagedByCurrentNode(pkBytes0)
			case 8:
				_ = holder.IsKeyRegistered(pkBytes0)
			case 9:
				_ = holder.IsPidManagedByCurrentNode("pid")
			}

			wg.Done()
		}(i % 10)
	}

	wg.Wait()
}
