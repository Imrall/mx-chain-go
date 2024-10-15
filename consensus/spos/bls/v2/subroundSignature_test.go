package v2_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/check"
	"github.com/multiversx/mx-chain-core-go/data"
	"github.com/multiversx/mx-chain-core-go/data/block"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/multiversx/mx-chain-go/common"
	"github.com/multiversx/mx-chain-go/consensus"
	"github.com/multiversx/mx-chain-go/consensus/spos"
	"github.com/multiversx/mx-chain-go/consensus/spos/bls"
	v2 "github.com/multiversx/mx-chain-go/consensus/spos/bls/v2"
	dataRetrieverMock "github.com/multiversx/mx-chain-go/dataRetriever/mock"
	"github.com/multiversx/mx-chain-go/testscommon"
	consensusMocks "github.com/multiversx/mx-chain-go/testscommon/consensus"
	"github.com/multiversx/mx-chain-go/testscommon/consensus/initializers"
	"github.com/multiversx/mx-chain-go/testscommon/enableEpochsHandlerMock"
	"github.com/multiversx/mx-chain-go/testscommon/statusHandler"
)

const setThresholdJobsDone = "threshold"

func initSubroundSignatureWithContainer(container *consensusMocks.ConsensusCoreMock) v2.SubroundSignature {
	consensusState := initializers.InitConsensusState()
	ch := make(chan bool, 1)

	sr, _ := spos.NewSubround(
		bls.SrBlock,
		bls.SrSignature,
		bls.SrEndRound,
		int64(70*roundTimeDuration/100),
		int64(85*roundTimeDuration/100),
		"(SIGNATURE)",
		consensusState,
		ch,
		executeStoredMessages,
		container,
		chainID,
		currentPid,
		&statusHandler.AppStatusHandlerStub{},
	)

	srSignature, _ := v2.NewSubroundSignature(
		sr,
		&statusHandler.AppStatusHandlerStub{},
		&testscommon.SentSignatureTrackerStub{},
		&consensusMocks.SposWorkerMock{},
		&dataRetrieverMock.ThrottlerStub{},
	)

	return srSignature
}

func initSubroundSignature() v2.SubroundSignature {
	container := consensusMocks.InitConsensusCore()
	return initSubroundSignatureWithContainer(container)
}

func TestNewSubroundSignature(t *testing.T) {
	t.Parallel()

	container := consensusMocks.InitConsensusCore()
	consensusState := initializers.InitConsensusState()
	ch := make(chan bool, 1)

	sr, _ := spos.NewSubround(
		bls.SrBlock,
		bls.SrSignature,
		bls.SrEndRound,
		int64(70*roundTimeDuration/100),
		int64(85*roundTimeDuration/100),
		"(SIGNATURE)",
		consensusState,
		ch,
		executeStoredMessages,
		container,
		chainID,
		currentPid,
		&statusHandler.AppStatusHandlerStub{},
	)

	t.Run("nil subround should error", func(t *testing.T) {
		t.Parallel()

		srSignature, err := v2.NewSubroundSignature(
			nil,
			&statusHandler.AppStatusHandlerStub{},
			&testscommon.SentSignatureTrackerStub{},
			&consensusMocks.SposWorkerMock{},
			&dataRetrieverMock.ThrottlerStub{},
		)

		assert.Nil(t, srSignature)
		assert.Equal(t, spos.ErrNilSubround, err)
	})
	t.Run("nil worker should error", func(t *testing.T) {
		t.Parallel()

		srSignature, err := v2.NewSubroundSignature(
			sr,
			&statusHandler.AppStatusHandlerStub{},
			&testscommon.SentSignatureTrackerStub{},
			nil,
			&dataRetrieverMock.ThrottlerStub{},
		)

		assert.Nil(t, srSignature)
		assert.Equal(t, spos.ErrNilWorker, err)
	})
	t.Run("nil app status handler should error", func(t *testing.T) {
		t.Parallel()

		srSignature, err := v2.NewSubroundSignature(
			sr,
			nil,
			&testscommon.SentSignatureTrackerStub{},
			&consensusMocks.SposWorkerMock{},
			&dataRetrieverMock.ThrottlerStub{},
		)

		assert.Nil(t, srSignature)
		assert.Equal(t, spos.ErrNilAppStatusHandler, err)
	})
	t.Run("nil sent signatures tracker should error", func(t *testing.T) {
		t.Parallel()

		srSignature, err := v2.NewSubroundSignature(
			sr,
			&statusHandler.AppStatusHandlerStub{},
			nil,
			&consensusMocks.SposWorkerMock{},
			&dataRetrieverMock.ThrottlerStub{},
		)

		assert.Nil(t, srSignature)
		assert.Equal(t, v2.ErrNilSentSignatureTracker, err)
	})

	t.Run("nil signatureThrottler should error", func(t *testing.T) {
		t.Parallel()

		srSignature, err := v2.NewSubroundSignature(
			sr,
			&statusHandler.AppStatusHandlerStub{},
			&testscommon.SentSignatureTrackerStub{},
			&consensusMocks.SposWorkerMock{},
			nil,
		)

		assert.Nil(t, srSignature)
		assert.Equal(t, spos.ErrNilThrottler, err)
	})
}

func TestSubroundSignature_NewSubroundSignatureNilConsensusStateShouldFail(t *testing.T) {
	t.Parallel()

	container := consensusMocks.InitConsensusCore()
	consensusState := initializers.InitConsensusState()
	ch := make(chan bool, 1)

	sr, _ := spos.NewSubround(
		bls.SrBlock,
		bls.SrSignature,
		bls.SrEndRound,
		int64(70*roundTimeDuration/100),
		int64(85*roundTimeDuration/100),
		"(SIGNATURE)",
		consensusState,
		ch,
		executeStoredMessages,
		container,
		chainID,
		currentPid,
		&statusHandler.AppStatusHandlerStub{},
	)

	sr.ConsensusStateHandler = nil
	srSignature, err := v2.NewSubroundSignature(
		sr,
		&statusHandler.AppStatusHandlerStub{},
		&testscommon.SentSignatureTrackerStub{},
		&consensusMocks.SposWorkerMock{},
		&dataRetrieverMock.ThrottlerStub{},
	)

	assert.True(t, check.IfNil(srSignature))
	assert.Equal(t, spos.ErrNilConsensusState, err)
}

