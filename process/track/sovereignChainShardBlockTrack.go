package track

import (
	"errors"
	"fmt"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/check"
	"github.com/multiversx/mx-chain-core-go/data"
	"github.com/multiversx/mx-chain-core-go/data/block"
	"github.com/multiversx/mx-chain-go/process"
)

type sovereignChainShardBlockTrack struct {
	*shardBlockTrack
	mainChainNotarizationStartRound uint64
	genesisBlockNonce               map[uint64]bool
}

// NewSovereignChainShardBlockTrack creates an object for tracking the received shard blocks
func NewSovereignChainShardBlockTrack(shardBlockTrack *shardBlockTrack, mainChainNotarizationStartRound uint64) (*sovereignChainShardBlockTrack, error) {
	if shardBlockTrack == nil {
		return nil, process.ErrNilBlockTracker
	}

	scsbt := &sovereignChainShardBlockTrack{
		shardBlockTrack:                 shardBlockTrack,
		mainChainNotarizationStartRound: mainChainNotarizationStartRound,
		genesisBlockNonce:               make(map[uint64]bool),
	}

	bp, ok := scsbt.blockProcessor.(*blockProcessor)
	if !ok {
		return nil, process.ErrWrongTypeAssertion
	}

	scbp, err := NewSovereignChainBlockProcessor(bp)
	if err != nil {
		return nil, err
	}

	scsbt.blockProcessor = scbp
	scsbt.doReceivedHeaderJobFunc = scsbt.doReceivedHeaderJob
	scsbt.getFinalHeaderFunc = scsbt.getFinalHeader

	err = scsbt.initCrossNotarizedStartHeaders()
	if err != nil {
		return nil, err
	}

	return scsbt, nil
}

func (scsbt *sovereignChainShardBlockTrack) initCrossNotarizedStartHeaders() error {
	// TODO MX-15667:
	// 1. Versioning header for shard extended?
	// 2. Perhaps add dummy header nonce from config cross main chain notarized
	extendedShardHeader := &block.ShardHeaderExtended{
		Header: &block.HeaderV2{
			Header: &block.Header{
				Reserved: []byte("dummySovHdr"),
			},
		},
	}

	extendedShardHeaderHash, err := core.CalculateHash(scsbt.marshalizer, scsbt.hasher, extendedShardHeader)
	if err != nil {
		return fmt.Errorf("%w in sovereignChainShardBlockTrack.initCrossNotarizedStartHeaders", err)
	}

	scsbt.AddCrossNotarizedHeader(core.MainChainShardId, extendedShardHeader, extendedShardHeaderHash)
	return nil
}

// ComputeLongestSelfChain computes the longest chain from self shard
func (scsbt *sovereignChainShardBlockTrack) ComputeLongestSelfChain() (data.HeaderHandler, []byte, []data.HeaderHandler, [][]byte) {
	lastSelfNotarizedHeader, lastSelfNotarizedHeaderHash, err := scsbt.selfNotarizer.GetLastNotarizedHeader(scsbt.shardCoordinator.SelfId())
	if err != nil {
		log.Warn("ComputeLongestSelfChain.GetLastNotarizedHeader", "error", err.Error())
		return nil, nil, nil, nil
	}

	headers, hashes := scsbt.ComputeLongestChain(scsbt.shardCoordinator.SelfId(), lastSelfNotarizedHeader)
	return lastSelfNotarizedHeader, lastSelfNotarizedHeaderHash, headers, hashes
}

// GetSelfNotarizedHeader returns a self notarized header for self shard with a given offset, behind last self notarized header
func (scsbt *sovereignChainShardBlockTrack) GetSelfNotarizedHeader(_ uint32, offset uint64) (data.HeaderHandler, []byte, error) {
	return scsbt.selfNotarizer.GetNotarizedHeader(scsbt.shardCoordinator.SelfId(), offset)
}

func (scsbt *sovereignChainShardBlockTrack) doReceivedHeaderJob(headerHandler data.HeaderHandler, headerHash []byte) {
	extendedShardHeader, isExtendedShardHeaderReceived := headerHandler.(*block.ShardHeaderExtended)
	if isExtendedShardHeaderReceived {
		scsbt.receivedExtendedShardHeader(extendedShardHeader, headerHash)
		return
	}

	scsbt.receivedShardHeader(headerHandler, headerHash)
}

