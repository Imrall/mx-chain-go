package arwenvm

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/multiversx/mx-chain-go/integrationTests"
	"github.com/multiversx/mx-chain-vm-common-go/txDataBuilder"
	"github.com/multiversx/mx-chain-vm-go/mock/contracts"
	"github.com/multiversx/mx-chain-vm-go/testcommon"
	test "github.com/multiversx/mx-chain-vm-go/testcommon"
	"github.com/stretchr/testify/require"
)

var LegacyAsyncCallType = []byte{0}
var NewAsyncCallType = []byte{1}

func TestMockContract_AsyncLegacy_InShard(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}
	testConfig := &testcommon.TestConfig{
		GasProvided:     2000,
		GasUsedByParent: 400,
	}

	transferEGLD := big.NewInt(42)

	net := integrationTests.NewTestNetworkSized(t, 1, 1, 1)
	net.Start()
	net.Step()

	net.CreateWallets(1)
	net.MintWalletsUint64(100000000000)
	owner := net.Wallets[0]

	parentAddress, _ := GetAddressForNewAccount(t, net, net.NodesSharded[0][0])

	InitializeMockContracts(
		t, net,
		test.CreateMockContract(parentAddress).
			WithConfig(testConfig).
			WithMethods(contracts.WasteGasParentMock),
	)

	txData := txDataBuilder.NewBuilder().Func("wasteGas").ToBytes()
	tx := net.CreateTx(owner, parentAddress, transferEGLD, txData)
	tx.GasLimit = testConfig.GasProvided

	_ = net.SignAndSendTx(owner, tx)

	net.Steps(2)

	parentHandler := net.GetAccountHandler(parentAddress)
	expectedEgld := big.NewInt(0)
	expectedEgld.Add(MockInitialBalance, transferEGLD)
	require.Equal(t, expectedEgld, parentHandler.GetBalance())
}

func TestMockContract_AsyncLegacy_CrossShard(t *testing.T) {
	testMockContract_CrossShard(t, LegacyAsyncCallType)
}

func TestMockContract_NewAsync_CrossShard(t *testing.T) {
	testMockContract_CrossShard(t, NewAsyncCallType)
}

func testMockContract_CrossShard(t *testing.T, asyncCallType []byte) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}
	transferEGLD := big.NewInt(42)

	numberOfShards := 2
	net := integrationTests.NewTestNetworkSized(t, numberOfShards, 1, 1)
	net.Start()
	net.Step()

	net.CreateWallets(2)
	net.MintWalletsUint64(100000000000)
	ownerOfParent := net.Wallets[0]

	parentAddress, _ := GetAddressForNewAccount(t, net, net.NodesSharded[0][0])
	childAddress, _ := GetAddressForNewAccount(t, net, net.NodesSharded[1][0])

	thirdPartyAddress := MakeTestWalletAddress("thirdPartyAddress")
	vaultAddress := MakeTestWalletAddress("vaultAddress")

	testConfig := &testcommon.TestConfig{
		ParentBalance: 20,
		ChildBalance:  10,

		GasProvided:        2_000_000,
		GasProvidedToChild: 1_000_000,
		GasUsedByParent:    400,

		ChildAddress:              childAddress,
		ThirdPartyAddress:         thirdPartyAddress,
		VaultAddress:              vaultAddress,
		TransferFromParentToChild: 8,

		SuccessCallback: "myCallBack",
		ErrorCallback:   "myCallBack",

		IsLegacyAsync: bytes.Equal(asyncCallType, LegacyAsyncCallType),
	}

	InitializeMockContracts(
		t, net,
		test.CreateMockContractOnShard(parentAddress, 0).
			WithBalance(testConfig.ParentBalance).
			WithConfig(testConfig).
			WithMethods(contracts.PerformAsyncCallParentMock, contracts.CallBackParentMock),
		test.CreateMockContractOnShard(childAddress, 1).
			WithBalance(testConfig.ChildBalance).
			WithConfig(testConfig).
			WithMethods(contracts.TransferToThirdPartyAsyncChildMock),
	)

	txData := txDataBuilder.
		NewBuilder().
		Func("performAsyncCall").
		Bytes([]byte{0}).
		Bytes(asyncCallType).
		ToBytes()
	tx := net.CreateTx(ownerOfParent, parentAddress, transferEGLD, txData)
	tx.GasLimit = testConfig.GasProvided

	_ = net.SignAndSendTx(ownerOfParent, tx)

	net.Steps(16)

	parentHandler, err := net.NodesSharded[0][0].BlockchainHook.GetUserAccount(parentAddress)
	require.Nil(t, err)

	parentValueA, _, err := parentHandler.AccountDataHandler().RetrieveValue(test.ParentKeyA)
	require.Nil(t, err)
	require.Equal(t, test.ParentDataA, parentValueA)

	parentValueB, _, err := parentHandler.AccountDataHandler().RetrieveValue(test.ParentKeyB)
	require.Nil(t, err)
	require.Equal(t, test.ParentDataB, parentValueB)

	callbackValue, _, err := parentHandler.AccountDataHandler().RetrieveValue(test.CallbackKey)
	require.Nil(t, err)
	require.Equal(t, test.CallbackData, callbackValue)

	childHandler, err := net.NodesSharded[1][0].BlockchainHook.GetUserAccount(childAddress)
	require.Nil(t, err)

	childValue, _, err := childHandler.AccountDataHandler().RetrieveValue(test.ChildKey)
	require.Nil(t, err)
	require.Equal(t, test.ChildData, childValue)
}