func TestSubroundSignature_NewSubroundSignatureNilHasherShouldFail(t *testing.T) {
	t.Parallel()

	container := consensusMocks.InitConsensusCore()
	consensusState := initializers.InitConsensusState()
	ch := make(chan bool, 1)

	sr, _ := spos.NewSubround(
		bls.SrBlock,
		bls.SrSignature,
		bls.SrEndRound,
		int64(70*roundTimeDuration/100),
		int64(85*roundTimeDuration/100),
		"(SIGNATURE)",
		consensusState,
		ch,
		executeStoredMessages,
		container,
		chainID,
		currentPid,
		&statusHandler.AppStatusHandlerStub{},
	)
	container.SetHasher(nil)
	srSignature, err := v2.NewSubroundSignature(
		sr,
		&statusHandler.AppStatusHandlerStub{},
		&testscommon.SentSignatureTrackerStub{},
		&consensusMocks.SposWorkerMock{},
		&dataRetrieverMock.ThrottlerStub{},
	)

	assert.True(t, check.IfNil(srSignature))
	assert.Equal(t, spos.ErrNilHasher, err)
}

func TestSubroundSignature_NewSubroundSignatureNilMultiSignerContainerShouldFail(t *testing.T) {
	t.Parallel()

	container := consensusMocks.InitConsensusCore()
	consensusState := initializers.InitConsensusState()
	ch := make(chan bool, 1)

	sr, _ := spos.NewSubround(
		bls.SrBlock,
		bls.SrSignature,
		bls.SrEndRound,
		int64(70*roundTimeDuration/100),
		int64(85*roundTimeDuration/100),
		"(SIGNATURE)",
		consensusState,
		ch,
		executeStoredMessages,
		container,
		chainID,
		currentPid,
		&statusHandler.AppStatusHandlerStub{},
	)
	container.SetMultiSignerContainer(nil)
	srSignature, err := v2.NewSubroundSignature(
		sr,
		&statusHandler.AppStatusHandlerStub{},
		&testscommon.SentSignatureTrackerStub{},
		&consensusMocks.SposWorkerMock{},
		&dataRetrieverMock.ThrottlerStub{},
	)

	assert.True(t, check.IfNil(srSignature))
	assert.Equal(t, spos.ErrNilMultiSignerContainer, err)
}

func TestSubroundSignature_NewSubroundSignatureNilRoundHandlerShouldFail(t *testing.T) {
	t.Parallel()

	container := consensusMocks.InitConsensusCore()
	consensusState := initializers.InitConsensusState()
	ch := make(chan bool, 1)

	sr, _ := spos.NewSubround(
		bls.SrBlock,
		bls.SrSignature,
		bls.SrEndRound,
		int64(70*roundTimeDuration/100),
		int64(85*roundTimeDuration/100),
		"(SIGNATURE)",
		consensusState,
		ch,
		executeStoredMessages,
		container,
		chainID,
		currentPid,
		&statusHandler.AppStatusHandlerStub{},
	)
	container.SetRoundHandler(nil)

	srSignature, err := v2.NewSubroundSignature(
		sr,
		&statusHandler.AppStatusHandlerStub{},
		&testscommon.SentSignatureTrackerStub{},
		&consensusMocks.SposWorkerMock{},
		&dataRetrieverMock.ThrottlerStub{},
	)

	assert.True(t, check.IfNil(srSignature))
	assert.Equal(t, spos.ErrNilRoundHandler, err)
}

func TestSubroundSignature_NewSubroundSignatureNilSyncTimerShouldFail(t *testing.T) {
	t.Parallel()

	container := consensusMocks.InitConsensusCore()
	consensusState := initializers.InitConsensusState()
	ch := make(chan bool, 1)

	sr, _ := spos.NewSubround(
		bls.SrBlock,
		bls.SrSignature,
		bls.SrEndRound,
		int64(70*roundTimeDuration/100),
		int64(85*roundTimeDuration/100),
		"(SIGNATURE)",
		consensusState,
		ch,
		executeStoredMessages,
		container,
		chainID,
		currentPid,
		&statusHandler.AppStatusHandlerStub{},
	)
	container.SetSyncTimer(nil)
	srSignature, err := v2.NewSubroundSignature(
		sr,
		&statusHandler.AppStatusHandlerStub{},
		&testscommon.SentSignatureTrackerStub{},
		&consensusMocks.SposWorkerMock{},
		&dataRetrieverMock.ThrottlerStub{},
	)

	assert.True(t, check.IfNil(srSignature))
	assert.Equal(t, spos.ErrNilSyncTimer, err)
}

func TestSubroundSignature_NewSubroundSignatureNilAppStatusHandlerShouldFail(t *testing.T) {
	t.Parallel()

	container := consensusMocks.InitConsensusCore()
	consensusState := initializers.InitConsensusState()
	ch := make(chan bool, 1)

	sr, _ := spos.NewSubround(
		bls.SrBlock,
		bls.SrSignature,
		bls.SrEndRound,
		int64(70*roundTimeDuration/100),
		int64(85*roundTimeDuration/100),
		"(SIGNATURE)",
		consensusState,
		ch,
		executeStoredMessages,
		container,
		chainID,
		currentPid,
		&statusHandler.AppStatusHandlerStub{},
	)

	srSignature, err := v2.NewSubroundSignature(
		sr,
		nil,
		&testscommon.SentSignatureTrackerStub{},
		&consensusMocks.SposWorkerMock{},
		&dataRetrieverMock.ThrottlerStub{},
	)

	assert.True(t, check.IfNil(srSignature))
	assert.Equal(t, spos.ErrNilAppStatusHandler, err)
}

