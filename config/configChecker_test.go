package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func generateCorrectConfig() *Configs {
	return &Configs{
		EpochConfig: &EpochConfig{
			EnableEpochs: EnableEpochs{
				StakingV4Step1EnableEpoch: 4,
				StakingV4Step2EnableEpoch: 5,
				StakingV4Step3EnableEpoch: 6,
				MaxNodesChangeEnableEpoch: []MaxNodesChangeConfig{
					{
						EpochEnable:            0,
						MaxNumNodes:            36,
						NodesToShufflePerShard: 4,
					},
					{
						EpochEnable:            1,
						MaxNumNodes:            56,
						NodesToShufflePerShard: 2,
					},
					{
						EpochEnable:            6,
						MaxNumNodes:            48,
						NodesToShufflePerShard: 2,
					},
				},
			},
		},
		GeneralConfig: &Config{
			GeneralSettings: GeneralSettingsConfig{
				GenesisMaxNumberOfShards: 3,
			},
		},
	}
}

func TestSanityCheckEnableEpochsStakingV4(t *testing.T) {
	t.Parallel()

	t.Run("correct config, should work", func(t *testing.T) {
		t.Parallel()

		cfg := generateCorrectConfig()
		err := SanityCheckEnableEpochsStakingV4(cfg)
		require.Nil(t, err)
	})

	t.Run("staking v4 steps not in ascending order, should return error", func(t *testing.T) {
		t.Parallel()

		cfg := generateCorrectConfig()
		cfg.EpochConfig.EnableEpochs.StakingV4Step1EnableEpoch = 5
		cfg.EpochConfig.EnableEpochs.StakingV4Step2EnableEpoch = 5
		err := SanityCheckEnableEpochsStakingV4(cfg)
		require.Equal(t, errStakingV4StepsNotInOrder, err)

		cfg = generateCorrectConfig()
		cfg.EpochConfig.EnableEpochs.StakingV4Step2EnableEpoch = 5
		cfg.EpochConfig.EnableEpochs.StakingV4Step3EnableEpoch = 4
		err = SanityCheckEnableEpochsStakingV4(cfg)
		require.Equal(t, errStakingV4StepsNotInOrder, err)
	})

	t.Run("staking v4 steps not in cardinal order, should return error", func(t *testing.T) {
		t.Parallel()

		cfg := generateCorrectConfig()

		cfg.EpochConfig.EnableEpochs.StakingV4Step1EnableEpoch = 1
		cfg.EpochConfig.EnableEpochs.StakingV4Step2EnableEpoch = 3
		cfg.EpochConfig.EnableEpochs.StakingV4Step3EnableEpoch = 6
		err := SanityCheckEnableEpochsStakingV4(cfg)
		require.Equal(t, errStakingV4StepsNotInOrder, err)

		cfg.EpochConfig.EnableEpochs.StakingV4Step1EnableEpoch = 1
		cfg.EpochConfig.EnableEpochs.StakingV4Step2EnableEpoch = 2
		cfg.EpochConfig.EnableEpochs.StakingV4Step3EnableEpoch = 6
		err = SanityCheckEnableEpochsStakingV4(cfg)
		require.Equal(t, errStakingV4StepsNotInOrder, err)

		cfg.EpochConfig.EnableEpochs.StakingV4Step1EnableEpoch = 1
		cfg.EpochConfig.EnableEpochs.StakingV4Step2EnableEpoch = 5
		cfg.EpochConfig.EnableEpochs.StakingV4Step3EnableEpoch = 6
		err = SanityCheckEnableEpochsStakingV4(cfg)
		require.Equal(t, errStakingV4StepsNotInOrder, err)
	})

	t.Run("no previous config for max nodes change, should return error", func(t *testing.T) {
		t.Parallel()

		cfg := generateCorrectConfig()
		cfg.EpochConfig.EnableEpochs.MaxNodesChangeEnableEpoch = []MaxNodesChangeConfig{
			{
				EpochEnable:            6,
				MaxNumNodes:            48,
				NodesToShufflePerShard: 2,
			},
		}

		err := SanityCheckEnableEpochsStakingV4(cfg)
		require.Equal(t, errNotEnoughMaxNodesChanges, err)
	})

	t.Run("no max nodes config change for StakingV4Step3EnableEpoch, should return error", func(t *testing.T) {
		t.Parallel()

		cfg := generateCorrectConfig()
		cfg.EpochConfig.EnableEpochs.MaxNodesChangeEnableEpoch = []MaxNodesChangeConfig{
			{
				EpochEnable:            1,
				MaxNumNodes:            56,
				NodesToShufflePerShard: 2,
			},
			{
				EpochEnable:            444,
				MaxNumNodes:            48,
				NodesToShufflePerShard: 2,
			},
		}

		err := SanityCheckEnableEpochsStakingV4(cfg)
		require.NotNil(t, err)
		require.True(t, strings.Contains(err.Error(), errNoMaxNodesConfigChangeForStakingV4.Error()))
		require.True(t, strings.Contains(err.Error(), "6"))
	})

	t.Run("max nodes config change for StakingV4Step3EnableEpoch has no previous config change, should return error", func(t *testing.T) {
		t.Parallel()

		cfg := generateCorrectConfig()
		cfg.EpochConfig.EnableEpochs.MaxNodesChangeEnableEpoch = []MaxNodesChangeConfig{
			{
				EpochEnable:            cfg.EpochConfig.EnableEpochs.StakingV4Step3EnableEpoch,
				MaxNumNodes:            48,
				NodesToShufflePerShard: 2,
			},
			{
				EpochEnable:            444,
				MaxNumNodes:            56,
				NodesToShufflePerShard: 2,
			},
		}

		err := SanityCheckEnableEpochsStakingV4(cfg)
		require.NotNil(t, err)
		require.ErrorIs(t, err, errNoMaxNodesConfigBeforeStakingV4)
	})

	t.Run("stakingV4 config for max nodes changed with different nodes to shuffle, should return error", func(t *testing.T) {
		t.Parallel()

		cfg := generateCorrectConfig()
		cfg.EpochConfig.EnableEpochs.MaxNodesChangeEnableEpoch[1].NodesToShufflePerShard = 2
		cfg.EpochConfig.EnableEpochs.MaxNodesChangeEnableEpoch[2].NodesToShufflePerShard = 4

		err := SanityCheckEnableEpochsStakingV4(cfg)
		require.ErrorIs(t, err, errMismatchNodesToShuffle)
	})

	t.Run("stakingV4 config for max nodes changed with wrong max num nodes, should return error", func(t *testing.T) {
		t.Parallel()

		cfg := generateCorrectConfig()
		cfg.EpochConfig.EnableEpochs.MaxNodesChangeEnableEpoch[2].MaxNumNodes = 56

		err := SanityCheckEnableEpochsStakingV4(cfg)
		require.NotNil(t, err)
		require.True(t, strings.Contains(err.Error(), "expected"))
		require.True(t, strings.Contains(err.Error(), "48"))
		require.True(t, strings.Contains(err.Error(), "got"))
		require.True(t, strings.Contains(err.Error(), "56"))
	})
}

