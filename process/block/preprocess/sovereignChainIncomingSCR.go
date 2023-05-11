package preprocess

import (
	"github.com/multiversx/mx-chain-core-go/core/check"
	"github.com/multiversx/mx-chain-core-go/data"
	"github.com/multiversx/mx-chain-core-go/data/block"
	"github.com/multiversx/mx-chain-core-go/data/smartContractResult"
	"github.com/multiversx/mx-chain-go/process"
)

type sovereignChainIncomingSCR struct {
	*smartContractResults
}

func onRequestIncomingSCR(_ uint32, txHashes [][]byte) {
	log.Warn("sovereignChainIncomingSCR.onRequestIncomingSCR was called; not implemented", "missing scrs hashes", txHashes)
}

func NewSovereignChainIncomingSCR(scr *smartContractResults) (*sovereignChainIncomingSCR, error) {
	if scr == nil {
		return nil, process.ErrNilPreProcessor
	}

	sovereignSCR := &sovereignChainIncomingSCR{
		scr,
	}

	sovereignSCR.onRequestSmartContractResult = onRequestIncomingSCR
	return sovereignSCR, nil
}

// ProcessBlockTransactions processes all the smartContractResult from the block.Body, updates the state
func (scr *sovereignChainIncomingSCR) ProcessBlockTransactions(
	headerHandler data.HeaderHandler,
	body *block.Body,
	haveTime func() bool,
) (block.MiniBlockSlice, error) {
	if check.IfNil(body) {
		return nil, process.ErrNilBlockBody
	}

	// TODO: Should we handle any gas? Since txs are already executed on main chain

	log.Info("sovereignChainIncomingSCR.ProcessBlockTransactions called")

	createdMBs := make(block.MiniBlockSlice, 0)
	// basic validation already done in interceptors
	for i := 0; i < len(body.MiniBlocks); i++ {
		miniBlock := body.MiniBlocks[i]
		if miniBlock.Type != block.SmartContractResultBlock {
			continue
		}
		// smart contract results are needed to be processed only at destination and only if they are cross shard
		if miniBlock.ReceiverShardID != scr.shardCoordinator.SelfId() {
			continue
		}
		if miniBlock.SenderShardID == scr.shardCoordinator.SelfId() {
			continue
		}

		pi, err := scr.getIndexesOfLastTxProcessed(miniBlock, headerHandler)
		if err != nil {
			return nil, err
		}

		indexOfFirstTxToBeProcessed := pi.indexOfLastTxProcessed + 1
		err = process.CheckIfIndexesAreOutOfBound(indexOfFirstTxToBeProcessed, pi.indexOfLastTxProcessedByProposer, miniBlock)
		if err != nil {
			return nil, err
		}

		for j := indexOfFirstTxToBeProcessed; j <= pi.indexOfLastTxProcessedByProposer; j++ {
			if !haveTime() {
				return nil, process.ErrTimeIsOut
			}

			txHash := miniBlock.TxHashes[j]
			scr.scrForBlock.mutTxsForBlock.RLock()
			txInfoFromMap, ok := scr.scrForBlock.txHashAndInfo[string(txHash)]
			scr.scrForBlock.mutTxsForBlock.RUnlock()
			if !ok || check.IfNil(txInfoFromMap.tx) {
				log.Warn("missing transaction in ProcessBlockTransactions ", "type", miniBlock.Type, "txHash", txHash)
				return nil, process.ErrMissingTransaction
			}

			currScr, ok := txInfoFromMap.tx.(*smartContractResult.SmartContractResult)
			if !ok {
				return nil, process.ErrWrongTypeAssertion
			}

			scr.saveAccountBalanceForAddress(currScr.GetRcvAddr())

			_, err := scr.scrProcessor.ProcessSmartContractResult(currScr)
			if err != nil {
				return nil, err
			}

		}

		createdMBs = append(createdMBs, miniBlock)
	}

	return createdMBs, nil
}

