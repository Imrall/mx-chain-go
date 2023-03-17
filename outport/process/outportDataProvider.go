package process

import (
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/check"
	"github.com/multiversx/mx-chain-core-go/data"
	"github.com/multiversx/mx-chain-core-go/data/block"
	outportcore "github.com/multiversx/mx-chain-core-go/data/outport"
	"github.com/multiversx/mx-chain-core-go/data/receipt"
	"github.com/multiversx/mx-chain-core-go/data/rewardTx"
	"github.com/multiversx/mx-chain-core-go/data/smartContractResult"
	"github.com/multiversx/mx-chain-core-go/data/transaction"
	"github.com/multiversx/mx-chain-core-go/marshal"
	"github.com/multiversx/mx-chain-go/outport/process/alteredaccounts/shared"
	"github.com/multiversx/mx-chain-go/process"
	"github.com/multiversx/mx-chain-go/sharding"
	"github.com/multiversx/mx-chain-go/sharding/nodesCoordinator"
)

// ArgOutportDataProvider holds the arguments needed for creating a new instance of outportDataProvider
type ArgOutportDataProvider struct {
	IsImportDBMode           bool
	ShardCoordinator         sharding.Coordinator
	AlteredAccountsProvider  AlteredAccountsProviderHandler
	TransactionsFeeProcessor TransactionsFeeHandler
	TxCoordinator            process.TransactionCoordinator
	NodesCoordinator         nodesCoordinator.NodesCoordinator
	GasConsumedProvider      GasConsumedProvider
	EconomicsData            EconomicsDataHandler
	ExecutionOrderHandler    ExecutionOrderHandler
	Marshaller               marshal.Marshalizer
}

// ArgPrepareOutportSaveBlockData holds the arguments needed for prepare outport save block data
type ArgPrepareOutportSaveBlockData struct {
	HeaderHash             []byte
	Header                 data.HeaderHandler
	HeaderBytes            []byte
	HeaderType             string
	Body                   data.BodyHandler
	PreviousHeader         data.HeaderHandler
	RewardsTxs             map[string]data.TransactionHandler
	NotarizedHeadersHashes []string
	HighestFinalBlockNonce uint64
	HighestFinalBlockHash  []byte
}

type outportDataProvider struct {
	isImportDBMode           bool
	shardID                  uint32
	numOfShards              uint32
	alteredAccountsProvider  AlteredAccountsProviderHandler
	transactionsFeeProcessor TransactionsFeeHandler
	txCoordinator            process.TransactionCoordinator
	nodesCoordinator         nodesCoordinator.NodesCoordinator
	gasConsumedProvider      GasConsumedProvider
	economicsData            EconomicsDataHandler
	executionOrderHandler    ExecutionOrderHandler
	marshaller               marshal.Marshalizer
}

// NewOutportDataProvider will create a new instance of outportDataProvider
func NewOutportDataProvider(arg ArgOutportDataProvider) (*outportDataProvider, error) {
	return &outportDataProvider{
		shardID:                  arg.ShardCoordinator.SelfId(),
		numOfShards:              arg.ShardCoordinator.NumberOfShards(),
		alteredAccountsProvider:  arg.AlteredAccountsProvider,
		transactionsFeeProcessor: arg.TransactionsFeeProcessor,
		txCoordinator:            arg.TxCoordinator,
		nodesCoordinator:         arg.NodesCoordinator,
		gasConsumedProvider:      arg.GasConsumedProvider,
		economicsData:            arg.EconomicsData,
		executionOrderHandler:    arg.ExecutionOrderHandler,
		marshaller:               arg.Marshaller,
	}, nil
}