func TestSubroundSignature_NewSubroundSignatureShouldWork(t *testing.T) {
	t.Parallel()

	container := consensusMocks.InitConsensusCore()
	consensusState := initializers.InitConsensusState()
	ch := make(chan bool, 1)

	sr, _ := spos.NewSubround(
		bls.SrBlock,
		bls.SrSignature,
		bls.SrEndRound,
		int64(70*roundTimeDuration/100),
		int64(85*roundTimeDuration/100),
		"(SIGNATURE)",
		consensusState,
		ch,
		executeStoredMessages,
		container,
		chainID,
		currentPid,
		&statusHandler.AppStatusHandlerStub{},
	)

	srSignature, err := v2.NewSubroundSignature(
		sr,
		&statusHandler.AppStatusHandlerStub{},
		&testscommon.SentSignatureTrackerStub{},
		&consensusMocks.SposWorkerMock{},
		&dataRetrieverMock.ThrottlerStub{},
	)

	assert.False(t, check.IfNil(srSignature))
	assert.Nil(t, err)
}

func TestSubroundSignature_DoSignatureJob(t *testing.T) {
	t.Parallel()

	t.Run("with equivalent messages flag inactive", func(t *testing.T) {
		t.Parallel()

		container := consensusMocks.InitConsensusCore()
		sr := initSubroundSignatureWithContainer(container)

		sr.SetHeader(&block.Header{})
		sr.SetData(nil)
		r := sr.DoSignatureJob()
		assert.False(t, r)

		sr.SetData([]byte("X"))

		sr.SetHeader(nil)
		r = sr.DoSignatureJob()
		assert.False(t, r)

		sr.SetHeader(&block.Header{})

		err := errors.New("create signature share error")
		signingHandler := &consensusMocks.SigningHandlerStub{
			CreateSignatureShareForPublicKeyCalled: func(msg []byte, index uint16, epoch uint32, publicKeyBytes []byte) ([]byte, error) {
				return nil, err
			},
		}
		container.SetSigningHandler(signingHandler)

		r = sr.DoSignatureJob()
		assert.False(t, r)

		signingHandler = &consensusMocks.SigningHandlerStub{
			CreateSignatureShareForPublicKeyCalled: func(msg []byte, index uint16, epoch uint32, publicKeyBytes []byte) ([]byte, error) {
				return []byte("SIG"), nil
			},
		}
		container.SetSigningHandler(signingHandler)

		r = sr.DoSignatureJob()
		assert.True(t, r)

		sr.SetSelfPubKey("OTHER")
		r = sr.DoSignatureJob()
		assert.False(t, r)

		sr.SetSelfPubKey(sr.ConsensusGroup()[2])
		container.SetBroadcastMessenger(&consensusMocks.BroadcastMessengerMock{
			BroadcastConsensusMessageCalled: func(message *consensus.Message) error {
				return expectedErr
			},
		})
		r = sr.DoSignatureJob()
		assert.False(t, r)

		container.SetBroadcastMessenger(&consensusMocks.BroadcastMessengerMock{
			BroadcastConsensusMessageCalled: func(message *consensus.Message) error {
				return nil
			},
		})
		_ = sr.SetJobDone(sr.SelfPubKey(), bls.SrSignature, false)
		sr.SetRoundCanceled(false)
		leader, err := sr.GetLeader()
		assert.Nil(t, err)

		sr.SetSelfPubKey(leader)
		r = sr.DoSignatureJob()
		assert.True(t, r)
		assert.False(t, sr.GetRoundCanceled())
	})
	t.Run("with equivalent messages flag active should work", func(t *testing.T) {
		t.Parallel()

		container := consensusMocks.InitConsensusCore()
		enableEpochsHandler := &enableEpochsHandlerMock.EnableEpochsHandlerStub{
			IsFlagEnabledInEpochCalled: func(flag core.EnableEpochFlag, epoch uint32) bool {
				return flag == common.EquivalentMessagesFlag
			},
		}
		container.SetEnableEpochsHandler(enableEpochsHandler)
		sr := initSubroundSignatureWithContainer(container)

		sr.SetHeader(&block.Header{})
		leader, err := sr.GetLeader()
		assert.Nil(t, err)
		sr.SetSelfPubKey(leader)
		container.SetBroadcastMessenger(&consensusMocks.BroadcastMessengerMock{
			BroadcastConsensusMessageCalled: func(message *consensus.Message) error {
				assert.Fail(t, "should have not been called")
				return nil
			},
		})
		r := sr.DoSignatureJob()
		assert.True(t, r)

		assert.False(t, sr.GetRoundCanceled())
		assert.Nil(t, err)
		leaderJobDone, err := sr.JobDone(leader, bls.SrSignature)
		assert.NoError(t, err)
		assert.True(t, leaderJobDone)
		assert.True(t, sr.IsSubroundFinished(bls.SrSignature))
	})
}

