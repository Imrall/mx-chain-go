package v2

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/check"
	"github.com/multiversx/mx-chain-core-go/data"
	"github.com/multiversx/mx-chain-core-go/data/block"
	"github.com/multiversx/mx-chain-core-go/display"

	"github.com/multiversx/mx-chain-go/common"
	"github.com/multiversx/mx-chain-go/consensus"
	"github.com/multiversx/mx-chain-go/consensus/spos"
	"github.com/multiversx/mx-chain-go/consensus/spos/bls"
	"github.com/multiversx/mx-chain-go/p2p"
	"github.com/multiversx/mx-chain-go/process/headerCheck"
)

const timeBetweenSignaturesChecks = time.Millisecond * 5

type subroundEndRound struct {
	*spos.Subround
	processingThresholdPercentage int
	appStatusHandler              core.AppStatusHandler
	mutProcessingEndRound         sync.Mutex
	sentSignatureTracker          spos.SentSignaturesTracker
	worker                        spos.WorkerHandler
	signatureThrottler            core.Throttler
}

// NewSubroundEndRound creates a subroundEndRound object
func NewSubroundEndRound(
	baseSubround *spos.Subround,
	processingThresholdPercentage int,
	appStatusHandler core.AppStatusHandler,
	sentSignatureTracker spos.SentSignaturesTracker,
	worker spos.WorkerHandler,
	signatureThrottler core.Throttler,
) (*subroundEndRound, error) {
	err := checkNewSubroundEndRoundParams(
		baseSubround,
	)
	if err != nil {
		return nil, err
	}
	if check.IfNil(appStatusHandler) {
		return nil, spos.ErrNilAppStatusHandler
	}
	if check.IfNil(sentSignatureTracker) {
		return nil, ErrNilSentSignatureTracker
	}
	if check.IfNil(worker) {
		return nil, spos.ErrNilWorker
	}
	if check.IfNil(signatureThrottler) {
		return nil, spos.ErrNilThrottler
	}

	srEndRound := subroundEndRound{
		Subround:                      baseSubround,
		processingThresholdPercentage: processingThresholdPercentage,
		appStatusHandler:              appStatusHandler,
		mutProcessingEndRound:         sync.Mutex{},
		sentSignatureTracker:          sentSignatureTracker,
		worker:                        worker,
		signatureThrottler:            signatureThrottler,
	}
	srEndRound.Job = srEndRound.doEndRoundJob
	srEndRound.Check = srEndRound.doEndRoundConsensusCheck
	srEndRound.Extend = worker.Extend

	return &srEndRound, nil
}

func checkNewSubroundEndRoundParams(
	baseSubround *spos.Subround,
) error {
	if baseSubround == nil {
		return spos.ErrNilSubround
	}
	if check.IfNil(baseSubround.ConsensusStateHandler) {
		return spos.ErrNilConsensusState
	}

	err := spos.ValidateConsensusCore(baseSubround.ConsensusCoreHandler)

	return err
}

// receivedBlockHeaderFinalInfo method is called when a block header final info is received
func (sr *subroundEndRound) receivedBlockHeaderFinalInfo(_ context.Context, cnsDta *consensus.Message) bool {
	sr.mutProcessingEndRound.Lock()
	defer sr.mutProcessingEndRound.Unlock()

	messageSender := string(cnsDta.PubKey)

	if !sr.IsConsensusDataSet() {
		return false
	}
	if check.IfNil(sr.GetHeader()) {
		return false
	}

	// TODO[cleanup cns finality]: remove if statement
	isSenderAllowed := sr.IsNodeInConsensusGroup(messageSender)
	if !sr.EnableEpochsHandler().IsFlagEnabledInEpoch(common.EquivalentMessagesFlag, sr.GetHeader().GetEpoch()) {
		isSenderAllowed = sr.IsNodeLeaderInCurrentRound(messageSender)
	}
	if !isSenderAllowed { // is NOT this node leader in current round?
		sr.PeerHonestyHandler().ChangeScore(
			messageSender,
			spos.GetConsensusTopicID(sr.ShardCoordinator()),
			spos.LeaderPeerHonestyDecreaseFactor,
		)

		return false
	}

	// TODO[cleanup cns finality]: remove if
	isSelfSender := sr.IsNodeSelf(messageSender) || sr.IsKeyManagedBySelf([]byte(messageSender))
	if !sr.EnableEpochsHandler().IsFlagEnabledInEpoch(common.EquivalentMessagesFlag, sr.GetHeader().GetEpoch()) {
		isSelfSender = sr.IsSelfLeader()
	}
	if isSelfSender {
		return false
	}

	if !sr.IsConsensusDataEqual(cnsDta.BlockHeaderHash) {
		return false
	}

	if !sr.CanProcessReceivedMessage(cnsDta, sr.RoundHandler().Index(), sr.Current()) {
		return false
	}

	hasProof := sr.EquivalentProofsPool().HasProof(sr.ShardCoordinator().SelfId(), cnsDta.BlockHeaderHash)
	if hasProof && sr.EnableEpochsHandler().IsFlagEnabledInEpoch(common.EquivalentMessagesFlag, sr.GetHeader().GetEpoch()) {
		return true
	}

	if !sr.isBlockHeaderFinalInfoValid(cnsDta) {
		return false
	}

	log.Debug("step 3: block header final info has been received",
		"PubKeysBitmap", cnsDta.PubKeysBitmap,
		"AggregateSignature", cnsDta.AggregateSignature,
		"LeaderSignature", cnsDta.LeaderSignature)

	sr.PeerHonestyHandler().ChangeScore(
		messageSender,
		spos.GetConsensusTopicID(sr.ShardCoordinator()),
		spos.LeaderPeerHonestyIncreaseFactor,
	)

	return sr.doEndRoundJobByParticipant(cnsDta)
}

