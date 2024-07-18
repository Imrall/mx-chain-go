package bridge

import (
	"bytes"
	"math/big"
	"testing"
	"time"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/data/esdt"
	"github.com/stretchr/testify/require"

	"github.com/multiversx/mx-chain-go/config"
	chainSim "github.com/multiversx/mx-chain-go/integrationTests/chainSimulator"
	"github.com/multiversx/mx-chain-go/node/chainSimulator"
	"github.com/multiversx/mx-chain-go/node/chainSimulator/components/api"
	"github.com/multiversx/mx-chain-go/node/chainSimulator/dtos"
)

const (
	//enshrine esdt-safe contract without checks for prefix or issue cost paid for new tokens
	simpleEsdtSafeWasmPath = "testdata/simple-esdt-safe.wasm"
	actionNotAllowed       = "action is not allowed"
)

func TestChainSimulator_ExecuteOperationNotAllowedToMintFungibleTokenWithoutPrefix(t *testing.T) {
	bridgedInTokens := make([]chainSim.ArgsDepositToken, 0)
	bridgedInTokens = append(bridgedInTokens, chainSim.ArgsDepositToken{
		Identifier: "SOVT-5d8f56",
		Nonce:      0,
		Amount:     big.NewInt(123),
		Type:       core.Fungible,
	})

	testExecuteOperationNotAllowedToMintTokenWithoutPrefix(t, bridgedInTokens)
}

func TestChainSimulator_ExecuteOperationNotAllowedToMintNonFungibleTokenWithoutPrefix(t *testing.T) {
	bridgedInTokens := make([]chainSim.ArgsDepositToken, 0)
	bridgedInTokens = append(bridgedInTokens, chainSim.ArgsDepositToken{
		Identifier: "SOVT-5d8f56",
		Nonce:      5,
		Amount:     big.NewInt(1),
		Type:       core.NonFungible,
	})

	testExecuteOperationNotAllowedToMintTokenWithoutPrefix(t, bridgedInTokens)
}

func TestChainSimulator_ExecuteOperationNotAllowedToMintSemiFungibleTokenWithoutPrefix(t *testing.T) {
	bridgedInTokens := make([]chainSim.ArgsDepositToken, 0)
	bridgedInTokens = append(bridgedInTokens, chainSim.ArgsDepositToken{
		Identifier: "SOVT-5d8f56",
		Nonce:      3,
		Amount:     big.NewInt(15),
		Type:       core.SemiFungible,
	})

	testExecuteOperationNotAllowedToMintTokenWithoutPrefix(t, bridgedInTokens)
}

// Test flow:
// - deploy sovereign bridge contracts on the main chain
// - whitelist the bridge esdt safe contract to allow it to burn/mint cross chain esdt tokens
// - executeOperation token without prefix should not be allowed
func testExecuteOperationNotAllowedToMintTokenWithoutPrefix(
	t *testing.T,
	bridgedInTokens []chainSim.ArgsDepositToken,
) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	roundsPerEpoch := core.OptionalUint64{
		HasValue: true,
		Value:    20,
	}

	whiteListedAddress := "erd1qqqqqqqqqqqqqpgqmzzm05jeav6d5qvna0q2pmcllelkz8xddz3syjszx5"
	cs, err := chainSimulator.NewChainSimulator(chainSimulator.ArgsChainSimulator{
		BypassTxSignatureCheck:   true,
		TempDir:                  t.TempDir(),
		PathToInitialConfig:      defaultPathToInitialConfig,
		NumOfShards:              1,
		GenesisTimestamp:         time.Now().Unix(),
		RoundDurationInMillis:    uint64(6000),
		RoundsPerEpoch:           roundsPerEpoch,
		ApiInterface:             api.NewNoApiInterface(),
		MinNodesPerShard:         3,
		MetaChainMinNodes:        3,
		NumNodesWaitingListMeta:  0,
		NumNodesWaitingListShard: 0,
		AlterConfigsFunction: func(cfg *config.Configs) {
			cfg.GeneralConfig.VirtualMachine.Execution.TransferAndExecuteByUserAddresses = []string{whiteListedAddress}
		},
	})
	require.Nil(t, err)
	require.NotNil(t, cs)

	defer cs.Close()

	err = cs.GenerateBlocksUntilEpochIsReached(3)
	require.Nil(t, err)

	nodeHandler := cs.GetNodeHandler(0)

	// Deploy bridge setup
	initialAddress := "erd1l6xt0rqlyzw56a3k8xwwshq2dcjwy3q9cppucvqsmdyw8r98dz3sae0kxl"
	argsEsdtSafe := ArgsEsdtSafe{
		ChainPrefix:       "sov1",
		IssuePaymentToken: "ABC-123456",
	}
	bridgeData := deployBridgeSetup(t, cs, initialAddress, simpleEsdtSafeWasmPath, argsEsdtSafe, feeMarketWasmPath)

	esdtSafeEncoded, _ := nodeHandler.GetCoreComponents().AddressPubKeyConverter().Encode(bridgeData.ESDTSafeAddress)
	require.Equal(t, whiteListedAddress, esdtSafeEncoded)

	wallet, err := cs.GenerateAndMintWalletAddress(0, chainSim.InitialAmount)
	require.Nil(t, err)

	// execute operation, bridge in token without prefix
	// expecting not allowed to mint because token has no prefix
	txResult := executeOperation(t, cs, bridgeData.OwnerAccount.Wallet, wallet.Bytes, &bridgeData.OwnerAccount.Nonce, bridgeData.ESDTSafeAddress, bridgedInTokens, wallet.Bytes, nil)
	chainSim.RequireSignalError(t, txResult, actionNotAllowed)
}