func TestSubroundSignature_DoSignatureJobWithMultikey(t *testing.T) {
	t.Parallel()

	t.Run("with equivalent messages flag inactive", func(t *testing.T) {
		t.Parallel()

		container := consensusMocks.InitConsensusCore()
		consensusState := initializers.InitConsensusStateWithKeysHandler(
			&testscommon.KeysHandlerStub{
				IsKeyManagedByCurrentNodeCalled: func(pkBytes []byte) bool {
					return true
				},
			},
		)
		ch := make(chan bool, 1)

		sr, _ := spos.NewSubround(
			bls.SrBlock,
			bls.SrSignature,
			bls.SrEndRound,
			int64(70*roundTimeDuration/100),
			int64(85*roundTimeDuration/100),
			"(SIGNATURE)",
			consensusState,
			ch,
			executeStoredMessages,
			container,
			chainID,
			currentPid,
			&statusHandler.AppStatusHandlerStub{},
		)

		signatureSentForPks := make(map[string]struct{})
		mutex := sync.Mutex{}
		srSignature, _ := v2.NewSubroundSignature(
			sr,
			&statusHandler.AppStatusHandlerStub{},
			&testscommon.SentSignatureTrackerStub{
				SignatureSentCalled: func(pkBytes []byte) {
					mutex.Lock()
					signatureSentForPks[string(pkBytes)] = struct{}{}
					mutex.Unlock()
				},
			},
			&consensusMocks.SposWorkerMock{},
			&dataRetrieverMock.ThrottlerStub{},
		)

		srSignature.SetHeader(&block.Header{})
		srSignature.SetData(nil)
		r := srSignature.DoSignatureJob()
		assert.False(t, r)

		sr.SetData([]byte("X"))

		err := errors.New("create signature share error")
		signingHandler := &consensusMocks.SigningHandlerStub{
			CreateSignatureShareForPublicKeyCalled: func(msg []byte, index uint16, epoch uint32, publicKeyBytes []byte) ([]byte, error) {
				return nil, err
			},
		}
		container.SetSigningHandler(signingHandler)

		r = srSignature.DoSignatureJob()
		assert.False(t, r)

		signingHandler = &consensusMocks.SigningHandlerStub{
			CreateSignatureShareForPublicKeyCalled: func(msg []byte, index uint16, epoch uint32, publicKeyBytes []byte) ([]byte, error) {
				return []byte("SIG"), nil
			},
		}
		container.SetSigningHandler(signingHandler)

		r = srSignature.DoSignatureJob()
		assert.True(t, r)

		_ = sr.SetJobDone(sr.SelfPubKey(), bls.SrSignature, false)
		sr.SetRoundCanceled(false)
		leader, err := sr.GetLeader()
		assert.Nil(t, err)
		sr.SetSelfPubKey(leader)
		r = srSignature.DoSignatureJob()
		assert.True(t, r)
		assert.False(t, sr.GetRoundCanceled())
		expectedMap := map[string]struct{}{
			"A": {},
			"B": {},
			"C": {},
			"D": {},
			"E": {},
			"F": {},
			"G": {},
			"H": {},
			"I": {},
		}
		assert.Equal(t, expectedMap, signatureSentForPks)
	})
	t.Run("with equivalent messages flag active should work", func(t *testing.T) {
		t.Parallel()

		container := consensusMocks.InitConsensusCore()
		enableEpochsHandler := &enableEpochsHandlerMock.EnableEpochsHandlerStub{
			IsFlagEnabledInEpochCalled: func(flag core.EnableEpochFlag, epoch uint32) bool {
				return flag == common.EquivalentMessagesFlag
			},
		}
		container.SetEnableEpochsHandler(enableEpochsHandler)

		signingHandler := &consensusMocks.SigningHandlerStub{
			CreateSignatureShareForPublicKeyCalled: func(msg []byte, index uint16, epoch uint32, publicKeyBytes []byte) ([]byte, error) {
				return []byte("SIG"), nil
			},
		}
		container.SetSigningHandler(signingHandler)
		consensusState := initializers.InitConsensusStateWithKeysHandler(
			&testscommon.KeysHandlerStub{
				IsKeyManagedByCurrentNodeCalled: func(pkBytes []byte) bool {
					return true
				},
			},
		)
		ch := make(chan bool, 1)

		sr, _ := spos.NewSubround(
			bls.SrBlock,
			bls.SrSignature,
			bls.SrEndRound,
			int64(70*roundTimeDuration/100),
			int64(85*roundTimeDuration/100),
			"(SIGNATURE)",
			consensusState,
			ch,
			executeStoredMessages,
			container,
			chainID,
			currentPid,
			&statusHandler.AppStatusHandlerStub{},
		)

		signatureSentForPks := make(map[string]struct{})
		mutex := sync.Mutex{}
		srSignature, _ := v2.NewSubroundSignature(
			sr,
			&statusHandler.AppStatusHandlerStub{},
			&testscommon.SentSignatureTrackerStub{
				SignatureSentCalled: func(pkBytes []byte) {
					mutex.Lock()
					signatureSentForPks[string(pkBytes)] = struct{}{}
					mutex.Unlock()
				},
			},
			&consensusMocks.SposWorkerMock{},
			&dataRetrieverMock.ThrottlerStub{},
		)

		sr.SetHeader(&block.Header{})
		signaturesBroadcast := make(map[string]int)
		container.SetBroadcastMessenger(&consensusMocks.BroadcastMessengerMock{
			BroadcastConsensusMessageCalled: func(message *consensus.Message) error {
				mutex.Lock()
				signaturesBroadcast[string(message.PubKey)]++
				mutex.Unlock()
				return nil
			},
		})

		sr.SetSelfPubKey("OTHER")

		r := srSignature.DoSignatureJob()
		assert.True(t, r)

		assert.False(t, sr.GetRoundCanceled())
		assert.True(t, sr.IsSubroundFinished(bls.SrSignature))

		for _, pk := range sr.ConsensusGroup() {
			isJobDone, err := sr.JobDone(pk, bls.SrSignature)
			assert.NoError(t, err)
			assert.True(t, isJobDone)
		}

		expectedMap := map[string]struct{}{
			"A": {},
			"B": {},
			"C": {},
			"D": {},
			"E": {},
			"F": {},
			"G": {},
			"H": {},
			"I": {},
		}
		assert.Equal(t, expectedMap, signatureSentForPks)

		expectedBroadcastMap := map[string]int{
			"B": 1,
			"C": 1,
			"D": 1,
			"E": 1,
			"F": 1,
			"G": 1,
			"H": 1,
			"I": 1,
		}
		assert.Equal(t, expectedBroadcastMap, signaturesBroadcast)
	})
}