func (sr *subroundEndRound) isBlockHeaderFinalInfoValid(cnsDta *consensus.Message) bool {
	if check.IfNil(sr.GetHeader()) {
		return false
	}

	header := sr.GetHeader().ShallowClone()

	// TODO[cleanup cns finality]: remove this
	if !sr.EnableEpochsHandler().IsFlagEnabledInEpoch(common.EquivalentMessagesFlag, header.GetEpoch()) {
		return sr.verifySignatures(header, cnsDta)
	}

	err := sr.HeaderSigVerifier().VerifySignatureForHash(header, cnsDta.BlockHeaderHash, cnsDta.PubKeysBitmap, cnsDta.AggregateSignature)
	if err != nil {
		log.Debug("isBlockHeaderFinalInfoValid.VerifySignatureForHash", "error", err.Error())
		return false
	}

	return true
}

func (sr *subroundEndRound) verifySignatures(header data.HeaderHandler, cnsDta *consensus.Message) bool {
	err := header.SetPubKeysBitmap(cnsDta.PubKeysBitmap)
	if err != nil {
		log.Debug("verifySignatures.SetPubKeysBitmap", "error", err.Error())
		return false
	}

	err = header.SetSignature(cnsDta.AggregateSignature)
	if err != nil {
		log.Debug("verifySignatures.SetSignature", "error", err.Error())
		return false
	}

	err = header.SetLeaderSignature(cnsDta.LeaderSignature)
	if err != nil {
		log.Debug("verifySignatures.SetLeaderSignature", "error", err.Error())
		return false
	}

	err = sr.HeaderSigVerifier().VerifyLeaderSignature(header)
	if err != nil {
		log.Debug("verifySignatures.VerifyLeaderSignature", "error", err.Error())
		return false
	}
	err = sr.HeaderSigVerifier().VerifySignature(header)
	if err != nil {
		log.Debug("verifySignatures.VerifySignature", "error", err.Error())
		return false
	}

	return true
}

// receivedInvalidSignersInfo method is called when a message with invalid signers has been received
func (sr *subroundEndRound) receivedInvalidSignersInfo(_ context.Context, cnsDta *consensus.Message) bool {
	messageSender := string(cnsDta.PubKey)

	if !sr.IsConsensusDataSet() {
		return false
	}
	if check.IfNil(sr.GetHeader()) {
		return false
	}

	// TODO[cleanup cns finality]: remove if statement
	isSenderAllowed := sr.IsNodeInConsensusGroup(messageSender)
	if !sr.EnableEpochsHandler().IsFlagEnabledInEpoch(common.EquivalentMessagesFlag, sr.GetHeader().GetEpoch()) {
		isSenderAllowed = sr.IsNodeLeaderInCurrentRound(messageSender)
	}
	if !isSenderAllowed { // is NOT this node leader in current round?
		sr.PeerHonestyHandler().ChangeScore(
			messageSender,
			spos.GetConsensusTopicID(sr.ShardCoordinator()),
			spos.LeaderPeerHonestyDecreaseFactor,
		)

		return false
	}

	// TODO[cleanup cns finality]: update this check
	isSelfSender := sr.IsNodeSelf(messageSender) || sr.IsKeyManagedBySelf([]byte(messageSender))
	if !sr.EnableEpochsHandler().IsFlagEnabledInEpoch(common.EquivalentMessagesFlag, sr.GetHeader().GetEpoch()) {
		isSelfSender = sr.IsSelfLeader()
	}
	if isSelfSender {
		return false
	}

	if !sr.IsConsensusDataEqual(cnsDta.BlockHeaderHash) {
		return false
	}

	if !sr.CanProcessReceivedMessage(cnsDta, sr.RoundHandler().Index(), sr.Current()) {
		return false
	}

	if len(cnsDta.InvalidSigners) == 0 {
		return false
	}

	err := sr.verifyInvalidSigners(cnsDta.InvalidSigners)
	if err != nil {
		log.Trace("receivedInvalidSignersInfo.verifyInvalidSigners", "error", err.Error())
		return false
	}

	log.Debug("step 3: invalid signers info has been evaluated")

	sr.PeerHonestyHandler().ChangeScore(
		messageSender,
		spos.GetConsensusTopicID(sr.ShardCoordinator()),
		spos.LeaderPeerHonestyIncreaseFactor,
	)

	return true
}

func (sr *subroundEndRound) verifyInvalidSigners(invalidSigners []byte) error {
	messages, err := sr.MessageSigningHandler().Deserialize(invalidSigners)
	if err != nil {
		return err
	}

	for _, msg := range messages {
		err = sr.verifyInvalidSigner(msg)
		if err != nil {
			return err
		}
	}

	return nil
}

func (sr *subroundEndRound) verifyInvalidSigner(msg p2p.MessageP2P) error {
	err := sr.MessageSigningHandler().Verify(msg)
	if err != nil {
		return err
	}

	cnsMsg := &consensus.Message{}
	err = sr.Marshalizer().Unmarshal(cnsMsg, msg.Data())
	if err != nil {
		return err
	}

	err = sr.SigningHandler().VerifySingleSignature(cnsMsg.PubKey, cnsMsg.BlockHeaderHash, cnsMsg.SignatureShare)
	if err != nil {
		log.Debug("verifyInvalidSigner: confirmed that node provided invalid signature",
			"pubKey", cnsMsg.PubKey,
			"blockHeaderHash", cnsMsg.BlockHeaderHash,
			"error", err.Error(),
		)
		sr.applyBlacklistOnNode(msg.Peer())
	}

	return nil
}

func (sr *subroundEndRound) applyBlacklistOnNode(peer core.PeerID) {
	sr.PeerBlacklistHandler().BlacklistPeer(peer, common.InvalidSigningBlacklistDuration)
}