// PrepareOutportSaveBlockData will prepare the provided data in a format that will be accepted by an outport driver
func (odp *outportDataProvider) PrepareOutportSaveBlockData(arg ArgPrepareOutportSaveBlockData) (*outportcore.OutportBlockWithHeaderAndBody, error) {
	if check.IfNil(arg.Header) {
		return nil, errNilHeaderHandler
	}
	if check.IfNil(arg.Body) {
		return nil, errNilBodyHandler
	}

	pool, err := odp.createPool(arg.RewardsTxs)
	if err != nil {
		return nil, err
	}

	err = odp.transactionsFeeProcessor.PutFeeAndGasUsed(pool)
	if err != nil {
		return nil, fmt.Errorf("transactionsFeeProcessor.PutFeeAndGasUsed %w", err)
	}

	scheduledExecutedSCRsHashesPrevBlock, scheduledExecutedInvalidTxsHashesPrevBlock, err := odp.executionOrderHandler.PutExecutionOrderInTransactionPool(pool, arg.Header, arg.Body, arg.PreviousHeader)
	if err != nil {
		return nil, fmt.Errorf("executionOrderHandler.PutExecutionOrderInTransactionPool %w", err)
	}

	pool.ScheduledExecutedInvalidTxsHashesPrevBlock = scheduledExecutedInvalidTxsHashesPrevBlock
	pool.ScheduledExecutedSCRSHashesPrevBlock = scheduledExecutedSCRsHashesPrevBlock

	alteredAccounts, err := odp.alteredAccountsProvider.ExtractAlteredAccountsFromPool(pool, shared.AlteredAccountsOptions{
		WithAdditionalOutportData: true,
	})
	if err != nil {
		return nil, fmt.Errorf("alteredAccountsProvider.ExtractAlteredAccountsFromPool %s", err)
	}

	signersIndexes, err := odp.getSignersIndexes(arg.Header)
	if err != nil {
		return nil, err
	}

	return &outportcore.OutportBlockWithHeaderAndBody{
		OutportBlock: &outportcore.OutportBlock{
			BlockData:       nil, // this will be filed with specific data for each driver
			TransactionPool: pool,
			HeaderGasConsumption: &outportcore.HeaderGasConsumption{
				GasProvided:    odp.gasConsumedProvider.TotalGasProvidedWithScheduled(),
				GasRefunded:    odp.gasConsumedProvider.TotalGasRefunded(),
				GasPenalized:   odp.gasConsumedProvider.TotalGasPenalized(),
				MaxGasPerBlock: odp.economicsData.MaxGasLimitPerBlock(odp.shardID),
			},
			AlteredAccounts:        alteredAccounts,
			NotarizedHeadersHashes: arg.NotarizedHeadersHashes,
			NumberOfShards:         odp.numOfShards,
			IsImportDB:             odp.isImportDBMode,
			SignersIndexes:         signersIndexes,

			HighestFinalBlockNonce: arg.HighestFinalBlockNonce,
			HighestFinalBlockHash:  arg.HighestFinalBlockHash,
		},
		HeaderDataWithBody: &outportcore.HeaderDataWithBody{
			Body:       arg.Body,
			Header:     arg.Header,
			HeaderHash: arg.HeaderHash,
		},
	}, nil
}

func (odp *outportDataProvider) computeEpoch(header data.HeaderHandler) uint32 {
	epoch := header.GetEpoch()
	shouldDecreaseEpoch := header.IsStartOfEpochBlock() && epoch > 0 && odp.shardID != core.MetachainShardId
	if shouldDecreaseEpoch {
		epoch--
	}

	return epoch
}

func (odp *outportDataProvider) getSignersIndexes(header data.HeaderHandler) ([]uint64, error) {
	epoch := odp.computeEpoch(header)
	pubKeys, err := odp.nodesCoordinator.GetConsensusValidatorsPublicKeys(
		header.GetPrevRandSeed(),
		header.GetRound(),
		odp.shardID,
		epoch,
	)
	if err != nil {
		return nil, fmt.Errorf("nodesCoordinator.GetConsensusValidatorsPublicKeys %w", err)
	}

	signersIndexes, err := odp.nodesCoordinator.GetValidatorsIndexes(pubKeys, epoch)
	if err != nil {
		return nil, fmt.Errorf("nodesCoordinator.GetValidatorsIndexes %s", err)
	}

	return signersIndexes, nil
}

func (odp *outportDataProvider) createPool(rewardsTxs map[string]data.TransactionHandler) (*outportcore.TransactionPool, error) {
	if odp.shardID == core.MetachainShardId {
		return odp.createPoolForMeta(rewardsTxs)
	}

	return odp.createPoolForShard()
}

func (odp *outportDataProvider) createPoolForShard() (*outportcore.TransactionPool, error) {
	txs, err := getTxs(odp.txCoordinator.GetAllCurrentUsedTxs(block.TxBlock))
	if err != nil {
		return nil, err
	}

	scrs, err := getScrs(odp.txCoordinator.GetAllCurrentUsedTxs(block.SmartContractResultBlock))
	if err != nil {
		return nil, err
	}

	rewards, err := getRewards(odp.txCoordinator.GetAllCurrentUsedTxs(block.RewardsBlock))
	if err != nil {
		return nil, err
	}

	invalidTxs, err := getTxs(odp.txCoordinator.GetAllCurrentUsedTxs(block.InvalidBlock))
	if err != nil {
		return nil, err
	}

	receipts, err := getReceipts(odp.txCoordinator.GetAllCurrentUsedTxs(block.ReceiptBlock))
	if err != nil {
		return nil, err
	}

	logs, err := getLogs(odp.txCoordinator.GetAllCurrentLogs())
	if err != nil {
		return nil, err
	}

	return &outportcore.TransactionPool{
		Transactions:         txs,
		SmartContractResults: scrs,
		Rewards:              rewards,
		InvalidTxs:           invalidTxs,
		Receipts:             receipts,
		Logs:                 logs,
	}, nil
}