func TestSubroundSignature_SendSignature(t *testing.T) {

	t.Parallel()

	t.Run("sendSignatureForManagedKey will return false because of error", func(t *testing.T) {
		t.Parallel()

		container := consensusMocks.InitConsensusCore()

		container.SetSigningHandler(&consensusMocks.SigningHandlerStub{
			CreateSignatureShareForPublicKeyCalled: func(message []byte, index uint16, epoch uint32, publicKeyBytes []byte) ([]byte, error) {
				return make([]byte, 0), expErr
			},
		})
		consensusState := initializers.InitConsensusStateWithKeysHandler(
			&testscommon.KeysHandlerStub{
				IsKeyManagedByCurrentNodeCalled: func(pkBytes []byte) bool {
					return true
				},
			},
		)

		ch := make(chan bool, 1)

		sr, _ := spos.NewSubround(
			bls.SrBlock,
			bls.SrSignature,
			bls.SrEndRound,
			int64(70*roundTimeDuration/100),
			int64(85*roundTimeDuration/100),
			"(SIGNATURE)",
			consensusState,
			ch,
			executeStoredMessages,
			container,
			chainID,
			currentPid,
			&statusHandler.AppStatusHandlerStub{},
		)
		sr.SetHeader(&block.Header{})

		signatureSentForPks := make(map[string]struct{})
		srSignature, _ := v2.NewSubroundSignature(
			sr,
			&statusHandler.AppStatusHandlerStub{},
			&testscommon.SentSignatureTrackerStub{
				SignatureSentCalled: func(pkBytes []byte) {
					signatureSentForPks[string(pkBytes)] = struct{}{}
				},
			},
			&consensusMocks.SposWorkerMock{},
			&dataRetrieverMock.ThrottlerStub{},
		)

		r := srSignature.SendSignatureForManagedKey(0, "a")

		assert.False(t, r)
	})

	t.Run("sendSignatureForManagedKey should be false", func(t *testing.T) {
		t.Parallel()

		container := consensusMocks.InitConsensusCore()
		container.SetSigningHandler(&consensusMocks.SigningHandlerStub{
			CreateSignatureShareForPublicKeyCalled: func(message []byte, index uint16, epoch uint32, publicKeyBytes []byte) ([]byte, error) {
				return []byte("SIG"), nil
			},
		})

		enableEpochsHandler := &enableEpochsHandlerMock.EnableEpochsHandlerStub{
			IsFlagEnabledInEpochCalled: func(flag core.EnableEpochFlag, epoch uint32) bool {
				return flag == common.EquivalentMessagesFlag
			},
		}
		container.SetEnableEpochsHandler(enableEpochsHandler)

		container.SetBroadcastMessenger(&consensusMocks.BroadcastMessengerMock{
			BroadcastConsensusMessageCalled: func(message *consensus.Message) error {
				return fmt.Errorf("error")
			},
		})
		consensusState := initializers.InitConsensusStateWithKeysHandler(
			&testscommon.KeysHandlerStub{
				IsKeyManagedByCurrentNodeCalled: func(pkBytes []byte) bool {
					return true
				},
			},
		)

		ch := make(chan bool, 1)

		sr, _ := spos.NewSubround(
			bls.SrBlock,
			bls.SrSignature,
			bls.SrEndRound,
			int64(70*roundTimeDuration/100),
			int64(85*roundTimeDuration/100),
			"(SIGNATURE)",
			consensusState,
			ch,
			executeStoredMessages,
			container,
			chainID,
			currentPid,
			&statusHandler.AppStatusHandlerStub{},
		)
		sr.SetHeader(&block.Header{})

		signatureSentForPks := make(map[string]struct{})
		srSignature, _ := v2.NewSubroundSignature(
			sr,
			&statusHandler.AppStatusHandlerStub{},
			&testscommon.SentSignatureTrackerStub{
				SignatureSentCalled: func(pkBytes []byte) {
					signatureSentForPks[string(pkBytes)] = struct{}{}
				},
			},
			&consensusMocks.SposWorkerMock{},
			&dataRetrieverMock.ThrottlerStub{},
		)

		r := srSignature.SendSignatureForManagedKey(1, "a")

		assert.False(t, r)
	})

	t.Run("SentSignature should be called", func(t *testing.T) {
		t.Parallel()

		container := consensusMocks.InitConsensusCore()
		container.SetSigningHandler(&consensusMocks.SigningHandlerStub{
			CreateSignatureShareForPublicKeyCalled: func(message []byte, index uint16, epoch uint32, publicKeyBytes []byte) ([]byte, error) {
				return []byte("SIG"), nil
			},
		})

		enableEpochsHandler := &enableEpochsHandlerMock.EnableEpochsHandlerStub{
			IsFlagEnabledInEpochCalled: func(flag core.EnableEpochFlag, epoch uint32) bool {
				return flag == common.EquivalentMessagesFlag
			},
		}
		container.SetEnableEpochsHandler(enableEpochsHandler)

		container.SetBroadcastMessenger(&consensusMocks.BroadcastMessengerMock{
			BroadcastConsensusMessageCalled: func(message *consensus.Message) error {
				return nil
			},
		})
		consensusState := initializers.InitConsensusStateWithKeysHandler(
			&testscommon.KeysHandlerStub{
				IsKeyManagedByCurrentNodeCalled: func(pkBytes []byte) bool {
					return true
				},
			},
		)

		ch := make(chan bool, 1)

		sr, _ := spos.NewSubround(
			bls.SrBlock,
			bls.SrSignature,
			bls.SrEndRound,
			int64(70*roundTimeDuration/100),
			int64(85*roundTimeDuration/100),
			"(SIGNATURE)",
			consensusState,
			ch,
			executeStoredMessages,
			container,
			chainID,
			currentPid,
			&statusHandler.AppStatusHandlerStub{},
		)
		sr.SetHeader(&block.Header{})

		signatureSentForPks := make(map[string]struct{})
		varCalled := false
		srSignature, _ := v2.NewSubroundSignature(
			sr,
			&statusHandler.AppStatusHandlerStub{},
			&testscommon.SentSignatureTrackerStub{
				SignatureSentCalled: func(pkBytes []byte) {
					signatureSentForPks[string(pkBytes)] = struct{}{}
					varCalled = true
				},
			},
			&consensusMocks.SposWorkerMock{},
			&dataRetrieverMock.ThrottlerStub{},
		)

		_ = srSignature.SendSignatureForManagedKey(1, "a")

		assert.True(t, varCalled)
	})
}