func (sr *subroundEndRound) receivedHeader(headerHandler data.HeaderHandler) {
	isFlagEnabledForHeader := sr.EnableEpochsHandler().IsFlagEnabledInEpoch(common.EquivalentMessagesFlag, headerHandler.GetEpoch())
	// TODO[cleanup cns finality]: remove this method
	// if flag is enabled, no need to commit this header, as it will be committed once the proof is available
	if isFlagEnabledForHeader {
		return
	}

	isLeader := sr.IsSelfLeader()
	if sr.ConsensusGroup() == nil || isLeader {
		return
	}

	sr.mutProcessingEndRound.Lock()
	defer sr.mutProcessingEndRound.Unlock()

	sr.AddReceivedHeader(headerHandler)

	sr.doEndRoundJobByParticipant(nil)
}

// doEndRoundJob method does the job of the subround EndRound
func (sr *subroundEndRound) doEndRoundJob(_ context.Context) bool {
	if check.IfNil(sr.GetHeader()) {
		return false
	}

	// TODO[cleanup cns finality]: remove this code block
	isFlagEnabled := sr.EnableEpochsHandler().IsFlagEnabledInEpoch(common.EquivalentMessagesFlag, sr.GetHeader().GetEpoch())
	if !sr.IsSelfLeader() && !isFlagEnabled {
		if sr.IsSelfInConsensusGroup() {
			err := sr.prepareBroadcastBlockDataForValidator()
			if err != nil {
				log.Warn("validator in consensus group preparing for delayed broadcast",
					"error", err.Error())
			}
		}

		sr.mutProcessingEndRound.Lock()
		defer sr.mutProcessingEndRound.Unlock()

		return sr.doEndRoundJobByParticipant(nil)
	}

	if !sr.IsSelfInConsensusGroup() {
		sr.mutProcessingEndRound.Lock()
		defer sr.mutProcessingEndRound.Unlock()

		return sr.doEndRoundJobByParticipant(nil)
	}

	return sr.doEndRoundJobByLeader()
}

// TODO[cleanup cns finality]: rename this method, as this will be done by each participant
func (sr *subroundEndRound) doEndRoundJobByLeader() bool {
	sender, err := sr.getSender()
	if err != nil {
		return false
	}

	if !sr.waitForSignalSync() {
		return false
	}

	sr.mutProcessingEndRound.Lock()
	defer sr.mutProcessingEndRound.Unlock()

	if !sr.shouldSendFinalInfo() {
		return false
	}

	proof, ok := sr.sendFinalInfo(sender)
	if !ok {
		return false
	}

	// broadcast header
	// TODO[cleanup cns finality]: remove this, header already broadcast during subroundBlock
	if !sr.EnableEpochsHandler().IsFlagEnabledInEpoch(common.EquivalentMessagesFlag, sr.GetHeader().GetEpoch()) {
		err = sr.BroadcastMessenger().BroadcastHeader(sr.GetHeader(), sender)
		if err != nil {
			log.Warn("doEndRoundJobByLeader.BroadcastHeader", "error", err.Error())
		}
	}

	startTime := time.Now()
	err = sr.BlockProcessor().CommitBlock(sr.GetHeader(), sr.GetBody())
	elapsedTime := time.Since(startTime)
	if elapsedTime >= common.CommitMaxTime {
		log.Warn("doEndRoundJobByLeader.CommitBlock", "elapsed time", elapsedTime)
	} else {
		log.Debug("elapsed time to commit block",
			"time [s]", elapsedTime,
		)
	}
	if err != nil {
		log.Debug("doEndRoundJobByLeader.CommitBlock", "error", err)
		return false
	}

	if sr.EnableEpochsHandler().IsFlagEnabledInEpoch(common.EquivalentMessagesFlag, sr.GetHeader().GetEpoch()) {
		err = sr.EquivalentProofsPool().AddProof(proof)
		if err != nil {
			log.Debug("doEndRoundJobByLeader.AddProof", "error", err)
			return false
		}
	}

	sr.SetStatus(sr.Current(), spos.SsFinished)

	sr.worker.DisplayStatistics()

	log.Debug("step 3: Body and Header have been committed and header has been broadcast")

	err = sr.broadcastBlockDataLeader(sender)
	if err != nil {
		log.Debug("doEndRoundJobByLeader.broadcastBlockDataLeader", "error", err.Error())
	}

	msg := fmt.Sprintf("Added proposed block with nonce  %d  in blockchain", sr.GetHeader().GetNonce())
	log.Debug(display.Headline(msg, sr.SyncTimer().FormattedCurrentTime(), "+"))

	sr.updateMetricsForLeader()

	return true
}

func (sr *subroundEndRound) sendFinalInfo(sender []byte) (data.HeaderProofHandler, bool) {
	bitmap := sr.GenerateBitmap(bls.SrSignature)
	err := sr.checkSignaturesValidity(bitmap)
	if err != nil {
		log.Debug("sendFinalInfo.checkSignaturesValidity", "error", err.Error())
		return nil, false
	}

	// Aggregate signatures, handle invalid signers and send final info if needed
	bitmap, sig, err := sr.aggregateSigsAndHandleInvalidSigners(bitmap)
	if err != nil {
		log.Debug("sendFinalInfo.aggregateSigsAndHandleInvalidSigners", "error", err.Error())
		return nil, false
	}

	// TODO[cleanup cns finality]: remove this code block
	if !sr.EnableEpochsHandler().IsFlagEnabledInEpoch(common.EquivalentMessagesFlag, sr.GetHeader().GetEpoch()) {
		err = sr.GetHeader().SetPubKeysBitmap(bitmap)
		if err != nil {
			log.Debug("sendFinalInfo.SetPubKeysBitmap", "error", err.Error())
			return nil, false
		}

		err = sr.GetHeader().SetSignature(sig)
		if err != nil {
			log.Debug("sendFinalInfo.SetSignature", "error", err.Error())
			return nil, false
		}

		// Header is complete so the leader can sign it
		leaderSignature, err := sr.signBlockHeader(sender)
		if err != nil {
			log.Error(err.Error())
			return nil, false
		}

		err = sr.GetHeader().SetLeaderSignature(leaderSignature)
		if err != nil {
			log.Debug("sendFinalInfo.SetLeaderSignature", "error", err.Error())
			return nil, false
		}
	}

	ok := sr.ScheduledProcessor().IsProcessedOKWithTimeout()
	// placeholder for subroundEndRound.doEndRoundJobByLeader script
	if !ok {
		return nil, false
	}

	roundHandler := sr.RoundHandler()
	if roundHandler.RemainingTime(roundHandler.TimeStamp(), roundHandler.TimeDuration()) < 0 {
		log.Debug("sendFinalInfo: time is out -> cancel broadcasting final info and header",
			"round time stamp", roundHandler.TimeStamp(),
			"current time", time.Now())
		return nil, false
	}

	// broadcast header and final info section
	// TODO[cleanup cns finality]: remove leaderSigToBroadcast
	leaderSigToBroadcast := sr.GetHeader().GetLeaderSignature()
	if sr.EnableEpochsHandler().IsFlagEnabledInEpoch(common.EquivalentMessagesFlag, sr.GetHeader().GetEpoch()) {
		leaderSigToBroadcast = nil
	}

	if !sr.createAndBroadcastHeaderFinalInfoForKey(sig, bitmap, leaderSigToBroadcast, sender) {
		return nil, false
	}

	return &block.HeaderProof{
		PubKeysBitmap:       bitmap,
		AggregatedSignature: sig,
		HeaderHash:          sr.GetData(),
		HeaderEpoch:         sr.GetHeader().GetEpoch(),
		HeaderNonce:         sr.GetHeader().GetNonce(),
		HeaderShardId:       sr.GetHeader().GetShardID(),
	}, true
}

