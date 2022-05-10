package groups_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	apiErrors "github.com/ElrondNetwork/elrond-go/api/errors"
	"github.com/ElrondNetwork/elrond-go/api/groups"
	"github.com/ElrondNetwork/elrond-go/api/mock"
	"github.com/ElrondNetwork/elrond-go/api/shared"
	"github.com/ElrondNetwork/elrond-go/common"
	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewValidatorGroup(t *testing.T) {
	t.Parallel()

	t.Run("nil facade", func(t *testing.T) {
		hg, err := groups.NewValidatorGroup(nil)
		require.True(t, errors.Is(err, apiErrors.ErrNilFacadeHandler))
		require.Nil(t, hg)
	})

	t.Run("should work", func(t *testing.T) {
		hg, err := groups.NewValidatorGroup(&mock.FacadeStub{})
		require.NoError(t, err)
		require.NotNil(t, hg)
	})
}

type validatorStatisticsResponse struct {
	Result map[string]*state.ValidatorApiResponse `json:"statistics"`
	Error  string                                 `json:"error"`
}

type auctionListReponse struct {
	Data struct {
		Result []*common.AuctionListValidatorAPIResponse `json:"auctionList"`
	} `json:"data"`
	Error string
}

func TestValidatorStatistics_ErrorWhenFacadeFails(t *testing.T) {
	t.Parallel()

	errStr := "error in facade"

	facade := mock.FacadeStub{
		ValidatorStatisticsHandler: func() (map[string]*state.ValidatorApiResponse, error) {
			return nil, errors.New(errStr)
		},
	}

	validatorGroup, err := groups.NewValidatorGroup(&facade)
	require.NoError(t, err)

	ws := startWebServer(validatorGroup, "validator", getValidatorRoutesConfig())

	req, _ := http.NewRequest("GET", "/validator/statistics", nil)

	resp := httptest.NewRecorder()
	ws.ServeHTTP(resp, req)

	response := validatorStatisticsResponse{}
	loadResponse(resp.Body, &response)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Contains(t, response.Error, errStr)
}

func TestValidatorStatistics_ReturnsSuccessfully(t *testing.T) {
	t.Parallel()

	mapToReturn := make(map[string]*state.ValidatorApiResponse)
	mapToReturn["test"] = &state.ValidatorApiResponse{
		NumLeaderSuccess:    5,
		NumLeaderFailure:    2,
		NumValidatorSuccess: 7,
		NumValidatorFailure: 3,
	}

	facade := mock.FacadeStub{
		ValidatorStatisticsHandler: func() (map[string]*state.ValidatorApiResponse, error) {
			return mapToReturn, nil
		},
	}

	validatorGroup, err := groups.NewValidatorGroup(&facade)
	require.NoError(t, err)

	ws := startWebServer(validatorGroup, "validator", getValidatorRoutesConfig())

	req, _ := http.NewRequest("GET", "/validator/statistics", nil)

	resp := httptest.NewRecorder()
	ws.ServeHTTP(resp, req)

	response := shared.GenericAPIResponse{}
	loadResponse(resp.Body, &response)

	validatorStatistics := validatorStatisticsResponse{}
	mapResponseData := response.Data.(map[string]interface{})
	mapResponseDataBytes, _ := json.Marshal(mapResponseData)
	_ = json.Unmarshal(mapResponseDataBytes, &validatorStatistics)

	assert.Equal(t, http.StatusOK, resp.Code)

	assert.Equal(t, validatorStatistics.Result, mapToReturn)
}

func TestAuctionList_ErrorWhenFacadeFails(t *testing.T) {
	t.Parallel()

	errStr := "error in facade"

	facade := mock.FacadeStub{
		AuctionListHandler: func() ([]*common.AuctionListValidatorAPIResponse, error) {
			return nil, errors.New(errStr)
		},
	}

	validatorGroup, err := groups.NewValidatorGroup(&facade)
	require.NoError(t, err)

	ws := startWebServer(validatorGroup, "validator", getValidatorRoutesConfig())

	req, _ := http.NewRequest("GET", "/validator/auction", nil)

	resp := httptest.NewRecorder()
	ws.ServeHTTP(resp, req)

	response := auctionListReponse{}
	loadResponse(resp.Body, &response)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Contains(t, response.Error, errStr)
}

func TestAuctionList_ReturnsSuccessfully(t *testing.T) {
	t.Parallel()

	auctionListToReturn := []*common.AuctionListValidatorAPIResponse{
		{
			Owner:   "owner",
			NodeKey: "nodeKey",
			TopUp:   "112233",
		},
	}

	facade := mock.FacadeStub{
		AuctionListHandler: func() ([]*common.AuctionListValidatorAPIResponse, error) {
			return auctionListToReturn, nil
		},
	}

	validatorGroup, err := groups.NewValidatorGroup(&facade)
	require.NoError(t, err)

	ws := startWebServer(validatorGroup, "validator", getValidatorRoutesConfig())

	req, _ := http.NewRequest("GET", "/validator/auction", nil)

	resp := httptest.NewRecorder()
	ws.ServeHTTP(resp, req)

	response := auctionListReponse{}
	loadResponse(resp.Body, &response)

	assert.Equal(t, http.StatusOK, resp.Code)

	assert.Equal(t, response.Data.Result, auctionListToReturn)
}

func getValidatorRoutesConfig() config.ApiRoutesConfig {
	return config.ApiRoutesConfig{
		APIPackages: map[string]config.APIPackageConfig{
			"validator": {
				Routes: []config.RouteConfig{
					{Name: "/statistics", Open: true},
					{Name: "/auction", Open: true},
				},
			},
		},
	}
}