func TestSubroundSignature_DoSignatureJobForManagedKeys(t *testing.T) {

	t.Parallel()

	t.Run("should work", func(t *testing.T) {
		t.Parallel()
		container := consensusMocks.InitConsensusCore()
		enableEpochsHandler := &enableEpochsHandlerMock.EnableEpochsHandlerStub{
			IsFlagEnabledInEpochCalled: func(flag core.EnableEpochFlag, epoch uint32) bool {
				return flag == common.EquivalentMessagesFlag
			},
		}
		container.SetEnableEpochsHandler(enableEpochsHandler)

		signingHandler := &consensusMocks.SigningHandlerStub{
			CreateSignatureShareForPublicKeyCalled: func(msg []byte, index uint16, epoch uint32, publicKeyBytes []byte) ([]byte, error) {
				return []byte("SIG"), nil
			},
		}
		container.SetSigningHandler(signingHandler)
		consensusState := initializers.InitConsensusStateWithKeysHandler(
			&testscommon.KeysHandlerStub{
				IsKeyManagedByCurrentNodeCalled: func(pkBytes []byte) bool {
					return true
				},
			},
		)
		ch := make(chan bool, 1)

		sr, _ := spos.NewSubround(
			bls.SrBlock,
			bls.SrSignature,
			bls.SrEndRound,
			int64(70*roundTimeDuration/100),
			int64(85*roundTimeDuration/100),
			"(SIGNATURE)",
			consensusState,
			ch,
			executeStoredMessages,
			container,
			chainID,
			currentPid,
			&statusHandler.AppStatusHandlerStub{},
		)

		signatureSentForPks := make(map[string]struct{})
		mutex := sync.Mutex{}
		srSignature, _ := v2.NewSubroundSignature(
			sr,
			&statusHandler.AppStatusHandlerStub{},
			&testscommon.SentSignatureTrackerStub{
				SignatureSentCalled: func(pkBytes []byte) {
					mutex.Lock()
					signatureSentForPks[string(pkBytes)] = struct{}{}
					mutex.Unlock()
				},
			},
			&consensusMocks.SposWorkerMock{},
			&dataRetrieverMock.ThrottlerStub{},
		)

		sr.SetHeader(&block.Header{})
		signaturesBroadcast := make(map[string]int)
		container.SetBroadcastMessenger(&consensusMocks.BroadcastMessengerMock{
			BroadcastConsensusMessageCalled: func(message *consensus.Message) error {
				mutex.Lock()
				signaturesBroadcast[string(message.PubKey)]++
				mutex.Unlock()
				return nil
			},
		})

		sr.SetSelfPubKey("OTHER")

		r := srSignature.DoSignatureJobForManagedKeys(context.TODO())
		assert.True(t, r)

		for _, pk := range sr.ConsensusGroup() {
			isJobDone, err := sr.JobDone(pk, bls.SrSignature)
			assert.NoError(t, err)
			assert.True(t, isJobDone)
		}

		expectedMap := map[string]struct{}{
			"A": {},
			"B": {},
			"C": {},
			"D": {},
			"E": {},
			"F": {},
			"G": {},
			"H": {},
			"I": {},
		}
		assert.Equal(t, expectedMap, signatureSentForPks)

		expectedBroadcastMap := map[string]int{
			"B": 1,
			"C": 1,
			"D": 1,
			"E": 1,
			"F": 1,
			"G": 1,
			"H": 1,
			"I": 1,
		}
		assert.Equal(t, expectedBroadcastMap, signaturesBroadcast)

	})

	t.Run("should fail", func(t *testing.T) {
		t.Parallel()
		container := consensusMocks.InitConsensusCore()
		enableEpochsHandler := &enableEpochsHandlerMock.EnableEpochsHandlerStub{
			IsFlagEnabledInEpochCalled: func(flag core.EnableEpochFlag, epoch uint32) bool {
				return flag == common.EquivalentMessagesFlag
			},
		}
		container.SetEnableEpochsHandler(enableEpochsHandler)

		consensusState := initializers.InitConsensusStateWithKeysHandler(
			&testscommon.KeysHandlerStub{
				IsKeyManagedByCurrentNodeCalled: func(pkBytes []byte) bool {
					return true
				},
			},
		)
		ch := make(chan bool, 1)

		sr, _ := spos.NewSubround(
			bls.SrBlock,
			bls.SrSignature,
			bls.SrEndRound,
			int64(70*roundTimeDuration/100),
			int64(85*roundTimeDuration/100),
			"(SIGNATURE)",
			consensusState,
			ch,
			executeStoredMessages,
			container,
			chainID,
			currentPid,
			&statusHandler.AppStatusHandlerStub{},
		)

		srSignature, _ := v2.NewSubroundSignature(
			sr,
			&statusHandler.AppStatusHandlerStub{},
			&testscommon.SentSignatureTrackerStub{},
			&consensusMocks.SposWorkerMock{},
			&dataRetrieverMock.ThrottlerStub{
				CanProcessCalled: func() bool {
					return false
				},
			},
		)

		sr.SetHeader(&block.Header{})
		ctx, cancel := context.WithCancel(context.TODO())
		cancel()
		r := srSignature.DoSignatureJobForManagedKeys(ctx)
		assert.False(t, r)

	})

}

func TestSubroundSignature_ReceivedSignature(t *testing.T) {
	t.Parallel()

	container := consensusMocks.InitConsensusCore()
	sr := initSubroundSignatureWithContainer(container)
	signature := []byte("signature")
	cnsMsg := consensus.NewConsensusMessage(
		sr.GetData(),
		signature,
		nil,
		nil,
		[]byte(sr.ConsensusGroup()[1]),
		[]byte("sig"),
		int(bls.MtSignature),
		0,
		chainID,
		nil,
		nil,
		nil,
		currentPid,
		nil,
	)

	sr.SetHeader(&block.Header{})
	sr.SetData(nil)
	r := sr.ReceivedSignature(cnsMsg)
	assert.False(t, r)

	sr.SetData([]byte("Y"))
	r = sr.ReceivedSignature(cnsMsg)
	assert.False(t, r)

	sr.SetData([]byte("X"))
	r = sr.ReceivedSignature(cnsMsg)
	assert.False(t, r)

	leader, err := sr.GetLeader()
	assert.Nil(t, err)
	sr.SetSelfPubKey(leader)

	cnsMsg.PubKey = []byte("X")
	r = sr.ReceivedSignature(cnsMsg)
	assert.False(t, r)

	cnsMsg.PubKey = []byte(sr.ConsensusGroup()[1])
	maxCount := len(sr.ConsensusGroup()) * 2 / 3
	count := 0
	for i := 0; i < len(sr.ConsensusGroup()); i++ {
		if sr.ConsensusGroup()[i] != string(cnsMsg.PubKey) {
			_ = sr.SetJobDone(sr.ConsensusGroup()[i], bls.SrSignature, true)
			count++
			if count == maxCount {
				break
			}
		}
	}
	r = sr.ReceivedSignature(cnsMsg)
	assert.True(t, r)

	enableEpochsHandler := &enableEpochsHandlerMock.EnableEpochsHandlerStub{
		IsFlagEnabledInEpochCalled: func(flag core.EnableEpochFlag, epoch uint32) bool {
			return flag == common.EquivalentMessagesFlag
		},
	}
	container.SetEnableEpochsHandler(enableEpochsHandler)
	r = sr.ReceivedSignature(cnsMsg)
	assert.True(t, r)
}