func (sr *subroundEndRound) shouldSendFinalInfo() bool {
	// TODO[cleanup cns finality]: remove this check
	if !sr.EnableEpochsHandler().IsFlagEnabledInEpoch(common.EquivalentMessagesFlag, sr.GetHeader().GetEpoch()) {
		return true
	}

	// TODO: check if this is the best approach. Perhaps we don't want to relay only on the first received message
	if sr.EquivalentProofsPool().HasProof(sr.ShardCoordinator().SelfId(), sr.GetData()) {
		log.Debug("shouldSendFinalInfo: equivalent message already processed")
		return false
	}

	return true
}

func (sr *subroundEndRound) aggregateSigsAndHandleInvalidSigners(bitmap []byte) ([]byte, []byte, error) {
	sig, err := sr.SigningHandler().AggregateSigs(bitmap, sr.GetHeader().GetEpoch())
	if err != nil {
		log.Debug("doEndRoundJobByLeader.AggregateSigs", "error", err.Error())

		return sr.handleInvalidSignersOnAggSigFail()
	}

	err = sr.SigningHandler().SetAggregatedSig(sig)
	if err != nil {
		log.Debug("doEndRoundJobByLeader.SetAggregatedSig", "error", err.Error())
		return nil, nil, err
	}

	err = sr.SigningHandler().Verify(sr.GetData(), bitmap, sr.GetHeader().GetEpoch())
	if err != nil {
		log.Debug("doEndRoundJobByLeader.Verify", "error", err.Error())

		return sr.handleInvalidSignersOnAggSigFail()
	}

	return bitmap, sig, nil
}

func (sr *subroundEndRound) checkGoRoutinesThrottler(ctx context.Context) error {
	for {
		if sr.signatureThrottler.CanProcess() {
			break
		}

		select {
		case <-time.After(time.Millisecond):
			continue
		case <-ctx.Done():
			return spos.ErrTimeIsOut
		}
	}
	return nil
}

// verifySignature implements parallel signature verification
func (sr *subroundEndRound) verifySignature(i int, pk string, sigShare []byte) error {
	err := sr.SigningHandler().VerifySignatureShare(uint16(i), sigShare, sr.GetData(), sr.GetHeader().GetEpoch())
	if err != nil {
		log.Trace("VerifySignatureShare returned an error: ", err)
		errSetJob := sr.SetJobDone(pk, bls.SrSignature, false)
		if errSetJob != nil {
			return errSetJob
		}

		decreaseFactor := -spos.ValidatorPeerHonestyIncreaseFactor + spos.ValidatorPeerHonestyDecreaseFactor

		sr.PeerHonestyHandler().ChangeScore(
			pk,
			spos.GetConsensusTopicID(sr.ShardCoordinator()),
			decreaseFactor,
		)
		return err
	}

	log.Trace("verifyNodesOnAggSigVerificationFail: verifying signature share", "public key", pk)

	return nil
}

func (sr *subroundEndRound) verifyNodesOnAggSigFail(ctx context.Context) ([]string, error) {
	wg := &sync.WaitGroup{}
	mutex := &sync.Mutex{}
	invalidPubKeys := make([]string, 0)
	pubKeys := sr.ConsensusGroup()

	if check.IfNil(sr.GetHeader()) {
		return nil, spos.ErrNilHeader
	}

	for i, pk := range pubKeys {
		isJobDone, err := sr.JobDone(pk, bls.SrSignature)
		if err != nil || !isJobDone {
			continue
		}

		sigShare, err := sr.SigningHandler().SignatureShare(uint16(i))
		if err != nil {
			return nil, err
		}

		err = sr.checkGoRoutinesThrottler(ctx)
		if err != nil {
			return nil, err
		}

		sr.signatureThrottler.StartProcessing()

		wg.Add(1)

		go func(i int, pk string, wg *sync.WaitGroup, sigShare []byte) {
			defer func() {
				sr.signatureThrottler.EndProcessing()
				wg.Done()
			}()
			errSigVerification := sr.verifySignature(i, pk, sigShare)
			if errSigVerification != nil {
				mutex.Lock()
				invalidPubKeys = append(invalidPubKeys, pk)
				mutex.Unlock()
			}
		}(i, pk, wg, sigShare)
	}
	wg.Wait()

	return invalidPubKeys, nil
}