func (scsbt *sovereignChainShardBlockTrack) receivedExtendedShardHeader(
	extendedShardHeaderHandler data.ShardHeaderExtendedHandler,
	extendedShardHeaderHash []byte,
) {

	log.Debug("received extended shard header from network in block tracker",
		"shard", extendedShardHeaderHandler.GetShardID(),
		"epoch", extendedShardHeaderHandler.GetEpoch(),
		"round", extendedShardHeaderHandler.GetRound(),
		"nonce", extendedShardHeaderHandler.GetNonce(),
		"hash", extendedShardHeaderHash,
	)

	wasGenesisFirsHeaderAdded := false
	// TODO: This condition will permit to the sovereign chain to follow the main chain headers starting with a header
	// having a nonce higher than nonce 1 (the first block after genesis)
	if scsbt.isGenesisLastCrossNotarizedHeader() {
		//startHdr := map[uint32]data.HeaderHandler{
		//	core.MainChainShardId: extendedShardHeaderHandler,
		//}

		log.Error("ADDED GENESIS HEADER")

		//err := scsbt.crossNotarizer.InitNotarizedHeaders(startHdr)
		//if err != nil {
		//	log.Error("sovereignChainShardBlockTrack.receivedExtendedShardHeader.InitNotarizedHeaders", "error", err)
		//	return
		//}
		//scsbt.crossNotarizer.RemoveLastNotarizedHeader()
		//err := scsbt.initCrossNotarizedStartHeaders()
		//if err != nil {
		//	log.Error("WTF WTF WTF WTF")
		//}
		scsbt.crossNotarizer.AddNotarizedHeader(core.MainChainShardId, extendedShardHeaderHandler, extendedShardHeaderHash)
		wasGenesisFirsHeaderAdded = true
	}

	if !scsbt.shouldAddExtendedShardHeader(extendedShardHeaderHandler) {
		log.Error("received extended shard header is out of range", "nonce", extendedShardHeaderHandler.GetNonce())
		return
	}

	if !scsbt.addHeader(extendedShardHeaderHandler, extendedShardHeaderHash, core.MainChainShardId) {
		log.Error("received extended shard header was not added", "nonce", extendedShardHeaderHandler.GetNonce())
		return
	}

	scsbt.doWhitelistWithExtendedShardHeaderIfNeeded(extendedShardHeaderHandler)
	scsbt.blockProcessor.ProcessReceivedHeader(extendedShardHeaderHandler)

	if wasGenesisFirsHeaderAdded {
		scsbt.genesisBlockNonce[extendedShardHeaderHandler.GetRound()] = true
	}
}

func (scsbt *sovereignChainShardBlockTrack) IsLastNotarizedHeaderGenesisWhileSyncing() (uint64, bool) {
	lastNotarizedHeader, _, err := scsbt.crossNotarizer.GetLastNotarizedHeader(core.MainChainShardId)
	if err != nil {
		return 0, false
	}

	if len(scsbt.genesisBlockNonce) == 0 {
		return 0, false
	}

	isLastNotarizedBlockGenesis := scsbt.genesisBlockNonce[lastNotarizedHeader.GetRound()]
	shouldSyncAtLeastOneMoreBlock := lastNotarizedHeader.GetRound() > scsbt.mainChainNotarizationStartRound

	log.Error("IsLastNotarizedHeaderGenesisWhileSyncing",
		"isLastNotarizedBlockGenesis", isLastNotarizedBlockGenesis,
		"shouldSyncAtLeastOneMoreBlock", shouldSyncAtLeastOneMoreBlock,
		"result", isLastNotarizedBlockGenesis && shouldSyncAtLeastOneMoreBlock,
	)

	scsbt.isGenesisLastCrossNotarizedHeader()

	return lastNotarizedHeader.GetNonce() - 1, isLastNotarizedBlockGenesis && shouldSyncAtLeastOneMoreBlock
}

func (scsbt *sovereignChainShardBlockTrack) IsGenesisLastCrossNotarizedHeader() bool {
	return scsbt.isGenesisLastCrossNotarizedHeader()
}

func (scsbt *sovereignChainShardBlockTrack) isGenesisLastCrossNotarizedHeader() bool {
	lastNotarizedHeader, _, err := scsbt.crossNotarizer.GetLastNotarizedHeader(core.MainChainShardId)

	log.Error("isGenesisLastCrossNotarizedHeader",
		"lastNotarizedHeader.Nonce", lastNotarizedHeader.GetNonce(),
		"error", err,
	)

	isGenesisLastCrossNotarizedHeader := err != nil && errors.Is(err, process.ErrNotarizedHeadersSliceForShardIsNil) ||
		lastNotarizedHeader != nil && lastNotarizedHeader.GetNonce() == 0

	return isGenesisLastCrossNotarizedHeader
}

