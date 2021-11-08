package detector

import (
	"bytes"
	"math"
	"sync"

	"github.com/ElrondNetwork/elrond-go-core/data"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/slash"
)

type validatorHeadersMap map[string]slash.HeaderInfoList

type roundValidatorsHeadersCache struct {
	cache       map[uint64]validatorHeadersMap
	cacheMutex  sync.RWMutex
	oldestRound uint64
	cacheSize   uint64
}

// NewRoundValidatorHeaderCache creates a new instance of roundValidatorsHeadersCache, which
// is a round-based(per validator data) cache
func NewRoundValidatorHeaderCache(maxRounds uint64) *roundValidatorsHeadersCache {
	return &roundValidatorsHeadersCache{
		cache:       make(map[uint64]validatorHeadersMap),
		cacheMutex:  sync.RWMutex{},
		oldestRound: math.MaxUint64,
		cacheSize:   maxRounds,
	}
}

// Add adds in cache an intercepted data for a public key, in a given round.
// It has an eviction mechanism which always removes the oldest round entry when cache is full
func (rdc *roundValidatorsHeadersCache) Add(round uint64, pubKey []byte, headerInfo data.HeaderInfoHandler) error {
	pubKeyStr := string(pubKey)
	rdc.cacheMutex.Lock()
	defer rdc.cacheMutex.Unlock()

	if rdc.contains(round, pubKey, headerInfo.GetHash()) {
		return process.ErrHeadersNotDifferentHashes
	}

	if rdc.isCacheFull(round) {
		if round < rdc.oldestRound {
			return process.ErrHeaderRoundNotRelevant
		}
		rdc.updateOldestRound()
	}
	if round < rdc.oldestRound {
		rdc.oldestRound = round
	}

	validatorsMap, exists := rdc.cache[round]
	if !exists {
		rdc.cache[round] = validatorHeadersMap{pubKeyStr: slash.HeaderInfoList{headerInfo}}
		return nil
	}

	validatorsMap[pubKeyStr] = append(validatorsMap[pubKeyStr], headerInfo)

	return nil
}

func (rdc *roundValidatorsHeadersCache) contains(round uint64, pubKey []byte, hash []byte) bool {
	validatorsMap, exists := rdc.cache[round]
	if !exists {
		return false
	}

	headers, exist := validatorsMap[string(pubKey)]
	if !exist {
		return false
	}

	for _, currHeader := range headers {
		if bytes.Equal(currHeader.GetHash(), hash) {
			return true
		}
	}

	return false
}

func (rdc *roundValidatorsHeadersCache) isCacheFull(currRound uint64) bool {
	_, isCurrRoundInCache := rdc.cache[currRound]
	return len(rdc.cache) >= int(rdc.cacheSize) && !isCurrRoundInCache
}

func (rdc *roundValidatorsHeadersCache) updateOldestRound() {
	delete(rdc.cache, rdc.oldestRound)
	min := uint64(math.MaxUint64)

	for round := range rdc.cache {
		if round < min {
			min = round
		}
	}

	rdc.oldestRound = min
}

// GetHeaders returns all cached data for a public key, in a given round
func (rdc *roundValidatorsHeadersCache) GetHeaders(round uint64, pubKey []byte) []data.HeaderHandler {
	pubKeyStr := string(pubKey)
	rdc.cacheMutex.RLock()
	defer rdc.cacheMutex.RUnlock()

	validatorData, exists := rdc.cache[round]
	if !exists {
		return nil
	}
	headersInfo, exists := validatorData[pubKeyStr]
	if !exists {
		return nil
	}

	headers := make([]data.HeaderHandler, 0, len(headersInfo))
	for _, headerInfo := range headersInfo {
		headers = append(headers, headerInfo.GetHeaderHandler())
	}

	return headers
}

// GetPubKeys returns all cached public keys in a given round
func (rdc *roundValidatorsHeadersCache) GetPubKeys(round uint64) [][]byte {
	ret := make([][]byte, 0)
	rdc.cacheMutex.RLock()
	defer rdc.cacheMutex.RUnlock()

	if _, exists := rdc.cache[round]; exists {
		for pubKey := range rdc.cache[round] {
			ret = append(ret, []byte(pubKey))
		}
	}

	return ret
}

// IsInterfaceNil checks if the underlying pointer is nil
func (rdc *roundValidatorsHeadersCache) IsInterfaceNil() bool {
	return rdc == nil
}
