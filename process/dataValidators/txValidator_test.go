package dataValidators_test

import (
	"errors"
	"math/big"
	"strconv"
	"strings"
	"testing"

	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-go-core/data"
	"github.com/ElrondNetwork/elrond-go-core/data/receipt"
	"github.com/ElrondNetwork/elrond-go-core/data/transaction"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/dataValidators"
	"github.com/ElrondNetwork/elrond-go/process/mock"
	"github.com/ElrondNetwork/elrond-go/state"
	"github.com/ElrondNetwork/elrond-go/testscommon"
	"github.com/ElrondNetwork/elrond-go/testscommon/guardianMocks"
	stateMock "github.com/ElrondNetwork/elrond-go/testscommon/state"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getAccAdapter(nonce uint64, balance *big.Int) *stateMock.AccountsStub {
	accDB := &stateMock.AccountsStub{}
	accDB.GetExistingAccountCalled = func(address []byte) (handler vmcommon.AccountHandler, e error) {
		acc, _ := state.NewUserAccount(address)
		acc.Nonce = nonce
		acc.Balance = balance

		return acc, nil
	}

	return accDB
}

func createMockCoordinator(identifierPrefix string, currentShardID uint32) *mock.CoordinatorStub {
	return &mock.CoordinatorStub{
		CommunicationIdentifierCalled: func(destShardID uint32) string {
			return identifierPrefix + strconv.Itoa(int(destShardID))
		},
		SelfIdCalled: func() uint32 {
			return currentShardID
		},
	}
}

func getInterceptedTxHandler(
	sndShardId uint32,
	rcvShardId uint32,
	nonce uint64,
	sndAddr []byte,
	fee *big.Int,
) process.InterceptedTransactionHandler {
	return &mock.InterceptedTxHandlerStub{
		SenderShardIdCalled: func() uint32 {
			return sndShardId
		},
		ReceiverShardIdCalled: func() uint32 {
			return rcvShardId
		},
		NonceCalled: func() uint64 {
			return nonce
		},
		SenderAddressCalled: func() []byte {
			return sndAddr
		},
		FeeCalled: func() *big.Int {
			return fee
		},
		TransactionCalled: func() data.TransactionHandler {
			return &transaction.Transaction{}
		},
	}
}

func TestNewTxValidator_NilAccountsShouldErr(t *testing.T) {
	t.Parallel()

	shardCoordinator := createMockCoordinator("_", 0)
	maxNonceDeltaAllowed := 100
	txValidator, err := dataValidators.NewTxValidator(
		nil,
		shardCoordinator,
		&testscommon.WhiteListHandlerStub{},
		mock.NewPubkeyConverterMock(32),
		&guardianMocks.GuardianSigVerifierStub{},
		&testscommon.TxVersionCheckerStub{},
		maxNonceDeltaAllowed,
	)

	assert.Nil(t, txValidator)
	assert.Equal(t, process.ErrNilAccountsAdapter, err)
}

func TestNewTxValidator_NilShardCoordinatorShouldErr(t *testing.T) {
	t.Parallel()

	adb := getAccAdapter(0, big.NewInt(0))
	maxNonceDeltaAllowed := 100
	txValidator, err := dataValidators.NewTxValidator(
		adb,
		nil,
		&testscommon.WhiteListHandlerStub{},
		mock.NewPubkeyConverterMock(32),
		&guardianMocks.GuardianSigVerifierStub{},
		&testscommon.TxVersionCheckerStub{},
		maxNonceDeltaAllowed,
	)

	assert.Nil(t, txValidator)
	assert.Equal(t, process.ErrNilShardCoordinator, err)
}

func TestTxValidator_NewValidatorNilWhiteListHandlerShouldErr(t *testing.T) {
	t.Parallel()

	adb := getAccAdapter(0, big.NewInt(0))
	maxNonceDeltaAllowed := 100
	shardCoordinator := createMockCoordinator("_", 0)
	txValidator, err := dataValidators.NewTxValidator(
		adb,
		shardCoordinator,
		nil,
		mock.NewPubkeyConverterMock(32),
		&guardianMocks.GuardianSigVerifierStub{},
		&testscommon.TxVersionCheckerStub{},
		maxNonceDeltaAllowed,
	)

	assert.Nil(t, txValidator)
	assert.Equal(t, process.ErrNilWhiteListHandler, err)
}

func TestNewTxValidator_NilPubkeyConverterShouldErr(t *testing.T) {
	t.Parallel()

	adb := getAccAdapter(0, big.NewInt(0))
	maxNonceDeltaAllowed := 100
	shardCoordinator := createMockCoordinator("_", 0)
	txValidator, err := dataValidators.NewTxValidator(
		adb,
		shardCoordinator,
		&testscommon.WhiteListHandlerStub{},
		nil,
		&guardianMocks.GuardianSigVerifierStub{},
		&testscommon.TxVersionCheckerStub{},
		maxNonceDeltaAllowed,
	)

	assert.Nil(t, txValidator)
	assert.True(t, errors.Is(err, process.ErrNilPubkeyConverter))
}

func TestNewTxValidator_NilGuardianSigVerifierShouldErr(t *testing.T) {
	t.Parallel()

	adb := getAccAdapter(0, big.NewInt(0))
	shardCoordinator := createMockCoordinator("_", 0)
	maxNonceDeltaAllowed := 100
	txValidator, err := dataValidators.NewTxValidator(
		adb,
		shardCoordinator,
		&testscommon.WhiteListHandlerStub{},
		mock.NewPubkeyConverterMock(32),
		nil,
		&testscommon.TxVersionCheckerStub{},
		maxNonceDeltaAllowed,
	)
	assert.Nil(t, txValidator)
	assert.True(t, errors.Is(err, process.ErrNilGuardianSigVerifier))
}

func TestNewTxValidator_NilTxVersionCheckerShouldErr(t *testing.T) {
	t.Parallel()

	adb := getAccAdapter(0, big.NewInt(0))
	shardCoordinator := createMockCoordinator("_", 0)
	maxNonceDeltaAllowed := 100
	txValidator, err := dataValidators.NewTxValidator(
		adb,
		shardCoordinator,
		&testscommon.WhiteListHandlerStub{},
		mock.NewPubkeyConverterMock(32),
		&guardianMocks.GuardianSigVerifierStub{},
		nil,
		maxNonceDeltaAllowed,
	)
	assert.Nil(t, txValidator)
	assert.True(t, errors.Is(err, process.ErrNilTransactionVersionChecker))
}

func TestNewTxValidator_ShouldWork(t *testing.T) {
	t.Parallel()

	adb := getAccAdapter(0, big.NewInt(0))
	shardCoordinator := createMockCoordinator("_", 0)
	maxNonceDeltaAllowed := 100
	txValidator, err := dataValidators.NewTxValidator(
		adb,
		shardCoordinator,
		&testscommon.WhiteListHandlerStub{},
		mock.NewPubkeyConverterMock(32),
		&guardianMocks.GuardianSigVerifierStub{},
		&testscommon.TxVersionCheckerStub{},
		maxNonceDeltaAllowed,
	)

	assert.Nil(t, err)
	assert.NotNil(t, txValidator)

	result := txValidator.IsInterfaceNil()
	assert.Equal(t, false, result)
}

func TestTxValidator_CheckTxValidityTxCrossShardShouldWork(t *testing.T) {
	t.Parallel()

	adb := getAccAdapter(1, big.NewInt(0))
	currentShard := uint32(0)
	shardCoordinator := createMockCoordinator("_", currentShard)
	maxNonceDeltaAllowed := 100
	txValidator, err := dataValidators.NewTxValidator(
		adb,
		shardCoordinator,
		&testscommon.WhiteListHandlerStub{},
		mock.NewPubkeyConverterMock(32),
		&guardianMocks.GuardianSigVerifierStub{},
		&testscommon.TxVersionCheckerStub{},
		maxNonceDeltaAllowed,
	)
	assert.Nil(t, err)

	addressMock := []byte("address")
	txValidatorHandler := getInterceptedTxHandler(currentShard+1, currentShard, 1, addressMock, big.NewInt(0))

	result := txValidator.CheckTxValidity(txValidatorHandler)
	assert.Nil(t, result)
}

func TestTxValidator_CheckTxValidityAccountNonceIsGreaterThanTxNonceShouldReturnFalse(t *testing.T) {
	t.Parallel()

	accountNonce := uint64(100)
	txNonce := uint64(0)

	adb := getAccAdapter(accountNonce, big.NewInt(0))
	shardCoordinator := createMockCoordinator("_", 0)
	maxNonceDeltaAllowed := 100
	txValidator, err := dataValidators.NewTxValidator(
		adb,
		shardCoordinator,
		&testscommon.WhiteListHandlerStub{},
		mock.NewPubkeyConverterMock(32),
		&guardianMocks.GuardianSigVerifierStub{},
		&testscommon.TxVersionCheckerStub{},
		maxNonceDeltaAllowed,
	)
	assert.Nil(t, err)

	addressMock := []byte("address")
	currentShard := uint32(0)
	txValidatorHandler := getInterceptedTxHandler(currentShard, currentShard, txNonce, addressMock, big.NewInt(0))

	result := txValidator.CheckTxValidity(txValidatorHandler)
	assert.True(t, errors.Is(result, process.ErrWrongTransaction))
}

func TestTxValidator_CheckTxValidityTxNonceIsTooHigh(t *testing.T) {
	t.Parallel()

	accountNonce := uint64(100)
	maxNonceDeltaAllowed := 100
	txNonce := accountNonce + uint64(maxNonceDeltaAllowed) + 1

	adb := getAccAdapter(accountNonce, big.NewInt(0))
	shardCoordinator := createMockCoordinator("_", 0)
	txValidator, err := dataValidators.NewTxValidator(
		adb,
		shardCoordinator,
		&testscommon.WhiteListHandlerStub{},
		mock.NewPubkeyConverterMock(32),
		&guardianMocks.GuardianSigVerifierStub{},
		&testscommon.TxVersionCheckerStub{},
		maxNonceDeltaAllowed,
	)
	assert.Nil(t, err)

	addressMock := []byte("address")
	currentShard := uint32(0)
	txValidatorHandler := getInterceptedTxHandler(currentShard, currentShard, txNonce, addressMock, big.NewInt(0))

	result := txValidator.CheckTxValidity(txValidatorHandler)
	assert.True(t, errors.Is(result, process.ErrWrongTransaction))
}

func TestTxValidator_CheckTxValidityAccountBalanceIsLessThanTxTotalValueShouldReturnFalse(t *testing.T) {
	t.Parallel()

	accountNonce := uint64(0)
	txNonce := uint64(1)
	fee := big.NewInt(1000)
	accountBalance := big.NewInt(10)

	adb := getAccAdapter(accountNonce, accountBalance)
	shardCoordinator := createMockCoordinator("_", 0)
	maxNonceDeltaAllowed := 100
	txValidator, err := dataValidators.NewTxValidator(
		adb,
		shardCoordinator,
		&testscommon.WhiteListHandlerStub{},
		mock.NewPubkeyConverterMock(32),
		&guardianMocks.GuardianSigVerifierStub{},
		&testscommon.TxVersionCheckerStub{},
		maxNonceDeltaAllowed,
	)
	assert.Nil(t, err)

	addressMock := []byte("address")
	currentShard := uint32(0)
	txValidatorHandler := getInterceptedTxHandler(currentShard, currentShard, txNonce, addressMock, fee)

	result := txValidator.CheckTxValidity(txValidatorHandler)
	assert.NotNil(t, result)
	assert.True(t, errors.Is(result, process.ErrInsufficientFunds))
}

func TestTxValidator_CheckTxValidityAccountNotExitsShouldReturnFalse(t *testing.T) {
	t.Parallel()

	accDB := &stateMock.AccountsStub{}
	accDB.GetExistingAccountCalled = func(address []byte) (handler vmcommon.AccountHandler, e error) {
		return nil, errors.New("cannot find account")
	}
	shardCoordinator := createMockCoordinator("_", 0)
	maxNonceDeltaAllowed := 100
	txValidator, _ := dataValidators.NewTxValidator(
		accDB,
		shardCoordinator,
		&testscommon.WhiteListHandlerStub{},
		mock.NewPubkeyConverterMock(32),
		&guardianMocks.GuardianSigVerifierStub{},
		&testscommon.TxVersionCheckerStub{},
		maxNonceDeltaAllowed,
	)

	addressMock := []byte("address")
	currentShard := uint32(0)
	txValidatorHandler := getInterceptedTxHandler(currentShard, currentShard, 1, addressMock, big.NewInt(0))

	result := txValidator.CheckTxValidity(txValidatorHandler)
	assert.True(t, errors.Is(result, process.ErrAccountNotFound))
}

func TestTxValidator_CheckTxValidityAccountNotExitsButWhiteListedShouldReturnTrue(t *testing.T) {
	t.Parallel()

	accDB := &stateMock.AccountsStub{}
	accDB.GetExistingAccountCalled = func(address []byte) (handler vmcommon.AccountHandler, e error) {
		return nil, errors.New("cannot find account")
	}
	shardCoordinator := createMockCoordinator("_", 0)
	maxNonceDeltaAllowed := 100
	txValidator, _ := dataValidators.NewTxValidator(
		accDB,
		shardCoordinator,
		&testscommon.WhiteListHandlerStub{
			IsWhiteListedCalled: func(interceptedData process.InterceptedData) bool {
				return true
			},
		},
		mock.NewPubkeyConverterMock(32),
		&guardianMocks.GuardianSigVerifierStub{},
		&testscommon.TxVersionCheckerStub{},
		maxNonceDeltaAllowed,
	)

	addressMock := []byte("address")
	currentShard := uint32(0)
	txValidatorHandler := getInterceptedTxHandler(currentShard, currentShard, 1, addressMock, big.NewInt(0))

	interceptedTx := struct {
		process.InterceptedData
		process.InterceptedTransactionHandler
	}{
		InterceptedData:               nil,
		InterceptedTransactionHandler: txValidatorHandler,
	}

	// interceptedTx needs to be of type InterceptedData & TxValidatorHandler
	result := txValidator.CheckTxValidity(interceptedTx)
	assert.Nil(t, result)
}

func TestTxValidator_CheckTxValidityWrongAccountTypeShouldReturnFalse(t *testing.T) {
	t.Parallel()

	accDB := &stateMock.AccountsStub{}
	accDB.GetExistingAccountCalled = func(address []byte) (handler vmcommon.AccountHandler, e error) {
		return state.NewPeerAccount(address)
	}
	shardCoordinator := createMockCoordinator("_", 0)
	maxNonceDeltaAllowed := 100
	txValidator, _ := dataValidators.NewTxValidator(
		accDB,
		shardCoordinator,
		&testscommon.WhiteListHandlerStub{},
		mock.NewPubkeyConverterMock(32),
		&guardianMocks.GuardianSigVerifierStub{},
		&testscommon.TxVersionCheckerStub{},
		maxNonceDeltaAllowed,
	)

	addressMock := []byte("address")
	currentShard := uint32(0)
	txValidatorHandler := getInterceptedTxHandler(currentShard, currentShard, 1, addressMock, big.NewInt(0))

	result := txValidator.CheckTxValidity(txValidatorHandler)
	assert.True(t, errors.Is(result, process.ErrWrongTypeAssertion))
}

func TestTxValidator_CheckTxValidityTxIsOkShouldReturnTrue(t *testing.T) {
	t.Parallel()

	accountNonce := uint64(0)
	accountBalance := big.NewInt(10)
	adb := getAccAdapter(accountNonce, accountBalance)
	shardCoordinator := createMockCoordinator("_", 0)
	maxNonceDeltaAllowed := 100
	txValidator, _ := dataValidators.NewTxValidator(
		adb,
		shardCoordinator,
		&testscommon.WhiteListHandlerStub{},
		mock.NewPubkeyConverterMock(32),
		&guardianMocks.GuardianSigVerifierStub{},
		&testscommon.TxVersionCheckerStub{},
		maxNonceDeltaAllowed,
	)

	addressMock := []byte("address")
	currentShard := uint32(0)
	txValidatorHandler := getInterceptedTxHandler(currentShard, currentShard, 1, addressMock, big.NewInt(0))

	result := txValidator.CheckTxValidity(txValidatorHandler)
	assert.Nil(t, result)
}

func TestTxValidator_checkPermission(t *testing.T) {
	adb := getAccAdapter(0, big.NewInt(0))
	shardCoordinator := createMockCoordinator("_", 0)
	maxNonceDeltaAllowed := 100
	txValidator, err := dataValidators.NewTxValidator(
		adb,
		shardCoordinator,
		&testscommon.WhiteListHandlerStub{},
		mock.NewPubkeyConverterMock(32),
		&guardianMocks.GuardianSigVerifierStub{},
		&testscommon.TxVersionCheckerStub{},
		maxNonceDeltaAllowed,
	)
	require.Nil(t, err)

	t.Run("non frozen account with getTxData error should err", func(t *testing.T) {
		inTx := getDefaultInterceptedTx()
		inTx.TransactionCalled = func() data.TransactionHandler {
			return nil
		}
		acc := &stateMock.UserAccountStub{
			IsFrozenCalled: func() bool {
				return false
			},
		}
		err = txValidator.CheckPermission(inTx, acc)
		require.Equal(t, process.ErrNilTransaction, err)
	})
	t.Run("non frozen account without getTxData error should allow", func(t *testing.T) {
		inTx := getDefaultInterceptedTx()
		inTx.TransactionCalled = func() data.TransactionHandler {
			return &transaction.Transaction{}
		}

		acc := &stateMock.UserAccountStub{
			IsFrozenCalled: func() bool {
				return false
			},
		}
		err = txValidator.CheckPermission(inTx, acc)
		require.Nil(t, err)
	})
	t.Run("frozen account with no guarded tx and no bypass permission should err", func(t *testing.T) {
		inTx := getDefaultInterceptedTx()
		inTx.TransactionCalled = func() data.TransactionHandler {
			return &transaction.Transaction{
				Data: []byte("dummy data"),
			}
		}

		acc := createDummyFrozenAccount()
		txV, err := dataValidators.NewTxValidator(
			adb,
			shardCoordinator,
			&testscommon.WhiteListHandlerStub{},
			mock.NewPubkeyConverterMock(32),
			&guardianMocks.GuardianSigVerifierStub{
				VerifyGuardianSignatureCalled: func(account vmcommon.UserAccountHandler, inTx process.InterceptedTransactionHandler) error {
					return errors.New("error")
				},
			},
			&testscommon.TxVersionCheckerStub{
				IsGuardedTransactionCalled: func(tx *transaction.Transaction) bool {
					return false
				},
			},
			maxNonceDeltaAllowed,
		)
		require.Nil(t, err)

		err = txV.CheckPermission(inTx, acc)
		require.True(t, errors.Is(err, process.ErrOperationNotPermitted))
	})
	t.Run("frozen account with no guarded tx and bypass permission should allow", func(t *testing.T) {
		inTx := getDefaultInterceptedTx()
		inTx.TransactionCalled = func() data.TransactionHandler {
			return &transaction.Transaction{
				Data: []byte("SetGuardian@..."),
			}
		}

		acc := createDummyFrozenAccount()
		txV, err := dataValidators.NewTxValidator(
			adb,
			shardCoordinator,
			&testscommon.WhiteListHandlerStub{},
			mock.NewPubkeyConverterMock(32),
			&guardianMocks.GuardianSigVerifierStub{
				VerifyGuardianSignatureCalled: func(account vmcommon.UserAccountHandler, inTx process.InterceptedTransactionHandler) error {
					return errors.New("error")
				},
			},
			&testscommon.TxVersionCheckerStub{
				IsGuardedTransactionCalled: func(tx *transaction.Transaction) bool {
					return false
				},
			},
			maxNonceDeltaAllowed,
		)
		require.Nil(t, err)

		err = txV.CheckPermission(inTx, acc)
		require.Nil(t, err)
	})
	t.Run("frozen account with guarded Tx should allow", func(t *testing.T) {
		inTx := getDefaultInterceptedTx()
		inTx.TransactionCalled = func() data.TransactionHandler {
			return &transaction.Transaction{
				Data: []byte("dummy data"),
			}
		}

		acc := createDummyFrozenAccount()
		txV, err := dataValidators.NewTxValidator(
			adb,
			shardCoordinator,
			&testscommon.WhiteListHandlerStub{},
			mock.NewPubkeyConverterMock(32),
			&guardianMocks.GuardianSigVerifierStub{
				VerifyGuardianSignatureCalled: func(account vmcommon.UserAccountHandler, inTx process.InterceptedTransactionHandler) error {
					return nil
				},
			},
			&testscommon.TxVersionCheckerStub{
				IsGuardedTransactionCalled: func(tx *transaction.Transaction) bool {
					return true
				},
			},
			maxNonceDeltaAllowed,
		)
		require.Nil(t, err)

		err = txV.CheckPermission(inTx, acc)
		require.Nil(t, err)
	})
}

func TestTxValidator_checkGuardedTransaction(t *testing.T) {
	adb := getAccAdapter(0, big.NewInt(0))
	shardCoordinator := createMockCoordinator("_", 0)
	maxNonceDeltaAllowed := 100
	txValidator, err := dataValidators.NewTxValidator(
		adb,
		shardCoordinator,
		&testscommon.WhiteListHandlerStub{},
		mock.NewPubkeyConverterMock(32),
		&guardianMocks.GuardianSigVerifierStub{},
		&testscommon.TxVersionCheckerStub{},
		maxNonceDeltaAllowed,
	)
	require.Nil(t, err)

	t.Run("nil tx should err", func(t *testing.T) {
		inTx := getDefaultInterceptedTx()
		inTx.TransactionCalled = func() data.TransactionHandler {
			return nil
		}
		acc := &stateMock.UserAccountStub{}
		err = txValidator.CheckGuardedTransaction(inTx, acc)
		require.Equal(t, process.ErrNilTransaction, err)
	})
	t.Run("invalid transaction should fail", func(t *testing.T) {
		inTx := getDefaultInterceptedTx()
		inTx.TransactionCalled = func() data.TransactionHandler {
			return &receipt.Receipt{}
		}
		acc := &stateMock.UserAccountStub{}
		err = txValidator.CheckGuardedTransaction(inTx, acc)
		require.True(t, errors.Is(err, process.ErrWrongTypeAssertion))
	})
	t.Run("not guarded Tx should err", func(t *testing.T) {
		inTx := getDefaultInterceptedTx()
		inTx.TransactionCalled = func() data.TransactionHandler {
			return &transaction.Transaction{}
		}
		acc := &stateMock.UserAccountStub{}

		txV, err := dataValidators.NewTxValidator(
			adb,
			shardCoordinator,
			&testscommon.WhiteListHandlerStub{},
			mock.NewPubkeyConverterMock(32),
			&guardianMocks.GuardianSigVerifierStub{},
			&testscommon.TxVersionCheckerStub{
				IsGuardedTransactionCalled: func(tx *transaction.Transaction) bool {
					return false
				},
			},
			maxNonceDeltaAllowed,
		)
		require.Nil(t, err)
		err = txV.CheckGuardedTransaction(inTx, acc)
		require.True(t, errors.Is(err, process.ErrOperationNotPermitted))
	})
	t.Run("non user account should err", func(t *testing.T) {
		inTx := getDefaultInterceptedTx()
		inTx.TransactionCalled = func() data.TransactionHandler {
			return &transaction.Transaction{}
		}

		var acc state.UserAccountHandler

		txV, err := dataValidators.NewTxValidator(
			adb,
			shardCoordinator,
			&testscommon.WhiteListHandlerStub{},
			mock.NewPubkeyConverterMock(32),
			&guardianMocks.GuardianSigVerifierStub{},
			&testscommon.TxVersionCheckerStub{
				IsGuardedTransactionCalled: func(tx *transaction.Transaction) bool {
					return true
				},
			},
			maxNonceDeltaAllowed,
		)
		require.Nil(t, err)
		err = txV.CheckGuardedTransaction(inTx, acc)
		require.True(t, errors.Is(err, process.ErrWrongTypeAssertion))
	})
	t.Run("invalid guardian signature should err", func(t *testing.T) {
		inTx := getDefaultInterceptedTx()
		inTx.TransactionCalled = func() data.TransactionHandler {
			return &transaction.Transaction{}
		}

		acc := state.NewEmptyUserAccount()

		expectedSigVerifyError := errors.New("expected error")

		txV, err := dataValidators.NewTxValidator(
			adb,
			shardCoordinator,
			&testscommon.WhiteListHandlerStub{},
			mock.NewPubkeyConverterMock(32),
			&guardianMocks.GuardianSigVerifierStub{
				VerifyGuardianSignatureCalled: func(account vmcommon.UserAccountHandler, inTx process.InterceptedTransactionHandler) error {
					return expectedSigVerifyError
				},
			},
			&testscommon.TxVersionCheckerStub{
				IsGuardedTransactionCalled: func(tx *transaction.Transaction) bool {
					return true
				},
			},
			maxNonceDeltaAllowed,
		)
		require.Nil(t, err)
		err = txV.CheckGuardedTransaction(inTx, acc)
		require.True(t, errors.Is(err, process.ErrOperationNotPermitted))
		require.True(t, strings.Contains(err.Error(), expectedSigVerifyError.Error()))
	})
	t.Run("valid signed guarded tx OK", func(t *testing.T) {
		inTx := getDefaultInterceptedTx()
		inTx.TransactionCalled = func() data.TransactionHandler {
			return &transaction.Transaction{}
		}

		acc := state.NewEmptyUserAccount()
		txV, err := dataValidators.NewTxValidator(
			adb,
			shardCoordinator,
			&testscommon.WhiteListHandlerStub{},
			mock.NewPubkeyConverterMock(32),
			&guardianMocks.GuardianSigVerifierStub{
				VerifyGuardianSignatureCalled: func(account vmcommon.UserAccountHandler, inTx process.InterceptedTransactionHandler) error {
					return nil
				},
			},
			&testscommon.TxVersionCheckerStub{
				IsGuardedTransactionCalled: func(tx *transaction.Transaction) bool {
					return true
				},
			},
			maxNonceDeltaAllowed,
		)
		require.Nil(t, err)
		err = txV.CheckGuardedTransaction(inTx, acc)
		require.Nil(t, err)
	})
}

func Test_checkOperationAllowedToBypassGuardian(t *testing.T) {
	t.Run("operations not allowed to bypass", func(t *testing.T) {
		txData := []byte("#@!")
		require.Equal(t, process.ErrOperationNotPermitted, dataValidators.CheckOperationAllowedToBypassGuardian(txData))
		txData = []byte(nil)
		require.Equal(t, process.ErrOperationNotPermitted, dataValidators.CheckOperationAllowedToBypassGuardian(txData))
		txData = []byte("SomeOtherFunction@")
		require.Equal(t, process.ErrOperationNotPermitted, dataValidators.CheckOperationAllowedToBypassGuardian(txData))
	})
	t.Run("setGuardian data field (non builtin call) not allowed", func(t *testing.T) {
		txData := []byte("setGuardian")
		require.Equal(t, process.ErrOperationNotPermitted, dataValidators.CheckOperationAllowedToBypassGuardian(txData))
	})
	t.Run("set guardian builtin call allowed to bypass", func(t *testing.T) {
		txData := []byte("SetGuardian@")
		require.Nil(t, dataValidators.CheckOperationAllowedToBypassGuardian(txData))
	})
}

func Test_getTxData(t *testing.T) {
	t.Run("nil tx in intercepted tx returns error", func(t *testing.T) {
		interceptedTx := getDefaultInterceptedTx()
		interceptedTx.TransactionCalled = func() data.TransactionHandler { return nil }
		txData, err := dataValidators.GetTxData(interceptedTx)
		require.Equal(t, process.ErrNilTransaction, err)
		require.Nil(t, txData)
	})
	t.Run("non nil intercepted tx without data", func(t *testing.T) {
		expectedData := []byte(nil)
		interceptedTx := getDefaultInterceptedTx()
		interceptedTx.TransactionCalled = func() data.TransactionHandler {
			return &transaction.Transaction{
				Data: expectedData,
			}
		}
		txData, err := dataValidators.GetTxData(interceptedTx)
		require.Nil(t, err)
		require.Equal(t, expectedData, txData)
	})
	t.Run("non nil intercepted tx with data", func(t *testing.T) {
		expectedData := []byte("expected data")
		interceptedTx := getDefaultInterceptedTx()
		interceptedTx.TransactionCalled = func() data.TransactionHandler {
			return &transaction.Transaction{
				Data: expectedData,
			}
		}
		txData, err := dataValidators.GetTxData(interceptedTx)
		require.Nil(t, err)
		require.Equal(t, expectedData, txData)
	})
}

//------- IsInterfaceNil

func TestTxValidator_IsInterfaceNil(t *testing.T) {
	t.Parallel()

	adb := getAccAdapter(0, big.NewInt(0))
	shardCoordinator := createMockCoordinator("_", 0)
	txValidator, _ := dataValidators.NewTxValidator(
		adb,
		shardCoordinator,
		&testscommon.WhiteListHandlerStub{},
		mock.NewPubkeyConverterMock(32),
		&guardianMocks.GuardianSigVerifierStub{},
		&testscommon.TxVersionCheckerStub{},
		100,
	)
	_ = txValidator
	txValidator = nil

	assert.True(t, check.IfNil(txValidator))
}

func getDefaultInterceptedTx() *mock.InterceptedTxHandlerStub {
	return &mock.InterceptedTxHandlerStub{
		SenderShardIdCalled: func() uint32 {
			return 0
		},
		ReceiverShardIdCalled: func() uint32 {
			return 1
		},
		NonceCalled: func() uint64 {
			return 0
		},
		SenderAddressCalled: func() []byte {
			return []byte("sender address")
		},
		FeeCalled: func() *big.Int {
			return big.NewInt(100000)
		},
		TransactionCalled: func() data.TransactionHandler {
			return &transaction.Transaction{}
		},
	}
}

func createDummyFrozenAccount() state.UserAccountHandler {
	acc := state.NewEmptyUserAccount()
	metadata := &vmcommon.CodeMetadata{Frozen: true}
	acc.SetCodeMetadata(metadata.ToBytes())
	return acc
}
