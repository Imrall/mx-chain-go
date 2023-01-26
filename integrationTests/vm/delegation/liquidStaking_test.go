//go:build !race
// +build !race

package delegation

import (
	"bytes"
	"math/big"
	"testing"
	"time"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-go/integrationTests"
	"github.com/multiversx/mx-chain-go/integrationTests/vm/esdt"
	"github.com/multiversx/mx-chain-go/testscommon/txDataBuilder"
	"github.com/multiversx/mx-chain-go/vm"
	logger "github.com/multiversx/mx-chain-logger-go"
	vmcommon "github.com/multiversx/mx-chain-vm-common-go"
	"github.com/stretchr/testify/require"
)

var log = logger.GetOrCreate("liquidStaking")

func TestDelegationSystemSCWithLiquidStaking(t *testing.T) {
	t.Skip("this test seems to be incompatible with later flags;" +
		"since liquid staking will be most likely used on RUST SC and not on protocol level, we will be disable this test")

	if testing.Short() {
		t.Skip("this is not a short test")
	}

	nodes, idxProposers, delegationAddress, tokenID, nonce, round := setupNodesDelegationContractInitLiquidStaking(t)
	defer func() {
		for _, n := range nodes {
			_ = n.Messenger.Close()
		}
	}()

	txData := txDataBuilder.NewBuilder().Clear().
		Func("claimDelegatedPosition").
		Bytes(big.NewInt(1).Bytes()).
		Bytes(delegationAddress).
		Bytes(big.NewInt(5000).Bytes()).
		ToString()
	for _, node := range nodes {
		integrationTests.CreateAndSendTransaction(node, nodes, big.NewInt(0), vm.LiquidStakingSCAddress, txData, core.MinMetaTxExtraGasCost)
	}

	nrRoundsToPropagateMultiShard := 12
	time.Sleep(time.Second)
	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)
	time.Sleep(time.Second)

	// claim again
	for _, node := range nodes {
		integrationTests.CreateAndSendTransaction(node, nodes, big.NewInt(0), vm.LiquidStakingSCAddress, txData, core.MinMetaTxExtraGasCost)
	}

	time.Sleep(time.Second)
	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)
	time.Sleep(time.Second)

	for i := 1; i < len(nodes); i++ {
		checkLPPosition(t, nodes[i].OwnAccount.Address, nodes, tokenID, uint64(1), big.NewInt(10000))
	}
	// owner is not allowed to get LP position
	checkLPPosition(t, nodes[0].OwnAccount.Address, nodes, tokenID, uint64(1), big.NewInt(0))
	metaNode := getNodeWithShardID(nodes, core.MetachainShardId)
	allDelegatorAddresses := make([][]byte, 0)
	for i := 1; i < len(nodes); i++ {
		allDelegatorAddresses = append(allDelegatorAddresses, nodes[i].OwnAccount.Address)
	}
	verifyDelegatorIsDeleted(t, metaNode, allDelegatorAddresses, delegationAddress)

	oneTransfer := &vmcommon.ESDTTransfer{
		ESDTValue:      big.NewInt(1000),
		ESDTTokenName:  tokenID,
		ESDTTokenType:  uint32(core.NonFungible),
		ESDTTokenNonce: 1,
	}
	esdtTransfers := []*vmcommon.ESDTTransfer{oneTransfer, oneTransfer, oneTransfer, oneTransfer, oneTransfer}
	txBuilder := txDataBuilder.NewBuilder().MultiTransferESDTNFT(vm.LiquidStakingSCAddress, esdtTransfers)
	txBuilder.Bytes([]byte("unDelegatePosition"))
	for _, node := range nodes {
		integrationTests.CreateAndSendTransaction(node, nodes, big.NewInt(0), node.OwnAccount.Address, txBuilder.ToString(), core.MinMetaTxExtraGasCost)
	}

	txBuilder = txDataBuilder.NewBuilder().MultiTransferESDTNFT(vm.LiquidStakingSCAddress, esdtTransfers)
	txBuilder.Bytes([]byte("returnPosition"))
	for _, node := range nodes {
		integrationTests.CreateAndSendTransaction(node, nodes, big.NewInt(0), node.OwnAccount.Address, txBuilder.ToString(), core.MinMetaTxExtraGasCost)
	}
	time.Sleep(time.Second)
	finalWait := 20
	_, _ = integrationTests.WaitOperationToBeDone(t, nodes, finalWait, nonce, round, idxProposers)
	time.Sleep(time.Second)

	for _, node := range nodes {
		checkLPPosition(t, node.OwnAccount.Address, nodes, tokenID, uint64(1), big.NewInt(0))
	}

	verifyDelegatorsStake(t, metaNode, "getUserActiveStake", allDelegatorAddresses, delegationAddress, big.NewInt(5000))
	verifyDelegatorsStake(t, metaNode, "getUserUnStakedValue", allDelegatorAddresses, delegationAddress, big.NewInt(5000))
}