func (sr *subroundEndRound) getFullMessagesForInvalidSigners(invalidPubKeys []string) ([]byte, error) {
	p2pMessages := make([]p2p.MessageP2P, 0)

	for _, pk := range invalidPubKeys {
		p2pMsg, ok := sr.GetMessageWithSignature(pk)
		if !ok {
			log.Trace("message not found in state for invalid signer", "pubkey", pk)
			continue
		}

		p2pMessages = append(p2pMessages, p2pMsg)
	}

	invalidSigners, err := sr.MessageSigningHandler().Serialize(p2pMessages)
	if err != nil {
		return nil, err
	}

	return invalidSigners, nil
}

func (sr *subroundEndRound) handleInvalidSignersOnAggSigFail() ([]byte, []byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), sr.RoundHandler().TimeDuration())
	invalidPubKeys, err := sr.verifyNodesOnAggSigFail(ctx)
	cancel()
	if err != nil {
		log.Debug("doEndRoundJobByLeader.verifyNodesOnAggSigFail", "error", err.Error())
		return nil, nil, err
	}

	invalidSigners, err := sr.getFullMessagesForInvalidSigners(invalidPubKeys)
	if err != nil {
		log.Debug("doEndRoundJobByLeader.getFullMessagesForInvalidSigners", "error", err.Error())
		return nil, nil, err
	}

	if len(invalidSigners) > 0 {
		sr.createAndBroadcastInvalidSigners(invalidSigners)
	}

	bitmap, sig, err := sr.computeAggSigOnValidNodes()
	if err != nil {
		log.Debug("doEndRoundJobByLeader.computeAggSigOnValidNodes", "error", err.Error())
		return nil, nil, err
	}

	return bitmap, sig, nil
}

func (sr *subroundEndRound) computeAggSigOnValidNodes() ([]byte, []byte, error) {
	threshold := sr.Threshold(bls.SrSignature)
	numValidSigShares := sr.ComputeSize(bls.SrSignature)

	if check.IfNil(sr.GetHeader()) {
		return nil, nil, spos.ErrNilHeader
	}

	if numValidSigShares < threshold {
		return nil, nil, fmt.Errorf("%w: number of valid sig shares lower than threshold, numSigShares: %d, threshold: %d",
			spos.ErrInvalidNumSigShares, numValidSigShares, threshold)
	}

	bitmap := sr.GenerateBitmap(bls.SrSignature)
	err := sr.checkSignaturesValidity(bitmap)
	if err != nil {
		return nil, nil, err
	}

	sig, err := sr.SigningHandler().AggregateSigs(bitmap, sr.GetHeader().GetEpoch())
	if err != nil {
		return nil, nil, err
	}

	err = sr.SigningHandler().SetAggregatedSig(sig)
	if err != nil {
		return nil, nil, err
	}

	return bitmap, sig, nil
}

func (sr *subroundEndRound) createAndBroadcastHeaderFinalInfoForKey(signature []byte, bitmap []byte, leaderSignature []byte, pubKey []byte) bool {
	index, err := sr.ConsensusGroupIndex(string(pubKey))
	if err != nil {
		log.Debug("createAndBroadcastHeaderFinalInfoForKey.ConsensusGroupIndex", "error", err.Error())
		return false
	}

	headerProof := &block.HeaderProof{
		AggregatedSignature: signature,
		PubKeysBitmap:       bitmap,
		HeaderHash:          sr.GetData(),
		HeaderEpoch:         sr.GetHeader().GetEpoch(),
	}

	sr.BroadcastMessenger().PrepareBroadcastEquivalentProof(headerProof, index, pubKey)
	log.Debug("step 3: block header proof has been sent to delayed broadcaster",
		"PubKeysBitmap", bitmap,
		"AggregateSignature", signature,
		"LeaderSignature", leaderSignature,
		"Index", index)

	return true
}

func (sr *subroundEndRound) createAndBroadcastInvalidSigners(invalidSigners []byte) {
	// TODO[cleanup cns finality]: remove the leader check
	isEquivalentMessagesFlagEnabled := sr.EnableEpochsHandler().IsFlagEnabledInEpoch(common.EquivalentMessagesFlag, sr.GetHeader().GetEpoch())
	if !sr.IsSelfLeader() && !isEquivalentMessagesFlagEnabled {
		return
	}

	sender, err := sr.getSender()
	if err != nil {
		log.Debug("createAndBroadcastInvalidSigners.getSender", "error", err)
		return
	}

	cnsMsg := consensus.NewConsensusMessage(
		sr.GetData(),
		nil,
		nil,
		nil,
		sender,
		nil,
		int(bls.MtInvalidSigners),
		sr.RoundHandler().Index(),
		sr.ChainID(),
		nil,
		nil,
		nil,
		sr.GetAssociatedPid(sender),
		invalidSigners,
	)

	// TODO[Sorin next PR]: decide if we send this with the delayed broadcast
	err = sr.BroadcastMessenger().BroadcastConsensusMessage(cnsMsg)
	if err != nil {
		log.Debug("doEndRoundJob.BroadcastConsensusMessage", "error", err.Error())
		return
	}

	log.Debug("step 3: invalid signers info has been sent")
}

