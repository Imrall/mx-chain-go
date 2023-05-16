package incomingHeader

import (
	"encoding/hex"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/check"
	"github.com/multiversx/mx-chain-core-go/data"
	"github.com/multiversx/mx-chain-core-go/data/block"
	"github.com/multiversx/mx-chain-core-go/data/sovereign"
	"github.com/multiversx/mx-chain-core-go/hashing"
	"github.com/multiversx/mx-chain-core-go/marshal"
	logger "github.com/multiversx/mx-chain-logger-go"
)

var log = logger.GetOrCreate("headerSubscriber")

// ArgsIncomingHeaderProcessor is a struct placeholder for args needed to create a new incoming header processor
type ArgsIncomingHeaderProcessor struct {
	HeadersPool HeadersPool
	TxPool      TransactionPool
	Marshaller  marshal.Marshalizer
	Hasher      hashing.Hasher
}

type incomingHeaderProcessor struct {
	headersPool HeadersPool
	txPool      TransactionPool
	marshaller  marshal.Marshalizer
	hasher      hashing.Hasher
}

// NewIncomingHeaderProcessor creates an incoming header processor which should be able to receive incoming headers and events
// from a chain to local sovereign chain. This handler will validate the events(using proofs in the future) and create
// incoming miniblocks and transaction(which will be added in pool) to be executed in sovereign shard.
func NewIncomingHeaderProcessor(args ArgsIncomingHeaderProcessor) (*incomingHeaderProcessor, error) {
	if check.IfNil(args.HeadersPool) {
		return nil, errNilHeadersPool
	}
	if check.IfNil(args.TxPool) {
		return nil, errNilTxPool
	}
	if check.IfNil(args.Marshaller) {
		return nil, core.ErrNilMarshalizer
	}
	if check.IfNil(args.Hasher) {
		return nil, core.ErrNilHasher
	}

	return &incomingHeaderProcessor{
		headersPool: args.HeadersPool,
		txPool:      args.TxPool,
		marshaller:  args.Marshaller,
		hasher:      args.Hasher,
	}, nil
}

// AddHeader will receive the incoming header, validate it, create incoming mbs and transactions and add them to pool
func (ihp *incomingHeaderProcessor) AddHeader(headerHash []byte, header sovereign.IncomingHeaderHandler) error {
	log.Info("received incoming header", "hash", hex.EncodeToString(headerHash))

	headerV2, castOk := header.GetHeaderHandler().(*block.HeaderV2)
	if !castOk {
		return errInvalidHeaderType
	}

	incomingSCRs, err := ihp.createIncomingSCRs(header.GetIncomingEventHandlers())
	if err != nil {
		return err
	}

	incomingMB := createIncomingMb(incomingSCRs)
	extendedHeader := &block.ShardHeaderExtended{
		Header:             headerV2,
		IncomingMiniBlocks: []*block.MiniBlock{incomingMB},
	}

	err = ihp.addExtendedHeaderToPool(extendedHeader)
	if err != nil {
		return err
	}

	ihp.addSCRsToPool(incomingSCRs)
	return nil
}

// TODO: Implement this in task MX-14129
func createIncomingMb(_ []*scrInfo) *block.MiniBlock {
	return &block.MiniBlock{}
}

func (ihp *incomingHeaderProcessor) addExtendedHeaderToPool(extendedHeader data.ShardHeaderExtendedHandler) error {
	extendedHeaderHash, err := core.CalculateHash(ihp.marshaller, ihp.hasher, extendedHeader)
	if err != nil {
		return err
	}

	ihp.headersPool.AddHeader(extendedHeaderHash, extendedHeader)
	return nil
}

// IsInterfaceNil checks if the underlying pointer is nil
func (ihp *incomingHeaderProcessor) IsInterfaceNil() bool {
	return ihp == nil
}