func TestSubroundSignature_ReceivedSignatureStoreShareFailed(t *testing.T) {
	t.Parallel()

	errStore := errors.New("signature share store failed")
	storeSigShareCalled := false
	signingHandler := &consensusMocks.SigningHandlerStub{
		VerifySignatureShareCalled: func(index uint16, sig, msg []byte, epoch uint32) error {
			return nil
		},
		StoreSignatureShareCalled: func(index uint16, sig []byte) error {
			storeSigShareCalled = true
			return errStore
		},
	}

	container := consensusMocks.InitConsensusCore()
	container.SetSigningHandler(signingHandler)
	sr := initSubroundSignatureWithContainer(container)
	sr.SetHeader(&block.Header{})

	signature := []byte("signature")
	cnsMsg := consensus.NewConsensusMessage(
		sr.GetData(),
		signature,
		nil,
		nil,
		[]byte(sr.ConsensusGroup()[1]),
		[]byte("sig"),
		int(bls.MtSignature),
		0,
		chainID,
		nil,
		nil,
		nil,
		currentPid,
		nil,
	)

	sr.SetData(nil)
	r := sr.ReceivedSignature(cnsMsg)
	assert.False(t, r)

	sr.SetData([]byte("Y"))
	r = sr.ReceivedSignature(cnsMsg)
	assert.False(t, r)

	sr.SetData([]byte("X"))
	r = sr.ReceivedSignature(cnsMsg)
	assert.False(t, r)

	leader, err := sr.GetLeader()
	assert.Nil(t, err)
	sr.SetSelfPubKey(leader)

	cnsMsg.PubKey = []byte("X")
	r = sr.ReceivedSignature(cnsMsg)
	assert.False(t, r)

	cnsMsg.PubKey = []byte(sr.ConsensusGroup()[1])
	maxCount := len(sr.ConsensusGroup()) * 2 / 3
	count := 0
	for i := 0; i < len(sr.ConsensusGroup()); i++ {
		if sr.ConsensusGroup()[i] != string(cnsMsg.PubKey) {
			_ = sr.SetJobDone(sr.ConsensusGroup()[i], bls.SrSignature, true)
			count++
			if count == maxCount {
				break
			}
		}
	}
	r = sr.ReceivedSignature(cnsMsg)
	assert.False(t, r)
	assert.True(t, storeSigShareCalled)
}

func TestSubroundSignature_SignaturesCollected(t *testing.T) {
	t.Parallel()

	sr := initSubroundSignature()

	for i := 0; i < len(sr.ConsensusGroup()); i++ {
		_ = sr.SetJobDone(sr.ConsensusGroup()[i], bls.SrBlock, false)
		_ = sr.SetJobDone(sr.ConsensusGroup()[i], bls.SrSignature, false)
	}

	ok, n := sr.AreSignaturesCollected(2)
	assert.False(t, ok)
	assert.Equal(t, 0, n)

	ok, _ = sr.AreSignaturesCollected(2)
	assert.False(t, ok)

	_ = sr.SetJobDone("B", bls.SrSignature, true)
	isJobDone, _ := sr.JobDone("B", bls.SrSignature)
	assert.True(t, isJobDone)

	ok, _ = sr.AreSignaturesCollected(2)
	assert.False(t, ok)

	_ = sr.SetJobDone("C", bls.SrSignature, true)
	ok, _ = sr.AreSignaturesCollected(2)
	assert.True(t, ok)
}

func TestSubroundSignature_DoSignatureConsensusCheckShouldReturnFalseWhenRoundIsCanceled(t *testing.T) {
	t.Parallel()

	sr := initSubroundSignature()
	sr.SetRoundCanceled(true)
	assert.False(t, sr.DoSignatureConsensusCheck())
}

func TestSubroundSignature_DoSignatureConsensusCheckShouldReturnTrueWhenSubroundIsFinished(t *testing.T) {
	t.Parallel()

	sr := initSubroundSignature()
	sr.SetStatus(bls.SrSignature, spos.SsFinished)
	assert.True(t, sr.DoSignatureConsensusCheck())
}

func TestSubroundSignature_DoSignatureConsensusCheckShouldReturnTrueWhenSignaturesCollectedReturnTrue(t *testing.T) {
	t.Parallel()

	sr := initSubroundSignature()

	for i := 0; i < sr.Threshold(bls.SrSignature); i++ {
		_ = sr.SetJobDone(sr.ConsensusGroup()[i], bls.SrSignature, true)
	}

	sr.SetHeader(&block.HeaderV2{})
	assert.True(t, sr.DoSignatureConsensusCheck())
}

func TestSubroundSignature_DoSignatureConsensusCheckShouldReturnFalseWhenSignaturesCollectedReturnFalse(t *testing.T) {
	t.Parallel()

	sr := initSubroundSignature()
	sr.SetHeader(&block.HeaderV2{Header: createDefaultHeader()})
	assert.False(t, sr.DoSignatureConsensusCheck())
}

func TestSubroundSignature_DoSignatureConsensusCheckNotAllSignaturesCollectedAndTimeIsNotOut(t *testing.T) {
	t.Parallel()

	t.Run("with flag active, should return false - will be done on subroundEndRound", testSubroundSignatureDoSignatureConsensusCheck(argTestSubroundSignatureDoSignatureConsensusCheck{
		flagActive:                  true,
		jobsDone:                    setThresholdJobsDone,
		waitingAllSignaturesTimeOut: false,
		expectedResult:              false,
	}))
	t.Run("with flag inactive, should return false when not all signatures are collected and time is not out", testSubroundSignatureDoSignatureConsensusCheck(argTestSubroundSignatureDoSignatureConsensusCheck{
		flagActive:                  false,
		jobsDone:                    setThresholdJobsDone,
		waitingAllSignaturesTimeOut: false,
		expectedResult:              false,
	}))
}