func (scsbt *sovereignChainShardBlockTrack) shouldAddExtendedShardHeader(extendedShardHeaderHandler data.ShardHeaderExtendedHandler) bool {
	lastNotarizedHeader, _, err := scsbt.crossNotarizer.GetLastNotarizedHeader(core.MainChainShardId)
	if err != nil {
		log.Debug("shouldAddExtendedShardHeader.GetLastNotarizedHeader",
			"shard", extendedShardHeaderHandler.GetShardID(),
			"error", err.Error())
		return false
	}

	lastNotarizedHeaderNonce := lastNotarizedHeader.GetNonce()

	isHeaderOutOfRange := extendedShardHeaderHandler.GetNonce() > lastNotarizedHeaderNonce+uint64(scsbt.maxNumHeadersToKeepPerShard)
	return !isHeaderOutOfRange
}

func (scsbt *sovereignChainShardBlockTrack) doWhitelistWithExtendedShardHeaderIfNeeded(extendedShardHeaderHandler data.ShardHeaderExtendedHandler) {
	if check.IfNil(extendedShardHeaderHandler) {
		return
	}
	if scsbt.isExtendedShardHeaderOutOfRange(extendedShardHeaderHandler) {
		return
	}

	miniBlockHandlers := extendedShardHeaderHandler.GetIncomingMiniBlockHandlers()

	keys := make([][]byte, 0)
	for _, miniBlockHandler := range miniBlockHandlers {
		miniBlock, ok := miniBlockHandler.(*block.MiniBlock)
		if !ok {
			log.Warn("doWhitelistWithExtendedShardHeaderIfNeeded", "error", process.ErrWrongTypeAssertion)
			continue
		}

		keys = append(keys, miniBlock.TxHashes...)
	}

	scsbt.whitelistHandler.Add(keys)
}

func (scsbt *sovereignChainShardBlockTrack) isExtendedShardHeaderOutOfRange(extendedShardHeaderHandler data.ShardHeaderExtendedHandler) bool {
	lastCrossNotarizedHeader, _, err := scsbt.GetLastCrossNotarizedHeader(core.MainChainShardId)
	if err != nil {
		log.Debug("isExtendedShardHeaderOutOfRange.GetLastCrossNotarizedHeader",
			"shard", extendedShardHeaderHandler.GetShardID(),
			"error", err.Error())
		return true
	}

	isExtendedShardHeaderOutOfRange := extendedShardHeaderHandler.GetNonce() > lastCrossNotarizedHeader.GetNonce()+process.MaxHeadersToWhitelistInAdvance
	return isExtendedShardHeaderOutOfRange
}

// ComputeLongestExtendedShardChainFromLastNotarized returns the longest valid chain for extended shard chain from its last cross notarized header
func (scsbt *sovereignChainShardBlockTrack) ComputeLongestExtendedShardChainFromLastNotarized() ([]data.HeaderHandler, [][]byte, error) {
	lastCrossNotarizedHeader, lastCrossNotarizedHeaderHash, err := scsbt.GetLastCrossNotarizedHeader(core.MainChainShardId)
	if err != nil {
		return nil, nil, err
	}

	log.Error("sovereignChainShardBlockTrack.ComputeLongestExtendedShardChainFromLastNotarized: GetLastCrossNotarizedHeader", "nonce", lastCrossNotarizedHeader.GetNonce())

	hdrsForShard, hdrsHashesForShard := scsbt.ComputeLongestChain(core.MainChainShardId, lastCrossNotarizedHeader)

	log.Debug("sovereignChainShardBlockTrack.ComputeLongestExtendedShardChainFromLastNotarized: ComputeLongestChain", "num headers", len(hdrsForShard))
	for index := range hdrsForShard {
		log.Debug("sovereignChainShardBlockTrack.ComputeLongestExtendedShardChainFromLastNotarized", "round", hdrsForShard[index].GetRound(), "nonce", hdrsForShard[index].GetNonce(), "hash", hdrsHashesForShard[index])
	}

	//res := make([]data.HeaderHandler, 0)
	//res = append(res, lastCrossNotarizedHeader)
	//
	//resHashes := make([][]byte, 0)
	//resHashes = append(resHashes, lastCrossNotarizedHeaderHash)

	//\if len(hdrsForShard) == 0 && lastCrossNotarizedHeader.GetRound() == scsbt.mainChainNotarizationStartRound {
	//\	return res, resHashes, nil
	//\}

	//res := make([]data.HeaderHandler, 0)
	//res = append(res, lastCrossNotarizedHeader)
	//res = append(res, hdrsForShard...)
	//
	//resHashes := make([][]byte, 0)
	//resHashes = append(resHashes, lastCrossNotarizedHeaderHash)
	//resHashes = append(resHashes, hdrsHashesForShard...)

	//return res, resHashes, nil

	_ = lastCrossNotarizedHeaderHash

	//if len(hdrsForShard) > 1 && bytes.Equal(lastCrossNotarizedHeaderHash, hdrsHashesForShard[0]) {
	//	return hdrsForShard[1:], hdrsHashesForShard[1:], nil
	//}

	//if lastCrossNotarizedHeader.GetRound() == scsbt.mainChainNotarizationStartRound && len(hdrsForShard) != 0 {
	//	res := make([]data.HeaderHandler, 0)
	//	res = append(res, lastCrossNotarizedHeader)
	//	res = append(res, hdrsForShard...)
	//
	//	resHashes := make([][]byte, 0)
	//	resHashes = append(resHashes, lastCrossNotarizedHeaderHash)
	//	resHashes = append(resHashes, hdrsHashesForShard...)
	//
	//	log.Error("SPECIAL CASE")
	//
	//	return res, resHashes, nil
	//}

	return hdrsForShard, hdrsHashesForShard, nil
}