func (sr *subroundEndRound) doEndRoundJobByParticipant(cnsDta *consensus.Message) bool {
	if sr.GetRoundCanceled() {
		return false
	}
	if !sr.IsConsensusDataSet() {
		return false
	}
	if !sr.IsSubroundFinished(sr.Previous()) {
		return false
	}
	if sr.IsSubroundFinished(sr.Current()) {
		return false
	}

	haveHeader, header := sr.haveConsensusHeaderWithFullInfo(cnsDta)
	if !haveHeader {
		return false
	}

	defer func() {
		sr.SetProcessingBlock(false)
	}()

	sr.SetProcessingBlock(true)

	shouldNotCommitBlock := sr.GetExtendedCalled() || int64(header.GetRound()) < sr.RoundHandler().Index()
	if shouldNotCommitBlock {
		log.Debug("canceled round, extended has been called or round index has been changed",
			"round", sr.RoundHandler().Index(),
			"subround", sr.Name(),
			"header round", header.GetRound(),
			"extended called", sr.GetExtendedCalled(),
		)
		return false
	}

	if sr.isOutOfTime() {
		return false
	}

	ok := sr.ScheduledProcessor().IsProcessedOKWithTimeout()
	if !ok {
		return false
	}

	startTime := time.Now()
	err := sr.BlockProcessor().CommitBlock(header, sr.GetBody())
	elapsedTime := time.Since(startTime)
	if elapsedTime >= common.CommitMaxTime {
		log.Warn("doEndRoundJobByParticipant.CommitBlock", "elapsed time", elapsedTime)
	} else {
		log.Debug("elapsed time to commit block",
			"time [s]", elapsedTime,
		)
	}
	if err != nil {
		log.Debug("doEndRoundJobByParticipant.CommitBlock", "error", err.Error())
		return false
	}

	isSelfInConsensus := sr.IsSelfInConsensusGroup()
	isEquivalentMessagesFlagEnabled := sr.EnableEpochsHandler().IsFlagEnabledInEpoch(common.EquivalentMessagesFlag, header.GetEpoch())
	if isSelfInConsensus && cnsDta != nil && isEquivalentMessagesFlagEnabled {
		proof := &block.HeaderProof{
			PubKeysBitmap:       cnsDta.PubKeysBitmap,
			AggregatedSignature: cnsDta.AggregateSignature,
			HeaderHash:          cnsDta.BlockHeaderHash,
			HeaderEpoch:         header.GetEpoch(),
			HeaderNonce:         header.GetNonce(),
			HeaderShardId:       header.GetShardID(),
		}
		err = sr.EquivalentProofsPool().AddProof(proof)
		if err != nil {
			log.Debug("doEndRoundJobByParticipant.AddProof", "error", err)
			return false
		}
	}

	sr.SetStatus(sr.Current(), spos.SsFinished)

	// TODO[cleanup cns finality]: remove this
	if isSelfInConsensus && !isEquivalentMessagesFlagEnabled {
		err = sr.setHeaderForValidator(header)
		if err != nil {
			log.Warn("doEndRoundJobByParticipant", "error", err.Error())
		}
	}

	sr.worker.DisplayStatistics()

	log.Debug("step 3: Body and Header have been committed")

	headerTypeMsg := "received"
	if cnsDta != nil {
		headerTypeMsg = "assembled"
	}

	msg := fmt.Sprintf("Added %s block with nonce  %d  in blockchain", headerTypeMsg, header.GetNonce())
	log.Debug(display.Headline(msg, sr.SyncTimer().FormattedCurrentTime(), "-"))
	return true
}

func (sr *subroundEndRound) haveConsensusHeaderWithFullInfo(cnsDta *consensus.Message) (bool, data.HeaderHandler) {
	if cnsDta == nil {
		return sr.isConsensusHeaderReceived()
	}

	if check.IfNil(sr.GetHeader()) {
		return false, nil
	}

	header := sr.GetHeader().ShallowClone()
	// TODO[cleanup cns finality]: remove this
	if !sr.EnableEpochsHandler().IsFlagEnabledInEpoch(common.EquivalentMessagesFlag, header.GetEpoch()) {
		err := header.SetPubKeysBitmap(cnsDta.PubKeysBitmap)
		if err != nil {
			return false, nil
		}

		err = header.SetSignature(cnsDta.AggregateSignature)
		if err != nil {
			return false, nil
		}

		err = header.SetLeaderSignature(cnsDta.LeaderSignature)
		if err != nil {
			return false, nil
		}

		return true, header
	}

	return true, header
}

func (sr *subroundEndRound) isConsensusHeaderReceived() (bool, data.HeaderHandler) {
	if check.IfNil(sr.GetHeader()) {
		return false, nil
	}

	consensusHeaderHash, err := core.CalculateHash(sr.Marshalizer(), sr.Hasher(), sr.GetHeader())
	if err != nil {
		log.Debug("isConsensusHeaderReceived: calculate consensus header hash", "error", err.Error())
		return false, nil
	}

	receivedHeaders := sr.GetReceivedHeaders()

	var receivedHeaderHash []byte
	for index := range receivedHeaders {
		// TODO[cleanup cns finality]: remove this
		receivedHeader := receivedHeaders[index].ShallowClone()
		if !sr.EnableEpochsHandler().IsFlagEnabledInEpoch(common.EquivalentMessagesFlag, receivedHeader.GetEpoch()) {
			err = receivedHeader.SetLeaderSignature(nil)
			if err != nil {
				log.Debug("isConsensusHeaderReceived - SetLeaderSignature", "error", err.Error())
				return false, nil
			}

			err = receivedHeader.SetPubKeysBitmap(nil)
			if err != nil {
				log.Debug("isConsensusHeaderReceived - SetPubKeysBitmap", "error", err.Error())
				return false, nil
			}

			err = receivedHeader.SetSignature(nil)
			if err != nil {
				log.Debug("isConsensusHeaderReceived - SetSignature", "error", err.Error())
				return false, nil
			}
		}

		receivedHeaderHash, err = core.CalculateHash(sr.Marshalizer(), sr.Hasher(), receivedHeader)
		if err != nil {
			log.Debug("isConsensusHeaderReceived: calculate received header hash", "error", err.Error())
			return false, nil
		}

		if bytes.Equal(receivedHeaderHash, consensusHeaderHash) {
			return true, receivedHeaders[index]
		}
	}

	return false, nil
}

func (sr *subroundEndRound) signBlockHeader(leader []byte) ([]byte, error) {
	headerClone := sr.GetHeader().ShallowClone()
	err := headerClone.SetLeaderSignature(nil)
	if err != nil {
		return nil, err
	}

	marshalizedHdr, err := sr.Marshalizer().Marshal(headerClone)
	if err != nil {
		return nil, err
	}

	return sr.SigningHandler().CreateSignatureForPublicKey(marshalizedHdr, leader)
}