func TestSanityCheckNodesConfig(t *testing.T) {
	t.Parallel()

	numShards := uint32(3)
	t.Run("should work", func(t *testing.T) {
		t.Parallel()

		cfg := generateCorrectConfig().EpochConfig.EnableEpochs.MaxNodesChangeEnableEpoch
		err := SanityCheckNodesConfig(numShards, 20, cfg)
		require.Nil(t, err)

		cfg = []MaxNodesChangeConfig{
			{
				EpochEnable:            1,
				MaxNumNodes:            3200,
				NodesToShufflePerShard: 80,
			},
			{
				EpochEnable:            2,
				MaxNumNodes:            2880,
				NodesToShufflePerShard: 80,
			},
			{
				EpochEnable:            3,
				MaxNumNodes:            2240,
				NodesToShufflePerShard: 80,
			},
			{
				EpochEnable:            4,
				MaxNumNodes:            2240,
				NodesToShufflePerShard: 40,
			},
		}
		err = SanityCheckNodesConfig(numShards, 1920, cfg)
		require.Nil(t, err)
	})

	t.Run("zero nodes to shuffle per shard, should return error", func(t *testing.T) {
		t.Parallel()

		cfg := []MaxNodesChangeConfig{
			{
				EpochEnable:            4,
				MaxNumNodes:            3200,
				NodesToShufflePerShard: 0,
			},
		}
		err := SanityCheckNodesConfig(numShards, 1920, cfg)
		require.NotNil(t, err)
		require.True(t, strings.Contains(err.Error(), errZeroNodesToShufflePerShard.Error()))
		require.True(t, strings.Contains(err.Error(), "at EpochEnable = 4"))
	})

	t.Run("maxNumNodes < minNumNodesWithHysteresis, should return error ", func(t *testing.T) {
		t.Parallel()

		cfg := []MaxNodesChangeConfig{
			{
				EpochEnable:            4,
				MaxNumNodes:            1900,
				NodesToShufflePerShard: 80,
			},
		}
		err := SanityCheckNodesConfig(numShards, 1920, cfg)
		require.NotNil(t, err)
		require.True(t, strings.Contains(err.Error(), errMaxMinNodesInvalid.Error()))
		require.True(t, strings.Contains(err.Error(), "maxNumNodes: 1900"))
		require.True(t, strings.Contains(err.Error(), "minNumNodesWithHysteresis: 1920"))
	})

	t.Run("invalid nodes to shuffle per shard, should return error ", func(t *testing.T) {
		t.Parallel()

		cfg := []MaxNodesChangeConfig{
			{
				EpochEnable:            3,
				MaxNumNodes:            2240,
				NodesToShufflePerShard: 81,
			},
		}
		err := SanityCheckNodesConfig(numShards, 1920, cfg)
		require.NotNil(t, err)
		require.True(t, strings.Contains(err.Error(), errInvalidNodesToShuffle.Error()))
		require.True(t, strings.Contains(err.Error(), "nodesToShufflePerShard: 81"))
		require.True(t, strings.Contains(err.Error(), "waitingListPerShard: 80"))
	})
}
