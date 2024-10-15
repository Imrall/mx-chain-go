package consensus

import (
	"github.com/multiversx/mx-chain-core-go/data"
	"github.com/multiversx/mx-chain-core-go/hashing"
	"github.com/multiversx/mx-chain-core-go/marshal"

	"github.com/multiversx/mx-chain-go/common"
	cryptoCommon "github.com/multiversx/mx-chain-go/common/crypto"
	"github.com/multiversx/mx-chain-go/consensus"
	"github.com/multiversx/mx-chain-go/epochStart"
	"github.com/multiversx/mx-chain-go/ntp"
	"github.com/multiversx/mx-chain-go/process"
	"github.com/multiversx/mx-chain-go/sharding"
	"github.com/multiversx/mx-chain-go/sharding/nodesCoordinator"
)

// TODO: remove this mock component; implement setters for main component in export_test.go
// ConsensusCoreMock -
type ConsensusCoreMock struct {
	blockChain              data.ChainHandler
	blockProcessor          process.BlockProcessor
	headersSubscriber       consensus.HeadersPoolSubscriber
	bootstrapper            process.Bootstrapper
	broadcastMessenger      consensus.BroadcastMessenger
	chronologyHandler       consensus.ChronologyHandler
	hasher                  hashing.Hasher
	marshalizer             marshal.Marshalizer
	multiSignerContainer    cryptoCommon.MultiSignerContainer
	roundHandler            consensus.RoundHandler
	shardCoordinator        sharding.Coordinator
	syncTimer               ntp.SyncTimer
	validatorGroupSelector  nodesCoordinator.NodesCoordinator
	epochStartNotifier      epochStart.RegistrationHandler
	antifloodHandler        consensus.P2PAntifloodHandler
	peerHonestyHandler      consensus.PeerHonestyHandler
	headerSigVerifier       consensus.HeaderSigVerifier
	fallbackHeaderValidator consensus.FallbackHeaderValidator
	nodeRedundancyHandler   consensus.NodeRedundancyHandler
	scheduledProcessor      consensus.ScheduledProcessor
	messageSigningHandler   consensus.P2PSigningHandler
	peerBlacklistHandler    consensus.PeerBlacklistHandler
	signingHandler          consensus.SigningHandler
	enableEpochsHandler     common.EnableEpochsHandler
	equivalentProofsPool    consensus.EquivalentProofsPool
}

// GetAntiFloodHandler -
func (ccm *ConsensusCoreMock) GetAntiFloodHandler() consensus.P2PAntifloodHandler {
	return ccm.antifloodHandler
}

// Blockchain -
func (ccm *ConsensusCoreMock) Blockchain() data.ChainHandler {
	return ccm.blockChain
}

// BlockProcessor -
func (ccm *ConsensusCoreMock) BlockProcessor() process.BlockProcessor {
	return ccm.blockProcessor
}

// HeadersPoolSubscriber -
func (ccm *ConsensusCoreMock) HeadersPoolSubscriber() consensus.HeadersPoolSubscriber {
	return ccm.headersSubscriber
}

// BootStrapper -
func (ccm *ConsensusCoreMock) BootStrapper() process.Bootstrapper {
	return ccm.bootstrapper
}

// BroadcastMessenger -
func (ccm *ConsensusCoreMock) BroadcastMessenger() consensus.BroadcastMessenger {
	return ccm.broadcastMessenger
}

// Chronology -
func (ccm *ConsensusCoreMock) Chronology() consensus.ChronologyHandler {
	return ccm.chronologyHandler
}

// Hasher -
func (ccm *ConsensusCoreMock) Hasher() hashing.Hasher {
	return ccm.hasher
}

// Marshalizer -
func (ccm *ConsensusCoreMock) Marshalizer() marshal.Marshalizer {
	return ccm.marshalizer
}

// MultiSignerContainer -
func (ccm *ConsensusCoreMock) MultiSignerContainer() cryptoCommon.MultiSignerContainer {
	return ccm.multiSignerContainer
}

// RoundHandler -
func (ccm *ConsensusCoreMock) RoundHandler() consensus.RoundHandler {
	return ccm.roundHandler
}