func (sr *subroundEndRound) updateMetricsForLeader() {
	// TODO: decide if we keep these metrics the same way
	sr.appStatusHandler.Increment(common.MetricCountAcceptedBlocks)
	sr.appStatusHandler.SetStringValue(common.MetricConsensusRoundState,
		fmt.Sprintf("valid block produced in %f sec", time.Since(sr.RoundHandler().TimeStamp()).Seconds()))
}

func (sr *subroundEndRound) broadcastBlockDataLeader(sender []byte) error {
	// TODO[cleanup cns finality]: remove this method, block data was already broadcast during subroundBlock
	if sr.EnableEpochsHandler().IsFlagEnabledInEpoch(common.EquivalentMessagesFlag, sr.GetHeader().GetEpoch()) {
		return nil
	}

	miniBlocks, transactions, err := sr.BlockProcessor().MarshalizedDataToBroadcast(sr.GetHeader(), sr.GetBody())
	if err != nil {
		return err
	}

	return sr.BroadcastMessenger().BroadcastBlockDataLeader(sr.GetHeader(), miniBlocks, transactions, sender)
}

func (sr *subroundEndRound) setHeaderForValidator(header data.HeaderHandler) error {
	idx, pk, miniBlocks, transactions, err := sr.getIndexPkAndDataToBroadcast()
	if err != nil {
		return err
	}

	go sr.BroadcastMessenger().PrepareBroadcastHeaderValidator(header, miniBlocks, transactions, idx, pk)

	return nil
}

func (sr *subroundEndRound) prepareBroadcastBlockDataForValidator() error {
	idx, pk, miniBlocks, transactions, err := sr.getIndexPkAndDataToBroadcast()
	if err != nil {
		return err
	}

	go sr.BroadcastMessenger().PrepareBroadcastBlockDataValidator(sr.GetHeader(), miniBlocks, transactions, idx, pk)

	return nil
}

// doEndRoundConsensusCheck method checks if the consensus is achieved
func (sr *subroundEndRound) doEndRoundConsensusCheck() bool {
	if sr.GetRoundCanceled() {
		return false
	}

	return sr.IsSubroundFinished(sr.Current())
}

func (sr *subroundEndRound) checkSignaturesValidity(bitmap []byte) error {
	if !sr.hasProposerSignature(bitmap) {
		return spos.ErrMissingProposerSignature
	}

	consensusGroup := sr.ConsensusGroup()
	signers := headerCheck.ComputeSignersPublicKeys(consensusGroup, bitmap)
	for _, pubKey := range signers {
		isSigJobDone, err := sr.JobDone(pubKey, bls.SrSignature)
		if err != nil {
			return err
		}

		if !isSigJobDone {
			return spos.ErrNilSignature
		}
	}

	return nil
}

func (sr *subroundEndRound) hasProposerSignature(bitmap []byte) bool {
	// TODO[cleanup cns finality]: remove this check
	if !sr.EnableEpochsHandler().IsFlagEnabledInEpoch(common.EquivalentMessagesFlag, sr.GetHeader().GetEpoch()) {
		return true
	}

	return bitmap[0]&1 > 0
}

func (sr *subroundEndRound) isOutOfTime() bool {
	startTime := sr.GetRoundTimeStamp()
	maxTime := sr.RoundHandler().TimeDuration() * time.Duration(sr.processingThresholdPercentage) / 100
	if sr.RoundHandler().RemainingTime(startTime, maxTime) < 0 {
		log.Debug("canceled round, time is out",
			"round", sr.SyncTimer().FormattedCurrentTime(), sr.RoundHandler().Index(),
			"subround", sr.Name())

		sr.SetRoundCanceled(true)
		return true
	}

	return false
}

func (sr *subroundEndRound) getIndexPkAndDataToBroadcast() (int, []byte, map[uint32][]byte, map[string][][]byte, error) {
	minIdx := sr.getMinConsensusGroupIndexOfManagedKeys()

	idx, err := sr.SelfConsensusGroupIndex()
	if err == nil {
		if idx < minIdx {
			minIdx = idx
		}
	}

	if minIdx == sr.ConsensusGroupSize() {
		return -1, nil, nil, nil, err
	}

	miniBlocks, transactions, err := sr.BlockProcessor().MarshalizedDataToBroadcast(sr.GetHeader(), sr.GetBody())
	if err != nil {
		return -1, nil, nil, nil, err
	}

	consensusGroup := sr.ConsensusGroup()
	pk := []byte(consensusGroup[minIdx])

	return minIdx, pk, miniBlocks, transactions, nil
}

func (sr *subroundEndRound) getMinConsensusGroupIndexOfManagedKeys() int {
	minIdx := sr.ConsensusGroupSize()

	for idx, validator := range sr.ConsensusGroup() {
		if !sr.IsKeyManagedBySelf([]byte(validator)) {
			continue
		}

		if idx < minIdx {
			minIdx = idx
		}
	}

	return minIdx
}

func (sr *subroundEndRound) getSender() ([]byte, error) {
	// TODO[cleanup cns finality]: remove this code block
	if !sr.EnableEpochsHandler().IsFlagEnabledInEpoch(common.EquivalentMessagesFlag, sr.GetHeader().GetEpoch()) {
		leader, errGetLeader := sr.GetLeader()
		if errGetLeader != nil {
			log.Debug("GetLeader", "error", errGetLeader)
			return nil, errGetLeader
		}

		return []byte(leader), nil
	}

	for _, pk := range sr.ConsensusGroup() {
		pkBytes := []byte(pk)
		if !sr.IsKeyManagedBySelf(pkBytes) {
			continue
		}

		return pkBytes, nil
	}

	return []byte(sr.SelfPubKey()), nil
}

