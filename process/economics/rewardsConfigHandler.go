package economics

import (
	"fmt"
	"math/big"
	"sort"

	"github.com/multiversx/mx-chain-core-go/core"
	"github.com/multiversx/mx-chain-core-go/core/check"
	"github.com/multiversx/mx-chain-go/common"
	"github.com/multiversx/mx-chain-go/config"
	"github.com/multiversx/mx-chain-go/process"
	"github.com/multiversx/mx-chain-go/statusHandler"
)

type rewardsConfig struct {
	rewardsSettingEpoch              uint32
	leaderPercentage                 float64
	protocolSustainabilityPercentage float64
	protocolSustainabilityAddress    string
	developerPercentage              float64
	topUpGradientPoint               *big.Int
	topUpFactor                      float64
}

type rewardsConfigHandler struct {
	statusHandler         core.AppStatusHandler
	rewardsConfigSettings []*rewardsConfig
}

// NewRewardsConfigHandler returns a new instance of rewardsConfigHandler
func NewRewardsConfigHandler(rewardsSettings config.RewardsSettings) (*rewardsConfigHandler, error) {
	rewardsConfigSlice, err := checkAndParseRewardsSettings(rewardsSettings)
	if err != nil {
		return nil, err
	}

	sort.Slice(rewardsConfigSlice, func(i, j int) bool {
		return rewardsConfigSlice[i].rewardsSettingEpoch < rewardsConfigSlice[j].rewardsSettingEpoch
	})

	return &rewardsConfigHandler{
		statusHandler:         statusHandler.NewNilStatusHandler(),
		rewardsConfigSettings: rewardsConfigSlice,
	}, nil
}

// SetStatusHandler sets the provided status handler if not nil
func (handler *rewardsConfigHandler) SetStatusHandler(statusHandler core.AppStatusHandler) error {
	if check.IfNil(statusHandler) {
		return core.ErrNilAppStatusHandler
	}

	handler.statusHandler = statusHandler

	return nil
}

// GetLeaderPercentage returns the leader percentage in a specific epoch
func (handler *rewardsConfigHandler) GetLeaderPercentage(epoch uint32) float64 {
	rc := handler.getRewardsConfigForEpoch(epoch)
	return rc.leaderPercentage
}

// GetDeveloperPercentage returns the developer percentage in a specific epoch
func (handler *rewardsConfigHandler) GetDeveloperPercentage(epoch uint32) float64 {
	rc := handler.getRewardsConfigForEpoch(epoch)
	return rc.developerPercentage
}

// GetProtocolSustainabilityPercentage returns the protocol sustainability percentage in a specific epoch
func (handler *rewardsConfigHandler) GetProtocolSustainabilityPercentage(epoch uint32) float64 {
	rc := handler.getRewardsConfigForEpoch(epoch)
	return rc.protocolSustainabilityPercentage
}

// GetProtocolSustainabilityAddress returns the protocol sustainability address in a specific epoch
func (handler *rewardsConfigHandler) GetProtocolSustainabilityAddress(epoch uint32) string {
	rc := handler.getRewardsConfigForEpoch(epoch)
	return rc.protocolSustainabilityAddress
}

// GetTopUpFactor returns the top-up factor in a specific epoch
func (handler *rewardsConfigHandler) GetTopUpFactor(epoch uint32) float64 {
	rc := handler.getRewardsConfigForEpoch(epoch)
	return rc.topUpFactor
}

// GetTopUpGradientPoint returns the top-up gradient point in a specific epoch
func (handler *rewardsConfigHandler) GetTopUpGradientPoint(epoch uint32) *big.Int {
	rc := handler.getRewardsConfigForEpoch(epoch)
	return rc.topUpGradientPoint
}

func (handler *rewardsConfigHandler) getRewardsConfigForEpoch(epoch uint32) *rewardsConfig {
	rewardsConfigSetting := handler.rewardsConfigSettings[0]
	for i := 1; i < len(handler.rewardsConfigSettings); i++ {
		// as we go from epoch k to epoch k+1 we set the config for epoch k before computing the economics/rewards
		if epoch > handler.rewardsConfigSettings[i].rewardsSettingEpoch {
			rewardsConfigSetting = handler.rewardsConfigSettings[i]
		}
	}

	return rewardsConfigSetting
}

func (handler *rewardsConfigHandler) updateRewardsConfigMetrics(epoch uint32) {
	rc := handler.getRewardsConfigForEpoch(epoch)

	// TODO: add all metrics
	handler.statusHandler.SetStringValue(common.MetricLeaderPercentage, fmt.Sprintf("%f", rc.leaderPercentage))
	handler.statusHandler.SetStringValue(common.MetricRewardsTopUpGradientPoint, rc.topUpGradientPoint.String())
	handler.statusHandler.SetStringValue(common.MetricTopUpFactor, fmt.Sprintf("%f", rc.topUpFactor))

	log.Debug("economics: rewardsConfigHandler",
		"epoch", rc.rewardsSettingEpoch,
		"leaderPercentage", rc.leaderPercentage,
		"protocolSustainabilityPercentage", rc.protocolSustainabilityPercentage,
		"protocolSustainabilityAddress", rc.protocolSustainabilityAddress,
		"developerPercentage", rc.developerPercentage,
		"topUpFactor", rc.topUpFactor,
		"topUpGradientPoint", rc.topUpGradientPoint,
	)
}

func checkAndParseRewardsSettings(rewardsSettings config.RewardsSettings) ([]*rewardsConfig, error) {
	rewardsConfigSlice := make([]*rewardsConfig, 0, len(rewardsSettings.RewardsConfigByEpoch))
	for _, rewardsCfg := range rewardsSettings.RewardsConfigByEpoch {
		err := checkRewardConfig(rewardsCfg)
		if err != nil {
			return nil, err
		}

		topUpGradientPoint, _ := big.NewInt(0).SetString(rewardsCfg.TopUpGradientPoint, 10)

		rewardsConfigSlice = append(rewardsConfigSlice, &rewardsConfig{
			rewardsSettingEpoch:              rewardsCfg.EpochEnable,
			leaderPercentage:                 rewardsCfg.LeaderPercentage,
			protocolSustainabilityPercentage: rewardsCfg.ProtocolSustainabilityPercentage,
			protocolSustainabilityAddress:    rewardsCfg.ProtocolSustainabilityAddress,
			developerPercentage:              rewardsCfg.DeveloperPercentage,
			topUpGradientPoint:               topUpGradientPoint,
			topUpFactor:                      rewardsCfg.TopUpFactor,
		})
	}

	return rewardsConfigSlice, nil
}

func checkRewardConfig(rewardsCfg config.EpochRewardSettings) error {
	if isPercentageInvalid(rewardsCfg.LeaderPercentage) ||
		isPercentageInvalid(rewardsCfg.DeveloperPercentage) ||
		isPercentageInvalid(rewardsCfg.ProtocolSustainabilityPercentage) ||
		isPercentageInvalid(rewardsCfg.TopUpFactor) {
		return process.ErrInvalidRewardsPercentages
	}

	if len(rewardsCfg.ProtocolSustainabilityAddress) == 0 {
		return process.ErrNilProtocolSustainabilityAddress
	}

	_, ok := big.NewInt(0).SetString(rewardsCfg.TopUpGradientPoint, 10)
	if !ok {
		return process.ErrInvalidRewardsTopUpGradientPoint
	}

	return nil
}

func isPercentageInvalid(percentage float64) bool {
	isLessThanZero := percentage < 0.0
	isGreaterThanOne := percentage > 1.0
	if isLessThanZero || isGreaterThanOne {
		return true
	}
	return false
}