func TestChainSimulator_ExecuteOperationNotAllowedToMintFungibleContractNotWhitelisted(t *testing.T) {
	prefix := "sov1"
	bridgedInTokens := make([]chainSim.ArgsDepositToken, 0)
	bridgedInTokens = append(bridgedInTokens, chainSim.ArgsDepositToken{
		Identifier: prefix + "-SOVT-5d8f56",
		Nonce:      0,
		Amount:     big.NewInt(123),
		Type:       core.Fungible,
	})

	testExecuteOperationNotAllowedToMintFungibleContractNotWhitelisted(t, prefix, bridgedInTokens)
}

func TestChainSimulator_ExecuteOperationNotAllowedToMintNonFungibleContractNotWhitelisted(t *testing.T) {
	prefix := "sov2"
	bridgedInTokens := make([]chainSim.ArgsDepositToken, 0)
	bridgedInTokens = append(bridgedInTokens, chainSim.ArgsDepositToken{
		Identifier: prefix + "-SOVT-5d8f56",
		Nonce:      5,
		Amount:     big.NewInt(1),
		Type:       core.NonFungible,
	})

	testExecuteOperationNotAllowedToMintFungibleContractNotWhitelisted(t, prefix, bridgedInTokens)
}

func TestChainSimulator_ExecuteOperationNotAllowedToMintSemiFungibleContractNotWhitelisted(t *testing.T) {
	prefix := "sov3"
	bridgedInTokens := make([]chainSim.ArgsDepositToken, 0)
	bridgedInTokens = append(bridgedInTokens, chainSim.ArgsDepositToken{
		Identifier: prefix + "-SOVT-5d8f56",
		Nonce:      3,
		Amount:     big.NewInt(15),
		Type:       core.SemiFungible,
	})

	testExecuteOperationNotAllowedToMintFungibleContractNotWhitelisted(t, prefix, bridgedInTokens)
}