func setupNodesDelegationContractInitLiquidStaking(
	t *testing.T,
) ([]*integrationTests.TestProcessorNode, []int, []byte, []byte, uint64, uint64) {
	numOfShards := 2
	nodesPerShard := 2
	numMetachainNodes := 2

	nodes := integrationTests.CreateNodes(
		numOfShards,
		nodesPerShard,
		numMetachainNodes,
	)

	integrationTests.DisplayAndStartNodes(nodes)

	idxProposers := make([]int, numOfShards+1)
	for i := 0; i < numOfShards; i++ {
		idxProposers[i] = i * nodesPerShard
	}
	idxProposers[numOfShards] = numOfShards * nodesPerShard

	tokenID := initDelegationManagementAndLiquidStaking(nodes)

	initialVal := big.NewInt(10000000000)
	initialVal.Mul(initialVal, initialVal)
	integrationTests.MintAllNodes(nodes, initialVal)

	delegationAddress := createNewDelegationSystemSC(nodes[0], nodes)

	round := uint64(0)
	nonce := uint64(0)
	round = integrationTests.IncrementAndPrintRound(round)
	nonce++

	time.Sleep(time.Second)
	nrRoundsToPropagateMultiShard := 6
	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)
	time.Sleep(time.Second)

	txData := "delegate"
	for _, node := range nodes {
		integrationTests.CreateAndSendTransaction(node, nodes, big.NewInt(10000), delegationAddress, txData, core.MinMetaTxExtraGasCost)
	}

	time.Sleep(time.Second)
	nonce, round = integrationTests.WaitOperationToBeDone(t, nodes, nrRoundsToPropagateMultiShard, nonce, round, idxProposers)
	time.Sleep(time.Second)

	return nodes, idxProposers, delegationAddress, tokenID, nonce, round
}

func initDelegationManagementAndLiquidStaking(nodes []*integrationTests.TestProcessorNode) []byte {
	var tokenID []byte
	for _, node := range nodes {
		node.InitDelegationManager()
		tmpTokenID := node.InitLiquidStaking()
		if len(tmpTokenID) != 0 {
			if len(tokenID) == 0 {
				tokenID = tmpTokenID
			}

			if !bytes.Equal(tokenID, tmpTokenID) {
				log.Error("tokenID missmatch", "current", tmpTokenID, "old", tokenID)
			}
		}
	}
	return tokenID
}

func checkLPPosition(
	t *testing.T,
	address []byte,
	nodes []*integrationTests.TestProcessorNode,
	tokenID []byte,
	nonce uint64,
	value *big.Int,
) {
	esdtData := esdt.GetESDTTokenData(t, address, nodes, tokenID, nonce)

	if value.Cmp(big.NewInt(0)) == 0 {
		require.Nil(t, esdtData.TokenMetaData)
		return
	}

	require.NotNil(t, esdtData.TokenMetaData)
	require.Equal(t, vm.LiquidStakingSCAddress, esdtData.TokenMetaData.Creator)
	require.Equal(t, value.Bytes(), esdtData.Value.Bytes())
}
