package detector

import (
	"github.com/ElrondNetwork/elrond-go-core/data"
)

// RoundDetectorCache defines what a round-based(per validator data) cache should do.
type RoundDetectorCache interface {
	// Add should add in cache an intercepted data for a public key, in a given round.
	// If the public key has any data checked in the given round OR the round is
	// irrelevant(obsolete) to be cached, an error should be returned.
	// If the cache is full, it should have an eviction mechanism to always remove
	// the oldest round entry
	Add(round uint64, pubKey []byte, header data.HeaderInfoHandler) error
	// GetHeaders returns all cached headers for a public key, in a given round
	GetHeaders(round uint64, pubKey []byte) []data.HeaderHandler
	// GetPubKeys returns all cached public keys in a given round
	GetPubKeys(round uint64) [][]byte
	// IsInterfaceNil checks if the interface is nil
	IsInterfaceNil() bool
}

// HeadersCache defines what a header-hash-based cache should do
type HeadersCache interface {
	// Add should add in cache a header, along with its hash.
	// If the hash is already cached in the given round OR the
	// round is irrelevant(obsolete) to be cached, an error should be returned.
	// If the cache is full, it should have an eviction mechanism
	// to always remove the oldest round entry
	Add(round uint64, header data.HeaderInfoHandler) error
	// IsInterfaceNil checks if the interface is nil
	IsInterfaceNil() bool
}
