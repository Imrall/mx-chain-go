package config

import "errors"

var errStakingV4StepsNotInOrder = errors.New("staking v4 enable epoch steps should be in cardinal order(e.g.: StakingV4Step1EnableEpoch = 2, StakingV4Step2EnableEpoch = 3, StakingV4Step3EnableEpoch = 4)")

var errNotEnoughMaxNodesChanges = errors.New("not enough entries in MaxNodesChangeEnableEpoch config; expected one entry before stakingV4 and another one starting StakingV4Step3EnableEpoch")

var errNoMaxNodesConfigBeforeStakingV4 = errors.New("no previous config change entry in MaxNodesChangeEnableEpoch before entry with EpochEnable = StakingV4Step3EnableEpoch")

var errMismatchNodesToShuffle = errors.New("previous MaxNodesChangeEnableEpoch.NodesToShufflePerShard != MaxNodesChangeEnableEpoch.NodesToShufflePerShard with EnableEpoch = StakingV4Step3EnableEpoch")

var errNoMaxNodesConfigChangeForStakingV4 = errors.New("no MaxNodesChangeEnableEpoch config found for EpochEnable = StakingV4Step3EnableEpoch")

var errZeroNodesToShufflePerShard = errors.New("zero nodes to shuffle per shard found in config")

var errMaxMinNodesInvalid = errors.New("number of min nodes with hysteresis > number of max nodes")

var errInvalidNodesToShuffle = errors.New("number of nodes to shuffle per shard > waiting list size per shard")