// ShardCoordinator -
func (ccm *ConsensusCoreMock) ShardCoordinator() sharding.Coordinator {
	return ccm.shardCoordinator
}

// SyncTimer -
func (ccm *ConsensusCoreMock) SyncTimer() ntp.SyncTimer {
	return ccm.syncTimer
}

// NodesCoordinator -
func (ccm *ConsensusCoreMock) NodesCoordinator() nodesCoordinator.NodesCoordinator {
	return ccm.validatorGroupSelector
}

// EpochStartRegistrationHandler -
func (ccm *ConsensusCoreMock) EpochStartRegistrationHandler() epochStart.RegistrationHandler {
	return ccm.epochStartNotifier
}

// SetBlockchain -
func (ccm *ConsensusCoreMock) SetBlockchain(blockChain data.ChainHandler) {
	ccm.blockChain = blockChain
}

// SetHeaderSubscriber -
func (ccm *ConsensusCoreMock) SetHeaderSubscriber(headersSubscriber consensus.HeadersPoolSubscriber) {
	ccm.headersSubscriber = headersSubscriber
}

// SetBlockProcessor -
func (ccm *ConsensusCoreMock) SetBlockProcessor(blockProcessor process.BlockProcessor) {
	ccm.blockProcessor = blockProcessor
}

// SetBootStrapper -
func (ccm *ConsensusCoreMock) SetBootStrapper(bootstrapper process.Bootstrapper) {
	ccm.bootstrapper = bootstrapper
}

// SetBroadcastMessenger -
func (ccm *ConsensusCoreMock) SetBroadcastMessenger(broadcastMessenger consensus.BroadcastMessenger) {
	ccm.broadcastMessenger = broadcastMessenger
}

// SetChronology -
func (ccm *ConsensusCoreMock) SetChronology(chronologyHandler consensus.ChronologyHandler) {
	ccm.chronologyHandler = chronologyHandler
}

// SetHasher -
func (ccm *ConsensusCoreMock) SetHasher(hasher hashing.Hasher) {
	ccm.hasher = hasher
}

// SetMarshalizer -
func (ccm *ConsensusCoreMock) SetMarshalizer(marshalizer marshal.Marshalizer) {
	ccm.marshalizer = marshalizer
}

// SetMultiSignerContainer -
func (ccm *ConsensusCoreMock) SetMultiSignerContainer(multiSignerContainer cryptoCommon.MultiSignerContainer) {
	ccm.multiSignerContainer = multiSignerContainer
}

// SetRoundHandler -
func (ccm *ConsensusCoreMock) SetRoundHandler(roundHandler consensus.RoundHandler) {
	ccm.roundHandler = roundHandler
}

// SetShardCoordinator -
func (ccm *ConsensusCoreMock) SetShardCoordinator(shardCoordinator sharding.Coordinator) {
	ccm.shardCoordinator = shardCoordinator
}

// SetSyncTimer -
func (ccm *ConsensusCoreMock) SetSyncTimer(syncTimer ntp.SyncTimer) {
	ccm.syncTimer = syncTimer
}

// SetValidatorGroupSelector -
func (ccm *ConsensusCoreMock) SetValidatorGroupSelector(validatorGroupSelector nodesCoordinator.NodesCoordinator) {
	ccm.validatorGroupSelector = validatorGroupSelector
}

// SetEpochStartNotifier -
func (ccm *ConsensusCoreMock) SetEpochStartNotifier(epochStartNotifier epochStart.RegistrationHandler) {
	ccm.epochStartNotifier = epochStartNotifier
}

// SetAntifloodHandler -
func (ccm *ConsensusCoreMock) SetAntifloodHandler(antifloodHandler consensus.P2PAntifloodHandler) {
	ccm.antifloodHandler = antifloodHandler
}

// SetPeerHonestyHandler -
func (ccm *ConsensusCoreMock) SetPeerHonestyHandler(peerHonestyHandler consensus.PeerHonestyHandler) {
	ccm.peerHonestyHandler = peerHonestyHandler
}

