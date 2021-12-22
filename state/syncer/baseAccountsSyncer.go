package syncer

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-go-core/hashing"
	"github.com/ElrondNetwork/elrond-go-core/marshal"
	"github.com/ElrondNetwork/elrond-go/common"
	"github.com/ElrondNetwork/elrond-go/state"
	"github.com/ElrondNetwork/elrond-go/storage"
	"github.com/ElrondNetwork/elrond-go/trie"
	"github.com/ElrondNetwork/elrond-go/update"
)

type baseAccountsSyncer struct {
	hasher                    hashing.Hasher
	marshalizer               marshal.Marshalizer
	dataTries                 map[string]struct{}
	mutex                     sync.Mutex
	trieStorageManager        common.StorageManager
	requestHandler            trie.RequestHandler
	timeoutHandler            trie.TimeoutHandler
	shardId                   uint32
	cacher                    storage.Cacher
	rootHash                  []byte
	maxTrieLevelInMemory      uint
	name                      string
	maxHardCapForMissingNodes int
	trieSyncerVersion         int
	numTriesSynced            int32
	numMaxTries               int32
	trieExporter              update.TrieExporter
}

const timeBetweenStatisticsPrints = time.Second * 2

// ArgsNewBaseAccountsSyncer defines the arguments needed for the new account syncer
type ArgsNewBaseAccountsSyncer struct {
	Hasher                    hashing.Hasher
	Marshalizer               marshal.Marshalizer
	TrieStorageManager        common.StorageManager
	RequestHandler            trie.RequestHandler
	Timeout                   time.Duration
	Cacher                    storage.Cacher
	MaxTrieLevelInMemory      uint
	MaxHardCapForMissingNodes int
	TrieSyncerVersion         int
	TrieExporter              update.TrieExporter
}

func checkArgs(args ArgsNewBaseAccountsSyncer) error {
	if check.IfNil(args.Hasher) {
		return state.ErrNilHasher
	}
	if check.IfNil(args.Marshalizer) {
		return state.ErrNilMarshalizer
	}
	if check.IfNil(args.TrieStorageManager) {
		return state.ErrNilStorageManager
	}
	if check.IfNil(args.RequestHandler) {
		return state.ErrNilRequestHandler
	}
	if check.IfNil(args.Cacher) {
		return state.ErrNilCacher
	}
	if args.MaxHardCapForMissingNodes < 1 {
		return state.ErrInvalidMaxHardCapForMissingNodes
	}
	if check.IfNil(args.TrieExporter) {
		return state.ErrNilTrieExporter
	}

	return trie.CheckTrieSyncerVersion(args.TrieSyncerVersion)
}

func (b *baseAccountsSyncer) syncMainTrie(
	rootHash []byte,
	trieTopic string,
	ssh common.SizeSyncStatisticsHandler,
	ctx context.Context,
) (common.Trie, error) {
	b.rootHash = rootHash
	atomic.AddInt32(&b.numMaxTries, 1)

	log.Trace("syncing main trie", "roothash", rootHash)
	dataTrie, err := trie.NewTrie(b.trieStorageManager, b.marshalizer, b.hasher, b.maxTrieLevelInMemory)
	if err != nil {
		return nil, err
	}

	b.dataTries[string(rootHash)] = struct{}{}
	arg := trie.ArgTrieSyncer{
		RequestHandler:            b.requestHandler,
		InterceptedNodes:          b.cacher,
		DB:                        b.trieStorageManager,
		Marshalizer:               b.marshalizer,
		Hasher:                    b.hasher,
		ShardId:                   b.shardId,
		Topic:                     trieTopic,
		TrieSyncStatistics:        ssh,
		TimeoutHandler:            b.timeoutHandler,
		MaxHardCapForMissingNodes: b.maxHardCapForMissingNodes,
	}
	trieSyncer, err := trie.CreateTrieSyncer(arg, b.trieSyncerVersion)
	if err != nil {
		return nil, err
	}

	err = trieSyncer.StartSyncing(rootHash, ctx)
	if err != nil {
		return nil, err
	}

	atomic.AddInt32(&b.numTriesSynced, 1)

	log.Trace("finished syncing main trie", "roothash", rootHash)

	return dataTrie.Recreate(rootHash)
}

func (b *baseAccountsSyncer) printStatistics(ssh common.SizeSyncStatisticsHandler, ctx context.Context) {
	lastDataReceived := uint64(0)
	peakDataReceived := uint64(0)
	startedSync := time.Now()
	for {
		select {
		case <-ctx.Done():
			peakSpeed := convertBytesPerIntervalToSpeed(peakDataReceived, timeBetweenStatisticsPrints)
			finishedSync := time.Now()
			totalSyncDuration := finishedSync.Sub(startedSync)
			averageSpeed := convertBytesPerIntervalToSpeed(ssh.NumBytesReceived(), totalSyncDuration)

			log.Info("finished trie sync",
				"name", b.name,
				"time elapsed", totalSyncDuration.Truncate(time.Second),
				"num processed", ssh.NumReceived(),
				"num large nodes", ssh.NumLarge(),
				"num missing", ssh.NumMissing(),
				"state data size", core.ConvertBytes(ssh.NumBytesReceived()),
				"peak network speed", peakSpeed,
				"average network speed", averageSpeed,
			)
			return
		case <-time.After(timeBetweenStatisticsPrints):
			bytesReceivedDelta := ssh.NumBytesReceived() - lastDataReceived
			if ssh.NumBytesReceived() < lastDataReceived {
				bytesReceivedDelta = 0
			}
			lastDataReceived = ssh.NumBytesReceived()

			speed := convertBytesPerIntervalToSpeed(bytesReceivedDelta, timeBetweenStatisticsPrints)
			if peakDataReceived < bytesReceivedDelta {
				peakDataReceived = bytesReceivedDelta
			}

			log.Info("trie sync in progress",
				"name", b.name,
				"time elapsed", time.Since(startedSync).Truncate(time.Second),
				"num tries currently syncing", ssh.NumTries(),
				"num processed", ssh.NumReceived(),
				"num large nodes", ssh.NumLarge(),
				"num missing", ssh.NumMissing(),
				"num tries synced", fmt.Sprintf("%d/%d", atomic.LoadInt32(&b.numTriesSynced), atomic.LoadInt32(&b.numMaxTries)),
				"intercepted trie nodes cache size", core.ConvertBytes(b.cacher.SizeInBytesContained()),
				"num of intercepted trie nodes", b.cacher.Len(),
				"state data size", core.ConvertBytes(ssh.NumBytesReceived()),
				"network speed", speed)
		}
	}
}

func convertBytesPerIntervalToSpeed(bytes uint64, interval time.Duration) string {
	if interval < time.Millisecond {
		// con not compute precisely, highly likely to get an overflow
		return "N/A"
	}

	bytesReceivedPerSec := float64(bytes) / interval.Seconds()
	uint64Val := uint64(bytesReceivedPerSec)

	return fmt.Sprintf("%s/s", core.ConvertBytes(uint64Val))
}

// GetTrieExporter returns the trie exporter
func (b *baseAccountsSyncer) GetTrieExporter() update.TrieExporter {
	return b.trieExporter
}

// IsInterfaceNil returns true if underlying object is nil
func (b *baseAccountsSyncer) IsInterfaceNil() bool {
	return b == nil
}
