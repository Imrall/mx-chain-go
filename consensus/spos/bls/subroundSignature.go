package bls

import (
	"context"
	"encoding/hex"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/multiversx/mx-chain-core-go/core"
	atomicCore "github.com/multiversx/mx-chain-core-go/core/atomic"
	"github.com/multiversx/mx-chain-core-go/core/check"

	"github.com/multiversx/mx-chain-go/common"
	"github.com/multiversx/mx-chain-go/consensus"
	"github.com/multiversx/mx-chain-go/consensus/spos"
)

const timeSpentBetweenChecks = time.Millisecond

type subroundSignature struct {
	*spos.Subround
	appStatusHandler     core.AppStatusHandler
	sentSignatureTracker spos.SentSignaturesTracker
	signatureThrottler   core.Throttler
}

// NewSubroundSignature creates a subroundSignature object
func NewSubroundSignature(
	baseSubround *spos.Subround,
	appStatusHandler core.AppStatusHandler,
	sentSignatureTracker spos.SentSignaturesTracker,
	worker spos.WorkerHandler,
	signatureThrottler core.Throttler,
) (*subroundSignature, error) {
	err := checkNewSubroundSignatureParams(
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

	srSignature := subroundSignature{
		Subround:             baseSubround,
		appStatusHandler:     appStatusHandler,
		sentSignatureTracker: sentSignatureTracker,
		signatureThrottler:   signatureThrottler,
	}
	srSignature.Job = srSignature.doSignatureJob
	srSignature.Check = srSignature.doSignatureConsensusCheck
	srSignature.Extend = worker.Extend

	return &srSignature, nil
}

func checkNewSubroundSignatureParams(
	baseSubround *spos.Subround,
) error {
	if baseSubround == nil {
		return spos.ErrNilSubround
	}
	if baseSubround.ConsensusState == nil {
		return spos.ErrNilConsensusState
	}

	err := spos.ValidateConsensusCore(baseSubround.ConsensusCoreHandler)

	return err
}

// doSignatureJob method does the job of the subround Signature
func (sr *subroundSignature) doSignatureJob(ctx context.Context) bool {
	if !sr.CanDoSubroundJob(sr.Current()) {
		return false
	}
	if check.IfNil(sr.Header) {
		log.Error("doSignatureJob", "error", spos.ErrNilHeader)
		return false
	}

	isSelfSingleKeyLeader := sr.IsSelfLeaderInCurrentRound() && sr.ShouldConsiderSelfKeyInConsensus()
	isSelfSingleKeyInConsensusGroup := sr.IsNodeInConsensusGroup(sr.SelfPubKey()) && sr.ShouldConsiderSelfKeyInConsensus()
	isFlagActive := sr.EnableEpochsHandler().IsFlagEnabledInEpoch(common.EquivalentMessagesFlag, sr.Header.GetEpoch())
	// if single key leader, the signature has been sent on subroundBlock, thus the current round can be marked as finished
	if isSelfSingleKeyLeader && isFlagActive {
		leader, err := sr.GetLeader()
		if err != nil {
			return false
		}
		err = sr.SetJobDone(leader, sr.Current(), true)
		if err != nil {
			return false
		}

		sr.SetStatus(sr.Current(), spos.SsFinished)

		sr.appStatusHandler.SetStringValue(common.MetricConsensusRoundState, "signed")

		log.Debug("step 2: subround has been finished for leader",
			"subround", sr.Name())

		return true
	}

	if isSelfSingleKeyLeader || isSelfSingleKeyInConsensusGroup {
		if !sr.doSignatureJobForSingleKey(isSelfSingleKeyLeader, isFlagActive) {
			return false
		}
	}

	if !sr.doSignatureJobForManagedKeys(ctx) {
		return false
	}

	if isFlagActive {
		sr.SetStatus(sr.Current(), spos.SsFinished)

		log.Debug("step 2: subround has been finished",
			"subround", sr.Name())
	}

	return true
}

func (sr *subroundSignature) createAndSendSignatureMessage(signatureShare []byte, pkBytes []byte) bool {
	cnsMsg := consensus.NewConsensusMessage(
		sr.GetData(),
		signatureShare,
		nil,
		nil,
		pkBytes,
		nil,
		int(MtSignature),
		sr.RoundHandler().Index(),
		sr.ChainID(),
		nil,
		nil,
		nil,
		sr.GetAssociatedPid(pkBytes),
		nil,
	)

	err := sr.BroadcastMessenger().BroadcastConsensusMessage(cnsMsg)
	if err != nil {
		log.Debug("createAndSendSignatureMessage.BroadcastConsensusMessage",
			"error", err.Error(), "pk", pkBytes)
		return false
	}

	log.Debug("step 2: signature has been sent", "pk", pkBytes)

	return true
}

func (sr *subroundSignature) completeSignatureSubRound(pk string, shouldWaitForAllSigsAsync bool) bool {
	err := sr.SetJobDone(pk, sr.Current(), true)
	if err != nil {
		log.Debug("doSignatureJob.SetSelfJobDone",
			"subround", sr.Name(),
			"error", err.Error(),
			"pk", []byte(pk),
		)
		return false
	}

	// TODO[cleanup cns finality]: do not wait for signatures anymore, this will be done on subroundEndRound
	if shouldWaitForAllSigsAsync {
		go sr.waitAllSignatures()
	}

	return true
}

// receivedSignature method is called when a signature is received through the signature channel.
// If the signature is valid, then the jobDone map corresponding to the node which sent it,
// is set on true for the subround Signature
func (sr *subroundSignature) receivedSignature(_ context.Context, cnsDta *consensus.Message) bool {
	// TODO[cleanup cns finality]: remove this method, received signatures will be handled on subroundEndRound
	if sr.EnableEpochsHandler().IsFlagEnabledInEpoch(common.EquivalentMessagesFlag, sr.Header.GetEpoch()) {
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

	if !sr.IsSelfLeaderInCurrentRound() && !sr.IsMultiKeyLeaderInCurrentRound() {
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

	err = sr.SetJobDone(node, sr.Current(), true)
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

// doSignatureConsensusCheck method checks if the consensus in the subround Signature is achieved
func (sr *subroundSignature) doSignatureConsensusCheck() bool {
	if sr.RoundCanceled {
		return false
	}

	if sr.IsSubroundFinished(sr.Current()) {
		return true
	}

	if check.IfNil(sr.Header) {
		return false
	}

	isSelfInConsensusGroup := sr.IsNodeInConsensusGroup(sr.SelfPubKey()) || sr.IsMultiKeyInConsensusGroup()
	if !isSelfInConsensusGroup {
		log.Debug("step 2: subround has been finished",
			"subround", sr.Name())
		sr.SetStatus(sr.Current(), spos.SsFinished)

		return true
	}

	// TODO[cleanup cns finality]: simply return false and remove the rest of the method. This will be handled by subroundEndRound
	if sr.EnableEpochsHandler().IsFlagEnabledInEpoch(common.EquivalentMessagesFlag, sr.Header.GetEpoch()) {
		return false
	}

	isSelfLeader := sr.IsSelfLeaderInCurrentRound() || sr.IsMultiKeyLeaderInCurrentRound()

	threshold := sr.Threshold(sr.Current())
	if sr.FallbackHeaderValidator().ShouldApplyFallbackValidation(sr.Header) {
		threshold = sr.FallbackThreshold(sr.Current())
		log.Warn("subroundSignature.doSignatureConsensusCheck: fallback validation has been applied",
			"minimum number of signatures required", threshold,
			"actual number of signatures received", sr.getNumOfSignaturesCollected(),
		)
	}

	areSignaturesCollected, numSigs := sr.areSignaturesCollected(threshold)
	areAllSignaturesCollected := numSigs == sr.ConsensusGroupSize()

	isSignatureCollectionDone := areAllSignaturesCollected || (areSignaturesCollected && sr.WaitingAllSignaturesTimeOut)
	isJobDoneByLeader := isSelfLeader && isSignatureCollectionDone

	selfJobDone := true
	if sr.IsNodeInConsensusGroup(sr.SelfPubKey()) {
		selfJobDone = sr.IsSelfJobDone(sr.Current())
	}
	multiKeyJobDone := true
	if sr.IsMultiKeyInConsensusGroup() {
		multiKeyJobDone = sr.IsMultiKeyJobDone(sr.Current())
	}
	isJobDoneByConsensusNode := !isSelfLeader && isSelfInConsensusGroup && selfJobDone && multiKeyJobDone

	isSubroundFinished := isJobDoneByConsensusNode || isJobDoneByLeader

	if isSubroundFinished {
		if isSelfLeader {
			log.Debug("step 2: signatures",
				"received", numSigs,
				"total", len(sr.ConsensusGroup()))
		}

		log.Debug("step 2: subround has been finished",
			"subround", sr.Name())
		sr.SetStatus(sr.Current(), spos.SsFinished)

		sr.appStatusHandler.SetStringValue(common.MetricConsensusRoundState, "signed")

		return true
	}

	return false
}

// TODO[cleanup cns finality]: remove this, already moved on subroundEndRound
// areSignaturesCollected method checks if the signatures received from the nodes, belonging to the current
// jobDone group, are more than the necessary given threshold
func (sr *subroundSignature) areSignaturesCollected(threshold int) (bool, int) {
	n := sr.getNumOfSignaturesCollected()
	return n >= threshold, n
}

// TODO[cleanup cns finality]: remove this, already moved on subroundEndRound
func (sr *subroundSignature) getNumOfSignaturesCollected() int {
	n := 0

	for i := 0; i < len(sr.ConsensusGroup()); i++ {
		node := sr.ConsensusGroup()[i]

		isSignJobDone, err := sr.JobDone(node, sr.Current())
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

// TODO[cleanup cns finality]: remove this, already moved on subroundEndRound
func (sr *subroundSignature) waitAllSignatures() {
	remainingTime := sr.remainingTime()
	time.Sleep(remainingTime)

	if sr.IsSubroundFinished(sr.Current()) {
		return
	}

	sr.WaitingAllSignaturesTimeOut = true

	select {
	case sr.ConsensusChannel() <- true:
	default:
	}
}

// TODO[cleanup cns finality]: remove this, already moved on subroundEndRound
func (sr *subroundSignature) remainingTime() time.Duration {
	startTime := sr.RoundHandler().TimeStamp()
	maxTime := time.Duration(float64(sr.StartTime()) + float64(sr.EndTime()-sr.StartTime())*waitingAllSigsMaxTimeThreshold)
	remainigTime := sr.RoundHandler().RemainingTime(startTime, maxTime)

	return remainigTime
}

func (sr *subroundSignature) doSignatureJobForManagedKeys(ctx context.Context) bool {

	numMultiKeysSignaturesSent := int32(0)
	sentSigForAllKeys := atomicCore.Flag{}
	sentSigForAllKeys.SetValue(true)

	wg := sync.WaitGroup{}

	for idx, pk := range sr.ConsensusGroup() {
		pkBytes := []byte(pk)
		if !sr.IsKeyManagedByCurrentNode(pkBytes) {
			continue
		}

		if sr.IsJobDone(pk, sr.Current()) {
			continue
		}

		err := sr.checkGoRoutinesThrottler(ctx)
		if err != nil {
			return false
		}
		sr.signatureThrottler.StartProcessing()
		wg.Add(1)

		go func(idx int, pk string) {
			defer sr.signatureThrottler.EndProcessing()

			signatureSent := sr.sendSignatureForManagedKey(idx, pk)
			if signatureSent {
				atomic.AddInt32(&numMultiKeysSignaturesSent, 1)
			} else {
				sentSigForAllKeys.SetValue(false)
			}
			wg.Done()
		}(idx, pk)
	}

	wg.Wait()

	if numMultiKeysSignaturesSent > 0 {
		log.Debug("step 2: multi keys signatures have been sent", "num", numMultiKeysSignaturesSent)
	}

	return sentSigForAllKeys.IsSet()
}

func (sr *subroundSignature) sendSignatureForManagedKey(idx int, pk string) bool {
	isCurrentNodeMultiKeyLeader := sr.IsMultiKeyLeaderInCurrentRound()
	isFlagActive := sr.EnableEpochsHandler().IsFlagEnabledInEpoch(common.EquivalentMessagesFlag, sr.Header.GetEpoch())

	pkBytes := []byte(pk)

	signatureShare, err := sr.SigningHandler().CreateSignatureShareForPublicKey(
		sr.GetData(),
		uint16(idx),
		sr.Header.GetEpoch(),
		pkBytes,
	)
	if err != nil {
		log.Debug("doSignatureJobForManagedKeys.CreateSignatureShareForPublicKey", "error", err.Error())
		return false
	}

	isCurrentManagedKeyLeader := idx == spos.IndexOfLeaderInConsensusGroup
	// TODO[cleanup cns finality]: update the check
	// with the equivalent messages feature on, signatures from all managed keys must be broadcast, as the aggregation is done by any participant
	shouldBroadcastSignatureShare := (!isCurrentNodeMultiKeyLeader && !isFlagActive) ||
		(!isCurrentManagedKeyLeader && isFlagActive)
	if shouldBroadcastSignatureShare {
		ok := sr.createAndSendSignatureMessage(signatureShare, pkBytes)

		if !ok {
			return false
		}

	}
	// with the equivalent messages feature on, the leader signature is sent on subroundBlock, thus we should update its status here as well
	sr.sentSignatureTracker.SignatureSent(pkBytes)

	shouldWaitForAllSigsAsync := isCurrentManagedKeyLeader && !isFlagActive

	return sr.completeSignatureSubRound(pk, shouldWaitForAllSigsAsync)
}

func (sr *subroundSignature) checkGoRoutinesThrottler(ctx context.Context) error {
	for {
		if sr.signatureThrottler.CanProcess() {
			break
		}
		select {
		case <-time.After(timeSpentBetweenChecks):
			continue
		case <-ctx.Done():
			return fmt.Errorf("%w while checking the throttler", spos.ErrTimeIsOut)
		}
	}
	return nil
}

func (sr *subroundSignature) doSignatureJobForSingleKey(isSelfLeader bool, isFlagActive bool) bool {
	selfIndex, err := sr.SelfConsensusGroupIndex()
	if err != nil {
		log.Debug("doSignatureJob.SelfConsensusGroupIndex: not in consensus group")
		return false
	}

	signatureShare, err := sr.SigningHandler().CreateSignatureShareForPublicKey(
		sr.GetData(),
		uint16(selfIndex),
		sr.Header.GetEpoch(),
		[]byte(sr.SelfPubKey()),
	)
	if err != nil {
		log.Debug("doSignatureJob.CreateSignatureShareForPublicKey", "error", err.Error())
		return false
	}

	// leader already sent his signature on subround block
	if !isSelfLeader {
		ok := sr.createAndSendSignatureMessage(signatureShare, []byte(sr.SelfPubKey()))
		if !ok {
			return false
		}
	}

	shouldWaitForAllSigsAsync := isSelfLeader && !isFlagActive
	return sr.completeSignatureSubRound(sr.SelfPubKey(), shouldWaitForAllSigsAsync)
}

// IsInterfaceNil returns true if there is no value under the interface
func (sr *subroundSignature) IsInterfaceNil() bool {
	return sr == nil
}