// SetScheduledProcessor -
func (ccm *ConsensusCoreMock) SetScheduledProcessor(scheduledProcessor consensus.ScheduledProcessor) {
	ccm.scheduledProcessor = scheduledProcessor
}

// SetPeerBlacklistHandler -
func (ccm *ConsensusCoreMock) SetPeerBlacklistHandler(peerBlacklistHandler consensus.PeerBlacklistHandler) {
	ccm.peerBlacklistHandler = peerBlacklistHandler
}

// PeerHonestyHandler -
func (ccm *ConsensusCoreMock) PeerHonestyHandler() consensus.PeerHonestyHandler {
	return ccm.peerHonestyHandler
}

// HeaderSigVerifier -
func (ccm *ConsensusCoreMock) HeaderSigVerifier() consensus.HeaderSigVerifier {
	return ccm.headerSigVerifier
}

// SetHeaderSigVerifier -
func (ccm *ConsensusCoreMock) SetHeaderSigVerifier(headerSigVerifier consensus.HeaderSigVerifier) {
	ccm.headerSigVerifier = headerSigVerifier
}

// FallbackHeaderValidator -
func (ccm *ConsensusCoreMock) FallbackHeaderValidator() consensus.FallbackHeaderValidator {
	return ccm.fallbackHeaderValidator
}

// SetFallbackHeaderValidator -
func (ccm *ConsensusCoreMock) SetFallbackHeaderValidator(fallbackHeaderValidator consensus.FallbackHeaderValidator) {
	ccm.fallbackHeaderValidator = fallbackHeaderValidator
}

// NodeRedundancyHandler -
func (ccm *ConsensusCoreMock) NodeRedundancyHandler() consensus.NodeRedundancyHandler {
	return ccm.nodeRedundancyHandler
}

// ScheduledProcessor -
func (ccm *ConsensusCoreMock) ScheduledProcessor() consensus.ScheduledProcessor {
	return ccm.scheduledProcessor
}

// SetNodeRedundancyHandler -
func (ccm *ConsensusCoreMock) SetNodeRedundancyHandler(nodeRedundancyHandler consensus.NodeRedundancyHandler) {
	ccm.nodeRedundancyHandler = nodeRedundancyHandler
}

// MessageSigningHandler -
func (ccm *ConsensusCoreMock) MessageSigningHandler() consensus.P2PSigningHandler {
	return ccm.messageSigningHandler
}

// SetMessageSigningHandler -
func (ccm *ConsensusCoreMock) SetMessageSigningHandler(messageSigningHandler consensus.P2PSigningHandler) {
	ccm.messageSigningHandler = messageSigningHandler
}

// PeerBlacklistHandler will return the peer blacklist handler
func (ccm *ConsensusCoreMock) PeerBlacklistHandler() consensus.PeerBlacklistHandler {
	return ccm.peerBlacklistHandler
}

// SigningHandler -
func (ccm *ConsensusCoreMock) SigningHandler() consensus.SigningHandler {
	return ccm.signingHandler
}

// SetSigningHandler -
func (ccm *ConsensusCoreMock) SetSigningHandler(signingHandler consensus.SigningHandler) {
	ccm.signingHandler = signingHandler
}

// EnableEpochsHandler -
func (ccm *ConsensusCoreMock) EnableEpochsHandler() common.EnableEpochsHandler {
	return ccm.enableEpochsHandler
}

// SetEnableEpochsHandler -
func (ccm *ConsensusCoreMock) SetEnableEpochsHandler(enableEpochsHandler common.EnableEpochsHandler) {
	ccm.enableEpochsHandler = enableEpochsHandler
}

// EquivalentProofsPool -
func (ccm *ConsensusCoreMock) EquivalentProofsPool() consensus.EquivalentProofsPool {
	return ccm.equivalentProofsPool
}

// SetEquivalentProofsPool -
func (ccm *ConsensusCoreMock) SetEquivalentProofsPool(proofPool consensus.EquivalentProofsPool) {
	ccm.equivalentProofsPool = proofPool
}

// IsInterfaceNil returns true if there is no value under the interface
func (ccm *ConsensusCoreMock) IsInterfaceNil() bool {
	return ccm == nil
}