func (sr *subroundEndRound) waitForSignalSync() bool {
	// TODO[cleanup cns finality]: remove this
	if !sr.EnableEpochsHandler().IsFlagEnabledInEpoch(common.EquivalentMessagesFlag, sr.GetHeader().GetEpoch()) {
		return true
	}

	if sr.IsSubroundFinished(sr.Current()) {
		return true
	}

	if sr.checkReceivedSignatures() {
		return true
	}

	go sr.waitAllSignatures()

	timerBetweenStatusChecks := time.NewTimer(timeBetweenSignaturesChecks)

	remainingSRTime := sr.remainingTime()
	timeout := time.NewTimer(remainingSRTime)
	for {
		select {
		case <-timerBetweenStatusChecks.C:
			if sr.IsSubroundFinished(sr.Current()) {
				log.Trace("subround already finished", "subround", sr.Name())
				return false
			}

			if sr.checkReceivedSignatures() {
				return true
			}
			timerBetweenStatusChecks.Reset(timeBetweenSignaturesChecks)
		case <-timeout.C:
			log.Debug("timeout while waiting for signatures or final info", "subround", sr.Name())
			return false
		}
	}
}

func (sr *subroundEndRound) waitAllSignatures() {
	remainingTime := sr.remainingTime()
	time.Sleep(remainingTime)

	if sr.IsSubroundFinished(sr.Current()) {
		return
	}

	sr.SetWaitingAllSignaturesTimeOut(true)

	select {
	case sr.ConsensusChannel() <- true:
	default:
	}
}

func (sr *subroundEndRound) remainingTime() time.Duration {
	startTime := sr.RoundHandler().TimeStamp()
	maxTime := time.Duration(float64(sr.StartTime()) + float64(sr.EndTime()-sr.StartTime())*waitingAllSigsMaxTimeThreshold)
	remainingTime := sr.RoundHandler().RemainingTime(startTime, maxTime)

	return remainingTime
}

// receivedSignature method is called when a signature is received through the signature channel.
// If the signature is valid, then the jobDone map corresponding to the node which sent it,
// is set on true for the subround Signature
func (sr *subroundEndRound) receivedSignature(_ context.Context, cnsDta *consensus.Message) bool {
	// TODO[cleanup cns finality]: remove this check
	if !sr.EnableEpochsHandler().IsFlagEnabledInEpoch(common.EquivalentMessagesFlag, sr.GetHeader().GetEpoch()) {
		return true
	}

	node := string(cnsDta.PubKey)
	pkForLogs := core.GetTrimmedPk(hex.EncodeToString(cnsDta.PubKey))

	if !sr.IsConsensusDataSet() {
		return false
	}

	if !sr.IsNodeInConsensusGroup(node) {
		sr.PeerHonestyHandler().ChangeScore(
			node,
			spos.GetConsensusTopicID(sr.ShardCoordinator()),
			spos.ValidatorPeerHonestyDecreaseFactor,
		)

		return false
	}

	if !sr.IsConsensusDataEqual(cnsDta.BlockHeaderHash) {
		return false
	}

	if !sr.CanProcessReceivedMessage(cnsDta, sr.RoundHandler().Index(), sr.Current()) {
		return false
	}

	index, err := sr.ConsensusGroupIndex(node)
	if err != nil {
		log.Debug("receivedSignature.ConsensusGroupIndex",
			"node", pkForLogs,
			"error", err.Error())
		return false
	}

	err = sr.SigningHandler().StoreSignatureShare(uint16(index), cnsDta.SignatureShare)
	if err != nil {
		log.Debug("receivedSignature.StoreSignatureShare",
			"node", pkForLogs,
			"index", index,
			"error", err.Error())
		return false
	}

	err = sr.SetJobDone(node, bls.SrSignature, true)
	if err != nil {
		log.Debug("receivedSignature.SetJobDone",
			"node", pkForLogs,
			"subround", sr.Name(),
			"error", err.Error())
		return false
	}

	sr.PeerHonestyHandler().ChangeScore(
		node,
		spos.GetConsensusTopicID(sr.ShardCoordinator()),
		spos.ValidatorPeerHonestyIncreaseFactor,
	)

	return true
}

func (sr *subroundEndRound) checkReceivedSignatures() bool {
	threshold := sr.Threshold(bls.SrSignature)
	if sr.FallbackHeaderValidator().ShouldApplyFallbackValidation(sr.GetHeader()) {
		threshold = sr.FallbackThreshold(bls.SrSignature)
		log.Warn("subroundEndRound.checkReceivedSignatures: fallback validation has been applied",
			"minimum number of signatures required", threshold,
			"actual number of signatures received", sr.getNumOfSignaturesCollected(),
		)
	}

	areSignaturesCollected, numSigs := sr.areSignaturesCollected(threshold)
	areAllSignaturesCollected := numSigs == sr.ConsensusGroupSize()

	isSignatureCollectionDone := areAllSignaturesCollected || (areSignaturesCollected && sr.GetWaitingAllSignaturesTimeOut())

	isSelfJobDone := sr.IsSelfJobDone(bls.SrSignature)

	shouldStopWaitingSignatures := isSelfJobDone && isSignatureCollectionDone
	if shouldStopWaitingSignatures {
		log.Debug("step 2: signatures collection done",
			"subround", sr.Name(),
			"signatures received", numSigs,
			"total signatures", len(sr.ConsensusGroup()))

		return true
	}

	return false
}

func (sr *subroundEndRound) getNumOfSignaturesCollected() int {
	n := 0

	for i := 0; i < len(sr.ConsensusGroup()); i++ {
		node := sr.ConsensusGroup()[i]

		isSignJobDone, err := sr.JobDone(node, bls.SrSignature)
		if err != nil {
			log.Debug("getNumOfSignaturesCollected.JobDone",
				"node", node,
				"subround", sr.Name(),
				"error", err.Error())
			continue
		}

		if isSignJobDone {
			n++
		}
	}

	return n
}

// areSignaturesCollected method checks if the signatures received from the nodes, belonging to the current
// jobDone group, are more than the necessary given threshold
func (sr *subroundEndRound) areSignaturesCollected(threshold int) (bool, int) {
	n := sr.getNumOfSignaturesCollected()
	return n >= threshold, n
}

// IsInterfaceNil returns true if there is no value under the interface
func (sr *subroundEndRound) IsInterfaceNil() bool {
	return sr == nil
}