// CleanupHeadersBehindNonce removes from local pools old headers
func (scsbt *sovereignChainShardBlockTrack) CleanupHeadersBehindNonce(
	shardID uint32,
	selfNotarizedNonce uint64,
	crossNotarizedNonce uint64,
) {
	scsbt.selfNotarizer.CleanupNotarizedHeadersBehindNonce(shardID, selfNotarizedNonce)
	scsbt.cleanupTrackedHeadersBehindNonce(shardID, selfNotarizedNonce)

	//if shardID == core.MainChainShardId {
	//	scsbt.crossNotarizer.RemoveLastNotarizedHeader()
	//}

	//log.Error("sovereignChainShardBlockTrack.CleanupHeadersBehindNonce", "shardID", shardID, "crossNotarizedNonce", crossNotarizedNonce, "ACTUALLY", hdr.GetNonce())

	scsbt.crossNotarizer.CleanupNotarizedHeadersBehindNonce(core.MainChainShardId, crossNotarizedNonce)
	scsbt.cleanupTrackedHeadersBehindNonce(core.MainChainShardId, crossNotarizedNonce)
}

// DisplayTrackedHeaders displays tracked headers
func (scsbt *sovereignChainShardBlockTrack) DisplayTrackedHeaders() {
	scsbt.displayTrackedHeadersForShard(scsbt.shardCoordinator.SelfId(), "tracked headers")
	scsbt.selfNotarizer.DisplayNotarizedHeaders(scsbt.shardCoordinator.SelfId(), "self notarized headers")

	scsbt.displayTrackedHeadersForShard(core.MainChainShardId, "cross tracked headers")
	scsbt.crossNotarizer.DisplayNotarizedHeaders(core.MainChainShardId, "cross notarized headers")
}

func (scsbt *sovereignChainShardBlockTrack) getFinalHeader(headerHandler data.HeaderHandler) (data.HeaderHandler, error) {
	shardID := headerHandler.GetShardID()

	_, isExtendedShardHeaderReceived := headerHandler.(*block.ShardHeaderExtended)
	if isExtendedShardHeaderReceived {
		shardID = core.MainChainShardId
	}

	finalHeader, _, err := scsbt.getFinalHeaderForShard(shardID)
	if err != nil {
		return nil, err
	}

	return finalHeader, nil
}

// ShouldSkipMiniBlocksCreationFromSelf returns false for sovereign chain
func (scsbt *sovereignChainShardBlockTrack) ShouldSkipMiniBlocksCreationFromSelf() bool {
	return false
}

// IsShardStuck returns false for sovereign chain
func (scsbt *sovereignChainShardBlockTrack) IsShardStuck(_ uint32) bool {
	return false
}

// ComputeCrossInfo does nothing for sovereign chain
func (scsbt *sovereignChainShardBlockTrack) ComputeCrossInfo(_ []data.HeaderHandler) {
}