func (scr *sovereignChainIncomingSCR) ProcessMiniBlock(
	miniBlock *block.MiniBlock,
	haveTime func() bool,
	_ func() bool,
	_ bool,
	partialMbExecutionMode bool,
	indexOfLastTxProcessed int,
	preProcessorExecutionInfoHandler process.PreProcessorExecutionInfoHandler,
) ([][]byte, int, bool, error) {
	if miniBlock.Type != block.SmartContractResultBlock {
		return nil, indexOfLastTxProcessed, false, process.ErrWrongTypeInMiniBlock
	}

	numSCRsProcessed := 0
	var err error
	var txIndex int
	processedTxHashes := make([][]byte, 0)

	indexOfFirstTxToBeProcessed := indexOfLastTxProcessed + 1
	err = process.CheckIfIndexesAreOutOfBound(int32(indexOfFirstTxToBeProcessed), int32(len(miniBlock.TxHashes))-1, miniBlock)
	if err != nil {
		return nil, indexOfLastTxProcessed, false, err
	}

	miniBlockScrs, miniBlockTxHashes, err := scr.getAllScrsFromMiniBlock(miniBlock, haveTime)
	if err != nil {
		return nil, indexOfLastTxProcessed, false, err
	}

	if scr.blockSizeComputation.IsMaxBlockSizeWithoutThrottleReached(1, len(miniBlock.TxHashes)) {
		return nil, indexOfLastTxProcessed, false, process.ErrMaxBlockSizeReached
	}

	log.Debug("smartContractResults.ProcessMiniBlock: before processing")
	defer func() {
		log.Debug("smartContractResults.ProcessMiniBlock after processing")
	}()

	for txIndex = indexOfFirstTxToBeProcessed; txIndex < len(miniBlockScrs); txIndex++ {
		if !haveTime() {
			err = process.ErrTimeIsOut
			break
		}

		if err != nil {
			break
		}

		scr.saveAccountBalanceForAddress(miniBlockScrs[txIndex].GetRcvAddr())

		snapshot := scr.handleProcessTransactionInit(preProcessorExecutionInfoHandler, miniBlockTxHashes[txIndex])
		_, err = scr.scrProcessor.ProcessSmartContractResult(miniBlockScrs[txIndex])
		if err != nil {
			scr.handleProcessSovereignSCRError(preProcessorExecutionInfoHandler, snapshot, miniBlockTxHashes[txIndex])
			break
		}

		processedTxHashes = append(processedTxHashes, miniBlockTxHashes[txIndex])
		numSCRsProcessed++
	}

	if err != nil && !partialMbExecutionMode {
		return processedTxHashes, txIndex - 1, true, err
	}

	txShardInfoToSet := &txShardInfo{senderShardID: miniBlock.SenderShardID, receiverShardID: miniBlock.ReceiverShardID}

	scr.scrForBlock.mutTxsForBlock.Lock()
	for index, txHash := range miniBlockTxHashes {
		scr.scrForBlock.txHashAndInfo[string(txHash)] = &txInfo{tx: miniBlockScrs[index], txShardInfo: txShardInfoToSet}
	}
	scr.scrForBlock.mutTxsForBlock.Unlock()

	scr.blockSizeComputation.AddNumMiniBlocks(1)
	scr.blockSizeComputation.AddNumTxs(len(miniBlock.TxHashes))

	return nil, txIndex - 1, false, err
}

func (scr *sovereignChainIncomingSCR) handleProcessSovereignSCRError(preProcessorExecutionInfoHandler process.PreProcessorExecutionInfoHandler, snapshot int, txHash []byte) {
	errRevert := scr.accounts.RevertToSnapshot(snapshot)
	if errRevert != nil {
		log.Debug("scr.sovereignChainIncomingSCR: RevertToSnapshot", "error", errRevert.Error())
	}

	preProcessorExecutionInfoHandler.RevertProcessedTxsResults([][]byte{txHash}, txHash)
}
