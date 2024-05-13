package runType_test

import (
	"testing"

	"github.com/multiversx/mx-chain-core-go/hashing"
	"github.com/multiversx/mx-chain-core-go/marshal"
	"github.com/multiversx/mx-chain-go/common"
	"github.com/multiversx/mx-chain-go/errors"
	"github.com/multiversx/mx-chain-go/factory"
	"github.com/multiversx/mx-chain-go/factory/runType"
	"github.com/multiversx/mx-chain-go/testscommon/enableEpochsHandlerMock"
	factoryMock "github.com/multiversx/mx-chain-go/testscommon/factory"
	"github.com/multiversx/mx-chain-go/testscommon/hashingMocks"
	"github.com/multiversx/mx-chain-go/testscommon/marshallerMock"
	"github.com/stretchr/testify/require"
)

func TestNewManagedRunTypeComponents(t *testing.T) {
	t.Parallel()

	t.Run("nil factory should error", func(t *testing.T) {
		t.Parallel()

		managedRunTypeComponents, err := runType.NewManagedRunTypeComponents(nil)
		require.Equal(t, errors.ErrNilRunTypeComponentsFactory, err)
		require.Nil(t, managedRunTypeComponents)
	})
	t.Run("should work", func(t *testing.T) {
		t.Parallel()

		managedRunTypeComponents, err := createComponents()
		require.NoError(t, err)
		require.NotNil(t, managedRunTypeComponents)
	})
}

func TestManagedRunTypeComponents_Create(t *testing.T) {
	t.Parallel()

	t.Run("should work with getters", func(t *testing.T) {
		t.Parallel()

		managedRunTypeComponents, err := createComponents()
		require.NoError(t, err)

		require.Nil(t, managedRunTypeComponents.BlockChainHookHandlerCreator())
		require.Nil(t, managedRunTypeComponents.EpochStartBootstrapperCreator())
		require.Nil(t, managedRunTypeComponents.BootstrapperFromStorageCreator())
		require.Nil(t, managedRunTypeComponents.BlockProcessorCreator())
		require.Nil(t, managedRunTypeComponents.ForkDetectorCreator())
		require.Nil(t, managedRunTypeComponents.BlockTrackerCreator())
		require.Nil(t, managedRunTypeComponents.RequestHandlerCreator())
		require.Nil(t, managedRunTypeComponents.HeaderValidatorCreator())
		require.Nil(t, managedRunTypeComponents.ScheduledTxsExecutionCreator())
		require.Nil(t, managedRunTypeComponents.TransactionCoordinatorCreator())
		require.Nil(t, managedRunTypeComponents.ValidatorStatisticsProcessorCreator())
		require.Nil(t, managedRunTypeComponents.VMContextCreator())

		err = managedRunTypeComponents.Create()
		require.NoError(t, err)
		require.NotNil(t, managedRunTypeComponents.BlockChainHookHandlerCreator())
		require.NotNil(t, managedRunTypeComponents.EpochStartBootstrapperCreator())
		require.NotNil(t, managedRunTypeComponents.BootstrapperFromStorageCreator())
		require.NotNil(t, managedRunTypeComponents.BlockProcessorCreator())
		require.NotNil(t, managedRunTypeComponents.ForkDetectorCreator())
		require.NotNil(t, managedRunTypeComponents.BlockTrackerCreator())
		require.NotNil(t, managedRunTypeComponents.RequestHandlerCreator())
		require.NotNil(t, managedRunTypeComponents.HeaderValidatorCreator())
		require.NotNil(t, managedRunTypeComponents.ScheduledTxsExecutionCreator())
		require.NotNil(t, managedRunTypeComponents.TransactionCoordinatorCreator())
		require.NotNil(t, managedRunTypeComponents.ValidatorStatisticsProcessorCreator())
		require.NotNil(t, managedRunTypeComponents.VMContextCreator())

		require.Equal(t, factory.RunTypeComponentsName, managedRunTypeComponents.String())
		require.NoError(t, managedRunTypeComponents.Close())
	})
}

func TestManagedRunTypeComponents_Close(t *testing.T) {
	t.Parallel()

	managedRunTypeComponents, _ := createComponents()
	require.NoError(t, managedRunTypeComponents.Close())

	err := managedRunTypeComponents.Create()
	require.NoError(t, err)

	require.NoError(t, managedRunTypeComponents.Close())
	require.Nil(t, managedRunTypeComponents.BlockChainHookHandlerCreator())
}

func TestManagedRunTypeComponents_CheckSubcomponents(t *testing.T) {
	t.Parallel()

	managedRunTypeComponents, _ := createComponents()
	err := managedRunTypeComponents.CheckSubcomponents()
	require.Equal(t, errors.ErrNilRunTypeComponents, err)

	err = managedRunTypeComponents.Create()
	require.NoError(t, err)

	err = managedRunTypeComponents.CheckSubcomponents()
	require.NoError(t, err)

	require.NoError(t, managedRunTypeComponents.Close())
}

func TestManagedRunTypeComponents_IsInterfaceNil(t *testing.T) {
	t.Parallel()

	var managedRunTypeComponents factory.RunTypeComponentsHandler
	managedRunTypeComponents, _ = runType.NewManagedRunTypeComponents(nil)
	require.True(t, managedRunTypeComponents.IsInterfaceNil())

	managedRunTypeComponents, _ = createComponents()
	require.False(t, managedRunTypeComponents.IsInterfaceNil())
}

func createComponents() (factory.RunTypeComponentsHandler, error) {
	rcf, _ := runType.NewRunTypeComponentsFactory(createCoreComponents())
	return runType.NewManagedRunTypeComponents(rcf)
}

func createCoreComponents() factory.CoreComponentsHolder {
	return &factoryMock.CoreComponentsHolderMock{
		InternalMarshalizerCalled: func() marshal.Marshalizer {
			return &marshallerMock.MarshalizerMock{}
		},
		HasherCalled: func() hashing.Hasher {
			return &hashingMocks.HasherMock{}
		},
		EnableEpochsHandlerCalled: func() common.EnableEpochsHandler {
			return &enableEpochsHandlerMock.EnableEpochsHandlerStub{}
		},
	}
}
