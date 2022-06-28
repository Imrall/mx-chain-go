package stakingcommon

import (
	"github.com/ElrondNetwork/elrond-go/common"
	"github.com/ElrondNetwork/elrond-go/state"
)

// ValidatorsProviderStub -
type ValidatorsProviderStub struct {
	GetLatestValidatorsCalled func() map[string]*state.ValidatorApiResponse
	GetAuctionListCalled      func() ([]*common.AuctionListValidatorAPIResponse, error)
}

// GetLatestValidators -
func (vp *ValidatorsProviderStub) GetLatestValidators() map[string]*state.ValidatorApiResponse {
	if vp.GetLatestValidatorsCalled != nil {
		return vp.GetLatestValidatorsCalled()
	}

	return nil
}

// GetAuctionList -
func (vp *ValidatorsProviderStub) GetAuctionList() ([]*common.AuctionListValidatorAPIResponse, error) {
	if vp.GetAuctionListCalled != nil {
		return vp.GetAuctionListCalled()
	}

	return nil, nil
}

// Close -
func (vp *ValidatorsProviderStub) Close() error {
	return nil
}

// IsInterfaceNil -
func (vp *ValidatorsProviderStub) IsInterfaceNil() bool {
	return vp == nil
}
