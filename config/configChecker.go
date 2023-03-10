package config

import (
	"fmt"
)

// SanityCheckEnableEpochsStakingV4 checks if the enable epoch configs for stakingV4 are set correctly
func SanityCheckEnableEpochsStakingV4(cfg *Configs) error {
	enableEpochsCfg := cfg.EpochConfig.EnableEpochs
	if !areStakingV4StepsInOrder(enableEpochsCfg) {
		return errStakingV4StepsNotInOrder
	}

	numOfShards := cfg.GeneralConfig.GeneralSettings.GenesisMaxNumberOfShards
	return checkStakingV4MaxNodesChangeCfg(enableEpochsCfg, numOfShards)
}

func areStakingV4StepsInOrder(enableEpochsCfg EnableEpochs) bool {
	return (enableEpochsCfg.StakingV4Step1EnableEpoch == enableEpochsCfg.StakingV4Step2EnableEpoch-1) &&
		(enableEpochsCfg.StakingV4Step2EnableEpoch == enableEpochsCfg.StakingV4Step3EnableEpoch-1)
}

func checkStakingV4MaxNodesChangeCfg(enableEpochsCfg EnableEpochs, numOfShards uint32) error {
	maxNodesChangeCfg := enableEpochsCfg.MaxNodesChangeEnableEpoch
	if len(maxNodesChangeCfg) <= 1 {
		return errNotEnoughMaxNodesChanges
	}

	maxNodesConfigAdaptedForStakingV4 := false

	for idx, currMaxNodesChangeCfg := range maxNodesChangeCfg {
		if currMaxNodesChangeCfg.EpochEnable == enableEpochsCfg.StakingV4Step3EnableEpoch {
			maxNodesConfigAdaptedForStakingV4 = true

			if idx == 0 {
				return fmt.Errorf("found config change in MaxNodesChangeEnableEpoch for StakingV4Step3EnableEpoch = %d, but %w ",
					enableEpochsCfg.StakingV4Step3EnableEpoch, errNoMaxNodesConfigBeforeStakingV4)
			} else {
				prevMaxNodesChange := maxNodesChangeCfg[idx-1]
				err := checkMaxNodesChangedCorrectly(prevMaxNodesChange, currMaxNodesChangeCfg, numOfShards)
				if err != nil {
					return err
				}
			}

			break
		}
	}

	if !maxNodesConfigAdaptedForStakingV4 {
		return fmt.Errorf("%w = %d", errNoMaxNodesConfigChangeForStakingV4, enableEpochsCfg.StakingV4Step3EnableEpoch)
	}

	return nil
}

func checkMaxNodesChangedCorrectly(prevMaxNodesChange MaxNodesChangeConfig, currMaxNodesChange MaxNodesChangeConfig, numOfShards uint32) error {
	if prevMaxNodesChange.NodesToShufflePerShard != currMaxNodesChange.NodesToShufflePerShard {
		return errMismatchNodesToShuffle
	}

	totalShuffled := (numOfShards + 1) * prevMaxNodesChange.NodesToShufflePerShard
	expectedMaxNumNodes := prevMaxNodesChange.MaxNumNodes - totalShuffled
	if expectedMaxNumNodes != currMaxNodesChange.MaxNumNodes {
		return fmt.Errorf("expected MaxNodesChangeEnableEpoch.MaxNumNodes for StakingV4Step3EnableEpoch = %d, but got %d",
			expectedMaxNumNodes, currMaxNodesChange.MaxNumNodes)
	}

	return nil
}

// SanityCheckNodesConfig checks if the nodes limit setup is set correctly
func SanityCheckNodesConfig(
	numShards uint32,
	minNumNodesWithHysteresis uint32,
	maxNodesChange []MaxNodesChangeConfig,
) error {
	for _, maxNodesConfig := range maxNodesChange {
		err := checkMaxNodesConfig(numShards, minNumNodesWithHysteresis, maxNodesConfig)
		if err != nil {
			return fmt.Errorf("%w in MaxNodesChangeConfig at EpochEnable = %d", err, maxNodesConfig.EpochEnable)
		}
	}

	return nil
}

func checkMaxNodesConfig(
	numShards uint32,
	minNumNodesWithHysteresis uint32,
	maxNodesConfig MaxNodesChangeConfig,
) error {
	nodesToShufflePerShard := maxNodesConfig.NodesToShufflePerShard
	if nodesToShufflePerShard == 0 {
		return errZeroNodesToShufflePerShard
	}

	maxNumNodes := maxNodesConfig.MaxNumNodes
	if maxNumNodes < minNumNodesWithHysteresis {
		return fmt.Errorf("%w, maxNumNodes: %d, minNumNodesWithHysteresis: %d",
			errMaxMinNodesInvalid, maxNumNodes, minNumNodesWithHysteresis)
	}

	waitingListPerShard := (maxNumNodes - minNumNodesWithHysteresis) / (numShards + 1)
	if nodesToShufflePerShard > waitingListPerShard {
		return fmt.Errorf("%w, nodesToShufflePerShard: %d, waitingListPerShard: %d",
			errInvalidNodesToShuffle, nodesToShufflePerShard, waitingListPerShard)
	}

	return nil
}