func (odp *outportDataProvider) createPoolForMeta(rewardsTxs map[string]data.TransactionHandler) (*outportcore.TransactionPool, error) {
	txs, err := getTxs(odp.txCoordinator.GetAllCurrentUsedTxs(block.TxBlock))
	if err != nil {
		return nil, err
	}

	scrs, err := getScrs(odp.txCoordinator.GetAllCurrentUsedTxs(block.SmartContractResultBlock))
	if err != nil {
		return nil, err
	}

	rewards, err := getRewards(rewardsTxs)
	if err != nil {
		return nil, err
	}

	logs, err := getLogs(odp.txCoordinator.GetAllCurrentLogs())
	if err != nil {
		return nil, err
	}

	return &outportcore.TransactionPool{
		Transactions:         txs,
		SmartContractResults: scrs,
		Rewards:              rewards,
		Logs:                 logs,
	}, nil
}

func getTxs(txs map[string]data.TransactionHandler) (map[string]*outportcore.TxInfo, error) {
	ret := make(map[string]*outportcore.TxInfo, len(txs))

	for txHash, txHandler := range txs {
		tx, castOk := txHandler.(*transaction.Transaction)
		txHashHex := getHexEncodedHash(txHash)
		if !castOk {
			return nil, fmt.Errorf("%w, hash: %s", errCannotCastTransaction, txHashHex)
		}

		ret[txHashHex] = &outportcore.TxInfo{
			Transaction: tx,
			FeeInfo:     newFeeInfo(),
		}
	}

	return ret, nil
}

func getHexEncodedHash(txHash string) string {
	txHashBytes := []byte(txHash)
	return hex.EncodeToString(txHashBytes)
}

func newFeeInfo() *outportcore.FeeInfo {
	return &outportcore.FeeInfo{
		GasUsed:        0,
		Fee:            big.NewInt(0),
		InitialPaidFee: big.NewInt(0),
	}
}

func getScrs(scrs map[string]data.TransactionHandler) (map[string]*outportcore.SCRInfo, error) {
	ret := make(map[string]*outportcore.SCRInfo, len(scrs))

	for scrHash, txHandler := range scrs {
		scr, castOk := txHandler.(*smartContractResult.SmartContractResult)
		scrHashHex := getHexEncodedHash(scrHash)
		if !castOk {
			return nil, fmt.Errorf("%w, hash: %s", errCannotCastSCR, scrHashHex)
		}

		ret[scrHashHex] = &outportcore.SCRInfo{
			SmartContractResult: scr,
			FeeInfo:             newFeeInfo(),
		}
	}

	return ret, nil
}

func getRewards(rewards map[string]data.TransactionHandler) (map[string]*outportcore.RewardInfo, error) {
	ret := make(map[string]*outportcore.RewardInfo, len(rewards))

	for hash, txHandler := range rewards {
		reward, castOk := txHandler.(*rewardTx.RewardTx)
		hexHex := getHexEncodedHash(hash)
		if !castOk {
			return nil, fmt.Errorf("%w, hash: %s", errCannotCastReward, hexHex)
		}

		ret[hexHex] = &outportcore.RewardInfo{
			Reward: reward,
		}
	}

	return ret, nil
}

func getReceipts(receipts map[string]data.TransactionHandler) (map[string]*receipt.Receipt, error) {
	ret := make(map[string]*receipt.Receipt, len(receipts))

	for hash, receiptHandler := range receipts {
		tx, castOk := receiptHandler.(*receipt.Receipt)
		hashHex := getHexEncodedHash(hash)
		if !castOk {
			return nil, fmt.Errorf("%w, hash: %s", errCannotCastReceipt, hashHex)
		}

		ret[hashHex] = tx
	}

	return ret, nil
}

func getLogs(logs []*data.LogData) (map[string]*transaction.Log, error) {
	ret := make(map[string]*transaction.Log, len(logs))

	for _, logHandler := range logs {
		eventHandlers := logHandler.GetLogEvents()
		events, err := getEvents(eventHandlers)
		txHashHex := getHexEncodedHash(logHandler.TxHash)
		if err != nil {
			return nil, fmt.Errorf("%w, hash: %s", err, txHashHex)
		}

		ret[txHashHex] = &transaction.Log{
			Address: logHandler.GetAddress(),
			Events:  events,
		}
	}
	return ret, nil
}

func getEvents(eventHandlers []data.EventHandler) ([]*transaction.Event, error) {
	events := make([]*transaction.Event, len(eventHandlers))

	for idx, eventHandler := range eventHandlers {
		event, castOk := eventHandler.(*transaction.Event)
		if !castOk {
			return nil, errCannotCastEvent
		}

		events[idx] = event
	}

	return events, nil
}

// IsInterfaceNil returns true if there is no value under the interface
func (odp *outportDataProvider) IsInterfaceNil() bool {
	return odp == nil
}