// Test flow:
// - deploy sovereign bridge contracts on the main chain
// - whitelist WRONG bridge esdt safe contract not allowing to burn/mint cross chain esdt tokens
// - executeOperation token with prefix should not be allowed
func testExecuteOperationNotAllowedToMintFungibleContractNotWhitelisted(
	t *testing.T,
	prefix string,
	bridgedInTokens []chainSim.ArgsDepositToken,
) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	roundsPerEpoch := core.OptionalUint64{
		HasValue: true,
		Value:    20,
	}

	whiteListedAddress := "erd1qqqqqqqqqqqqqpgqcw92wj0huvaghg4aeuykknp7hstmrmhudz3shjdhtt"
	cs, err := chainSimulator.NewChainSimulator(chainSimulator.ArgsChainSimulator{
		BypassTxSignatureCheck:   true,
		TempDir:                  t.TempDir(),
		PathToInitialConfig:      defaultPathToInitialConfig,
		NumOfShards:              1,
		GenesisTimestamp:         time.Now().Unix(),
		RoundDurationInMillis:    uint64(6000),
		RoundsPerEpoch:           roundsPerEpoch,
		ApiInterface:             api.NewNoApiInterface(),
		MinNodesPerShard:         3,
		MetaChainMinNodes:        3,
		NumNodesWaitingListMeta:  0,
		NumNodesWaitingListShard: 0,
		AlterConfigsFunction: func(cfg *config.Configs) {
			cfg.GeneralConfig.VirtualMachine.Execution.TransferAndExecuteByUserAddresses = []string{whiteListedAddress}
		},
	})
	require.Nil(t, err)
	require.NotNil(t, cs)

	defer cs.Close()

	err = cs.GenerateBlocksUntilEpochIsReached(3)
	require.Nil(t, err)

	nodeHandler := cs.GetNodeHandler(0)

	// Deploy bridge setup
	initialAddress := "erd1l6xt0rqlyzw56a3k8xwwshq2dcjwy3q9cppucvqsmdyw8r98dz3sae0kxl"
	argsEsdtSafe := ArgsEsdtSafe{
		ChainPrefix:       prefix,
		IssuePaymentToken: "ABC-123456",
	}
	bridgeData := deployBridgeSetup(t, cs, initialAddress, simpleEsdtSafeWasmPath, argsEsdtSafe, feeMarketWasmPath)

	// esdt-safe address generated is NOT whitelisted
	esdtSafeEncoded, _ := nodeHandler.GetCoreComponents().AddressPubKeyConverter().Encode(bridgeData.ESDTSafeAddress)
	require.NotEqual(t, whiteListedAddress, esdtSafeEncoded)

	wallet, err := cs.GenerateAndMintWalletAddress(0, chainSim.InitialAmount)
	require.Nil(t, err)

	// execute operation, bridge in token with prefix
	// expecting not allowed to mint because ESDT-safe contract is not whitelisted
	txResult := executeOperation(t, cs, bridgeData.OwnerAccount.Wallet, wallet.Bytes, &bridgeData.OwnerAccount.Nonce, bridgeData.ESDTSafeAddress, bridgedInTokens, wallet.Bytes, nil)
	chainSim.RequireSignalError(t, txResult, actionNotAllowed)
}

func TestChainSimulator_DepositNotAllowedToBurnFungibleContractNotWhitelisted(t *testing.T) {
	prefix := "sov1"
	bridgedOutTokens := make([]chainSim.ArgsDepositToken, 0)
	bridgedOutTokens = append(bridgedOutTokens, chainSim.ArgsDepositToken{
		Identifier: prefix + "-SOVT-5d8f56",
		Nonce:      0,
		Amount:     big.NewInt(123),
		Type:       core.Fungible,
	})

	testDepositNotAllowedToBurnTokensContractNotWhitelisted(t, prefix, bridgedOutTokens)
}

func TestChainSimulator_DepositNotAllowedToBurnNonFungibleContractNotWhitelisted(t *testing.T) {
	prefix := "sov2"
	bridgedOutTokens := make([]chainSim.ArgsDepositToken, 0)
	bridgedOutTokens = append(bridgedOutTokens, chainSim.ArgsDepositToken{
		Identifier: prefix + "-SOVNFT-5d8f56",
		Nonce:      3,
		Amount:     big.NewInt(1),
		Type:       core.NonFungible,
	})

	testDepositNotAllowedToBurnTokensContractNotWhitelisted(t, prefix, bridgedOutTokens)
}

func TestChainSimulator_DepositNotAllowedToBurnSemiFungibleContractNotWhitelisted(t *testing.T) {
	prefix := "sov3"
	bridgedOutTokens := make([]chainSim.ArgsDepositToken, 0)
	bridgedOutTokens = append(bridgedOutTokens, chainSim.ArgsDepositToken{
		Identifier: prefix + "-SOVSFT-5d8f56",
		Nonce:      3,
		Amount:     big.NewInt(15),
		Type:       core.SemiFungible,
	})

	testDepositNotAllowedToBurnTokensContractNotWhitelisted(t, prefix, bridgedOutTokens)
}