func TestSubroundSignature_DoSignatureConsensusCheckAllSignaturesCollected(t *testing.T) {
	t.Parallel()
	t.Run("with flag active, should return false - will be done on subroundEndRound", testSubroundSignatureDoSignatureConsensusCheck(argTestSubroundSignatureDoSignatureConsensusCheck{
		flagActive:                  true,
		jobsDone:                    "all",
		waitingAllSignaturesTimeOut: false,
		expectedResult:              false,
	}))
	t.Run("with flag inactive, should return true when all signatures are collected", testSubroundSignatureDoSignatureConsensusCheck(argTestSubroundSignatureDoSignatureConsensusCheck{
		flagActive:                  false,
		jobsDone:                    "all",
		waitingAllSignaturesTimeOut: false,
		expectedResult:              true,
	}))
}

func TestSubroundSignature_DoSignatureConsensusCheckEnoughButNotAllSignaturesCollectedAndTimeIsOut(t *testing.T) {
	t.Parallel()

	t.Run("with flag active, should return false - will be done on subroundEndRound", testSubroundSignatureDoSignatureConsensusCheck(argTestSubroundSignatureDoSignatureConsensusCheck{
		flagActive:                  true,
		jobsDone:                    setThresholdJobsDone,
		waitingAllSignaturesTimeOut: true,
		expectedResult:              false,
	}))
	t.Run("with flag inactive, should return true when enough but not all signatures collected and time is out", testSubroundSignatureDoSignatureConsensusCheck(argTestSubroundSignatureDoSignatureConsensusCheck{
		flagActive:                  false,
		jobsDone:                    setThresholdJobsDone,
		waitingAllSignaturesTimeOut: true,
		expectedResult:              true,
	}))
}

type argTestSubroundSignatureDoSignatureConsensusCheck struct {
	flagActive                  bool
	jobsDone                    string
	waitingAllSignaturesTimeOut bool
	expectedResult              bool
}

func testSubroundSignatureDoSignatureConsensusCheck(args argTestSubroundSignatureDoSignatureConsensusCheck) func(t *testing.T) {
	return func(t *testing.T) {
		t.Parallel()

		container := consensusMocks.InitConsensusCore()
		container.SetEnableEpochsHandler(&enableEpochsHandlerMock.EnableEpochsHandlerStub{
			IsFlagEnabledInEpochCalled: func(flag core.EnableEpochFlag, epoch uint32) bool {
				if flag == common.EquivalentMessagesFlag {
					return args.flagActive
				}
				return false
			},
		})
		sr := initSubroundSignatureWithContainer(container)
		sr.SetWaitingAllSignaturesTimeOut(args.waitingAllSignaturesTimeOut)

		if !args.flagActive {
			leader, err := sr.GetLeader()
			assert.Nil(t, err)
			sr.SetSelfPubKey(leader)
		}

		numberOfJobsDone := sr.ConsensusGroupSize()
		if args.jobsDone == setThresholdJobsDone {
			numberOfJobsDone = sr.Threshold(bls.SrSignature)
		}
		for i := 0; i < numberOfJobsDone; i++ {
			_ = sr.SetJobDone(sr.ConsensusGroup()[i], bls.SrSignature, true)
		}

		sr.SetHeader(&block.HeaderV2{})
		assert.Equal(t, args.expectedResult, sr.DoSignatureConsensusCheck())
	}
}

func TestSubroundSignature_DoSignatureConsensusCheckShouldReturnFalseWhenFallbackThresholdCouldNotBeApplied(t *testing.T) {
	t.Parallel()

	container := consensusMocks.InitConsensusCore()
	container.SetFallbackHeaderValidator(&testscommon.FallBackHeaderValidatorStub{
		ShouldApplyFallbackValidationCalled: func(headerHandler data.HeaderHandler) bool {
			return false
		},
	})
	sr := initSubroundSignatureWithContainer(container)
	sr.SetWaitingAllSignaturesTimeOut(false)

	leader, err := sr.GetLeader()
	assert.Nil(t, err)
	sr.SetSelfPubKey(leader)

	for i := 0; i < sr.FallbackThreshold(bls.SrSignature); i++ {
		_ = sr.SetJobDone(sr.ConsensusGroup()[i], bls.SrSignature, true)
	}

	assert.False(t, sr.DoSignatureConsensusCheck())
}

func TestSubroundSignature_DoSignatureConsensusCheckShouldReturnTrueWhenFallbackThresholdCouldBeApplied(t *testing.T) {
	t.Parallel()

	container := consensusMocks.InitConsensusCore()
	container.SetFallbackHeaderValidator(&testscommon.FallBackHeaderValidatorStub{
		ShouldApplyFallbackValidationCalled: func(headerHandler data.HeaderHandler) bool {
			return true
		},
	})
	sr := initSubroundSignatureWithContainer(container)
	sr.SetWaitingAllSignaturesTimeOut(true)

	leader, err := sr.GetLeader()
	assert.Nil(t, err)
	sr.SetSelfPubKey(leader)

	for i := 0; i < sr.FallbackThreshold(bls.SrSignature); i++ {
		_ = sr.SetJobDone(sr.ConsensusGroup()[i], bls.SrSignature, true)
	}

	sr.SetHeader(&block.HeaderV2{})
	assert.True(t, sr.DoSignatureConsensusCheck())
}

func TestSubroundSignature_ReceivedSignatureReturnFalseWhenConsensusDataIsNotEqual(t *testing.T) {
	t.Parallel()

	sr := initSubroundSignature()

	leader, err := sr.GetLeader()
	require.Nil(t, err)

	cnsMsg := consensus.NewConsensusMessage(
		append(sr.GetData(), []byte("X")...),
		[]byte("signature"),
		nil,
		nil,
		[]byte(leader),
		[]byte("sig"),
		int(bls.MtSignature),
		0,
		chainID,
		nil,
		nil,
		nil,
		currentPid,
		nil,
	)

	sr.SetHeader(&block.HeaderV2{})
	assert.False(t, sr.ReceivedSignature(cnsMsg))
}