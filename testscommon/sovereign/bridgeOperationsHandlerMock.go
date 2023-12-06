package sovereign

import (
	"context"

	"github.com/multiversx/mx-chain-core-go/data/sovereign"
)

// BridgeOperationsHandlerMock -
type BridgeOperationsHandlerMock struct {
	SendCalled func(ctx context.Context, data *sovereign.BridgeOperations) error
}

// Send -
func (mock *BridgeOperationsHandlerMock) Send(ctx context.Context, data *sovereign.BridgeOperations) (*sovereign.BridgeOperationsResponse, error) {
	if mock.SendCalled != nil {
		return nil, mock.SendCalled(ctx, data)
	}

	return nil, nil
}

// IsInterfaceNil -
func (mock *BridgeOperationsHandlerMock) IsInterfaceNil() bool {
	return mock == nil
}