// Test flow:
// - deploy sovereign bridge contracts on the main chain
// - whitelist WRONG bridge esdt safe contract not allowing to burn/mint cross chain esdt tokens
// - deposit token with prefix should not be allowed
func testDepositNotAllowedToBurnTokensContractNotWhitelisted(
	t *testing.T,
	prefix string,
	bridgedOutTokens []chainSim.ArgsDepositToken,
) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	roundsPerEpoch := core.OptionalUint64{
		HasValue: true,
		Value:    20,
	}

	whiteListedAddress := "erd1qqqqqqqqqqqqqpgqcw92wj0huvaghg4aeuykknp7hstmrmhudz3shjdhtt"
	cs, err := chainSimulator.NewChainSimulator(chainSimulator.ArgsChainSimulator{
		BypassTxSignatureCheck:   true,
		TempDir:                  t.TempDir(),
		PathToInitialConfig:      defaultPathToInitialConfig,
		NumOfShards:              1,
		GenesisTimestamp:         time.Now().Unix(),
		RoundDurationInMillis:    uint64(6000),
		RoundsPerEpoch:           roundsPerEpoch,
		ApiInterface:             api.NewNoApiInterface(),
		MinNodesPerShard:         3,
		MetaChainMinNodes:        3,
		NumNodesWaitingListMeta:  0,
		NumNodesWaitingListShard: 0,
		AlterConfigsFunction: func(cfg *config.Configs) {
			cfg.GeneralConfig.VirtualMachine.Execution.TransferAndExecuteByUserAddresses = []string{whiteListedAddress}
		},
	})
	require.Nil(t, err)
	require.NotNil(t, cs)

	defer cs.Close()

	err = cs.GenerateBlocksUntilEpochIsReached(3)
	require.Nil(t, err)

	nodeHandler := cs.GetNodeHandler(0)

	// Deploy bridge setup
	initialAddress := "erd1l6xt0rqlyzw56a3k8xwwshq2dcjwy3q9cppucvqsmdyw8r98dz3sae0kxl"
	argsEsdtSafe := ArgsEsdtSafe{
		ChainPrefix:       prefix,
		IssuePaymentToken: "ABC-123456",
	}
	bridgeData := deployBridgeSetup(t, cs, initialAddress, simpleEsdtSafeWasmPath, argsEsdtSafe, feeMarketWasmPath)

	// esdt-safe address generated is NOT whitelisted
	esdtSafeEncoded, _ := nodeHandler.GetCoreComponents().AddressPubKeyConverter().Encode(bridgeData.ESDTSafeAddress)
	require.NotEqual(t, whiteListedAddress, esdtSafeEncoded)

	walletAddress := "erd1j7t00tq4jhcarcl2kuw3aywplnqlh9y6w9sxx2v6namc3jxvn5qq0jumrh"
	err = cs.SetStateMultiple([]*dtos.AddressState{
		{
			Address: walletAddress,
			Balance: "10000000000000000000000",
		},
	})
	require.Nil(t, err)
	walletAddrBytes, _ := nodeHandler.GetCoreComponents().AddressPubKeyConverter().Decode(walletAddress)

	wallet := dtos.WalletAddress{Bech32: walletAddress, Bytes: walletAddrBytes}
	nonce := uint64(0)

	for _, token := range bridgedOutTokens {
		tokenData := esdt.ESDigitalToken{
			Value:         token.Amount,
			Type:          uint32(token.Type),
			TokenMetaData: getTokenMetaData(token),
		}
		chainSim.GetEsdtInWallet(t, cs, wallet, token.Identifier, token.Nonce, tokenData)

		err = cs.GenerateBlocks(1)
		require.Nil(t, err)
		chainSim.RequireAccountHasToken(t, cs, getTokenIdentifier(token), wallet.Bech32, token.Amount)
	}

	// deposit an array of tokens from main chain to sovereign chain,
	// expecting these tokens to NOT be burned by ESDT safe sc because is not whitelisted
	txResult := deposit(t, cs, wallet.Bytes, &nonce, bridgeData.ESDTSafeAddress, bridgedOutTokens, wallet.Bytes)
	chainSim.RequireSignalError(t, txResult, actionNotAllowed)
}

func getTokenMetaData(token chainSim.ArgsDepositToken) *esdt.MetaData {
	if token.Type == core.Fungible {
		return nil
	}
	return &esdt.MetaData{
		Nonce:   token.Nonce,
		Creator: bytes.Repeat([]byte{1}, 32),
	}
}
