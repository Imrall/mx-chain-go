package metachain

import (
	"math/big"
	"strings"
	"testing"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go/common"
	"github.com/ElrondNetwork/elrond-go/common/forking"
	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/epochStart"
	"github.com/ElrondNetwork/elrond-go/epochStart/notifier"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/sharding"
	"github.com/ElrondNetwork/elrond-go/state"
	"github.com/ElrondNetwork/elrond-go/testscommon/stakingcommon"
	"github.com/stretchr/testify/require"
)

func createSoftAuctionConfig() config.SoftAuctionConfig {
	return config.SoftAuctionConfig{
		TopUpStep: "10",
		MinTopUp:  "1",
		MaxTopUp:  "32000000",
	}
}

func createAuctionListSelectorArgs(maxNodesChangeConfig []config.MaxNodesChangeConfig) AuctionListSelectorArgs {
	epochNotifier := forking.NewGenericEpochNotifier()
	nodesConfigProvider, _ := notifier.NewNodesConfigProvider(epochNotifier, maxNodesChangeConfig)

	argsStakingDataProvider := createStakingDataProviderArgs()
	stakingSCProvider, _ := NewStakingDataProvider(argsStakingDataProvider)

	shardCoordinator, _ := sharding.NewMultiShardCoordinator(3, core.MetachainShardId)
	return AuctionListSelectorArgs{
		ShardCoordinator:             shardCoordinator,
		StakingDataProvider:          stakingSCProvider,
		MaxNodesChangeConfigProvider: nodesConfigProvider,
		SoftAuctionConfig:            createSoftAuctionConfig(),
	}
}

func createFullAuctionListSelectorArgs(maxNodesChangeConfig []config.MaxNodesChangeConfig) (AuctionListSelectorArgs, ArgsNewEpochStartSystemSCProcessing) {
	epochNotifier := forking.NewGenericEpochNotifier()
	nodesConfigProvider, _ := notifier.NewNodesConfigProvider(epochNotifier, maxNodesChangeConfig)

	argsSystemSC, _ := createFullArgumentsForSystemSCProcessing(0, createMemUnit())
	argsSystemSC.StakingDataProvider.EpochConfirmed(stakingV4EnableEpoch, 0)
	argsSystemSC.MaxNodesChangeConfigProvider = nodesConfigProvider
	return AuctionListSelectorArgs{
		ShardCoordinator:             argsSystemSC.ShardCoordinator,
		StakingDataProvider:          argsSystemSC.StakingDataProvider,
		MaxNodesChangeConfigProvider: nodesConfigProvider,
		SoftAuctionConfig:            createSoftAuctionConfig(),
	}, argsSystemSC
}

func fillValidatorsInfo(t *testing.T, validatorsMap state.ShardValidatorsInfoMapHandler, sdp epochStart.StakingDataProvider) {
	for _, validator := range validatorsMap.GetAllValidatorsInfo() {
		err := sdp.FillValidatorInfo(validator)
		require.Nil(t, err)
	}
}

func TestNewAuctionListSelector(t *testing.T) {
	t.Parallel()

	t.Run("nil shard coordinator", func(t *testing.T) {
		t.Parallel()
		args := createAuctionListSelectorArgs(nil)
		args.ShardCoordinator = nil
		als, err := NewAuctionListSelector(args)
		require.Nil(t, als)
		require.Equal(t, epochStart.ErrNilShardCoordinator, err)
	})

	t.Run("nil staking data provider", func(t *testing.T) {
		t.Parallel()
		args := createAuctionListSelectorArgs(nil)
		args.StakingDataProvider = nil
		als, err := NewAuctionListSelector(args)
		require.Nil(t, als)
		require.Equal(t, epochStart.ErrNilStakingDataProvider, err)
	})

	t.Run("nil max nodes change config provider", func(t *testing.T) {
		t.Parallel()
		args := createAuctionListSelectorArgs(nil)
		args.MaxNodesChangeConfigProvider = nil
		als, err := NewAuctionListSelector(args)
		require.Nil(t, als)
		require.Equal(t, epochStart.ErrNilMaxNodesChangeConfigProvider, err)
	})

	t.Run("invalid soft auction config", func(t *testing.T) {
		t.Parallel()
		args := createAuctionListSelectorArgs(nil)
		args.SoftAuctionConfig.TopUpStep = "0"
		als, err := NewAuctionListSelector(args)
		require.Nil(t, als)
		requireInvalidValueError(t, err, "step")
	})

	t.Run("should work", func(t *testing.T) {
		t.Parallel()
		args := createAuctionListSelectorArgs(nil)
		als, err := NewAuctionListSelector(args)
		require.NotNil(t, als)
		require.Nil(t, err)
	})
}

func requireInvalidValueError(t *testing.T, err error, msgToContain string) {
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), process.ErrInvalidValue.Error()))
	require.True(t, strings.Contains(err.Error(), msgToContain))
}

func TestGetAuctionConfig(t *testing.T) {
	t.Parallel()

	t.Run("invalid step", func(t *testing.T) {
		t.Parallel()

		cfg := createSoftAuctionConfig()

		cfg.TopUpStep = "dsa"
		res, err := getAuctionConfig(cfg, 1)
		require.Nil(t, res)
		requireInvalidValueError(t, err, "step")

		cfg.TopUpStep = "-1"
		res, err = getAuctionConfig(cfg, 1)
		require.Nil(t, res)
		requireInvalidValueError(t, err, "step")

		cfg.TopUpStep = "0"
		res, err = getAuctionConfig(cfg, 1)
		require.Nil(t, res)
		requireInvalidValueError(t, err, "step")
	})

	t.Run("invalid min top up", func(t *testing.T) {
		t.Parallel()

		cfg := createSoftAuctionConfig()

		cfg.MinTopUp = "dsa"
		res, err := getAuctionConfig(cfg, 1)
		require.Nil(t, res)
		requireInvalidValueError(t, err, "min top up")

		cfg.MinTopUp = "-1"
		res, err = getAuctionConfig(cfg, 1)
		require.Nil(t, res)
		requireInvalidValueError(t, err, "min top up")

		cfg.MinTopUp = "0"
		res, err = getAuctionConfig(cfg, 1)
		require.Nil(t, res)
		requireInvalidValueError(t, err, "min top up")
	})

	t.Run("invalid max top up", func(t *testing.T) {
		t.Parallel()

		cfg := createSoftAuctionConfig()

		cfg.MaxTopUp = "dsa"
		res, err := getAuctionConfig(cfg, 1)
		require.Nil(t, res)
		requireInvalidValueError(t, err, "max top up")

		cfg.MaxTopUp = "-1"
		res, err = getAuctionConfig(cfg, 1)
		require.Nil(t, res)
		requireInvalidValueError(t, err, "max top up")

		cfg.MaxTopUp = "0"
		res, err = getAuctionConfig(cfg, 1)
		require.Nil(t, res)
		requireInvalidValueError(t, err, "max top up")
	})

	t.Run("invalid denomination", func(t *testing.T) {
		t.Parallel()

		cfg := createSoftAuctionConfig()

		res, err := getAuctionConfig(cfg, -1)
		require.Nil(t, res)
		requireInvalidValueError(t, err, "denomination")
	})

	t.Run("should work", func(t *testing.T) {
		t.Parallel()

		cfg := config.SoftAuctionConfig{
			TopUpStep: "10",
			MinTopUp:  "1",
			MaxTopUp:  "444",
		}

		res, err := getAuctionConfig(cfg, 4)
		require.Nil(t, err)
		require.Equal(t, &auctionConfig{
			step:        big.NewInt(10),
			minTopUp:    big.NewInt(1),
			maxTopUp:    big.NewInt(444),
			denominator: big.NewInt(10000),
		}, res)
	})
}

func TestAuctionListSelector_SelectNodesFromAuction(t *testing.T) {
	t.Parallel()

	t.Run("nil randomness, expect error", func(t *testing.T) {
		t.Parallel()

		args := createAuctionListSelectorArgs(nil)
		als, _ := NewAuctionListSelector(args)
		err := als.SelectNodesFromAuctionList(state.NewShardValidatorsInfoMap(), nil)
		require.Equal(t, process.ErrNilRandSeed, err)
	})

	t.Run("empty auction list", func(t *testing.T) {
		t.Parallel()

		owner1 := []byte("owner1")
		owner1StakedKeys := [][]byte{[]byte("pubKey0")}

		validatorsInfo := state.NewShardValidatorsInfoMap()
		_ = validatorsInfo.Add(createValidatorInfo(owner1StakedKeys[0], common.EligibleList, owner1, 0))

		args, argsSystemSC := createFullAuctionListSelectorArgs([]config.MaxNodesChangeConfig{{MaxNumNodes: 2}})
		stakingcommon.RegisterValidatorKeys(argsSystemSC.UserAccountsDB, owner1, owner1, owner1StakedKeys, big.NewInt(1000), argsSystemSC.Marshalizer)
		fillValidatorsInfo(t, validatorsInfo, argsSystemSC.StakingDataProvider)

		als, _ := NewAuctionListSelector(args)
		err := als.SelectNodesFromAuctionList(state.NewShardValidatorsInfoMap(), []byte("rnd"))
		require.Nil(t, err)
		expectedValidatorsInfo := map[uint32][]state.ValidatorInfoHandler{
			0: {
				createValidatorInfo(owner1StakedKeys[0], common.EligibleList, owner1, 0),
			},
		}
		require.Equal(t, expectedValidatorsInfo, validatorsInfo.GetShardValidatorsInfoMap())
	})

	t.Run("not enough available slots to select auction nodes", func(t *testing.T) {
		t.Parallel()

		owner1 := []byte("owner1")
		owner2 := []byte("owner2")
		owner1StakedKeys := [][]byte{[]byte("pubKey0")}
		owner2StakedKeys := [][]byte{[]byte("pubKey1")}

		validatorsInfo := state.NewShardValidatorsInfoMap()
		_ = validatorsInfo.Add(createValidatorInfo(owner1StakedKeys[0], common.EligibleList, owner1, 0))
		_ = validatorsInfo.Add(createValidatorInfo(owner2StakedKeys[0], common.AuctionList, owner2, 0))

		args, argsSystemSC := createFullAuctionListSelectorArgs([]config.MaxNodesChangeConfig{{MaxNumNodes: 1}})
		stakingcommon.RegisterValidatorKeys(argsSystemSC.UserAccountsDB, owner1, owner1, owner1StakedKeys, big.NewInt(1000), argsSystemSC.Marshalizer)
		stakingcommon.RegisterValidatorKeys(argsSystemSC.UserAccountsDB, owner2, owner2, owner2StakedKeys, big.NewInt(1000), argsSystemSC.Marshalizer)
		fillValidatorsInfo(t, validatorsInfo, argsSystemSC.StakingDataProvider)

		als, _ := NewAuctionListSelector(args)
		err := als.SelectNodesFromAuctionList(validatorsInfo, []byte("rnd"))
		require.Nil(t, err)
		expectedValidatorsInfo := map[uint32][]state.ValidatorInfoHandler{
			0: {
				createValidatorInfo(owner1StakedKeys[0], common.EligibleList, owner1, 0),
				createValidatorInfo(owner2StakedKeys[0], common.AuctionList, owner2, 0),
			},
		}
		require.Equal(t, expectedValidatorsInfo, validatorsInfo.GetShardValidatorsInfoMap())
	})

	t.Run("one eligible + one auction, max num nodes = 1, number of nodes after shuffling = 0, expect node in auction is selected", func(t *testing.T) {
		t.Parallel()

		owner1 := []byte("owner1")
		owner2 := []byte("owner2")
		owner1StakedKeys := [][]byte{[]byte("pubKey0")}
		owner2StakedKeys := [][]byte{[]byte("pubKey1")}

		validatorsInfo := state.NewShardValidatorsInfoMap()
		_ = validatorsInfo.Add(createValidatorInfo(owner1StakedKeys[0], common.EligibleList, owner1, 0))
		_ = validatorsInfo.Add(createValidatorInfo(owner2StakedKeys[0], common.AuctionList, owner2, 0))

		args, argsSystemSC := createFullAuctionListSelectorArgs([]config.MaxNodesChangeConfig{{MaxNumNodes: 1, NodesToShufflePerShard: 1}})
		stakingcommon.RegisterValidatorKeys(argsSystemSC.UserAccountsDB, owner1, owner1, owner1StakedKeys, big.NewInt(1000), argsSystemSC.Marshalizer)
		stakingcommon.RegisterValidatorKeys(argsSystemSC.UserAccountsDB, owner2, owner2, owner2StakedKeys, big.NewInt(1000), argsSystemSC.Marshalizer)
		fillValidatorsInfo(t, validatorsInfo, argsSystemSC.StakingDataProvider)

		als, _ := NewAuctionListSelector(args)
		err := als.SelectNodesFromAuctionList(validatorsInfo, []byte("rnd"))
		require.Nil(t, err)
		expectedValidatorsInfo := map[uint32][]state.ValidatorInfoHandler{
			0: {
				createValidatorInfo(owner1StakedKeys[0], common.EligibleList, owner1, 0),
				createValidatorInfo(owner2StakedKeys[0], common.SelectedFromAuctionList, owner2, 0),
			},
		}
		require.Equal(t, expectedValidatorsInfo, validatorsInfo.GetShardValidatorsInfoMap())
	})

	t.Run("two available slots for auction nodes, but only one node in auction", func(t *testing.T) {
		t.Parallel()

		owner1 := []byte("owner1")
		owner1StakedKeys := [][]byte{[]byte("pubKey0")}
		validatorsInfo := state.NewShardValidatorsInfoMap()
		_ = validatorsInfo.Add(createValidatorInfo(owner1StakedKeys[0], common.AuctionList, owner1, 0))

		args, argsSystemSC := createFullAuctionListSelectorArgs([]config.MaxNodesChangeConfig{{MaxNumNodes: 2}})
		stakingcommon.RegisterValidatorKeys(argsSystemSC.UserAccountsDB, owner1, owner1, owner1StakedKeys, big.NewInt(1000), argsSystemSC.Marshalizer)
		fillValidatorsInfo(t, validatorsInfo, argsSystemSC.StakingDataProvider)

		als, _ := NewAuctionListSelector(args)
		err := als.SelectNodesFromAuctionList(validatorsInfo, []byte("rnd"))
		require.Nil(t, err)
		expectedValidatorsInfo := map[uint32][]state.ValidatorInfoHandler{
			0: {
				createValidatorInfo(owner1StakedKeys[0], common.SelectedFromAuctionList, owner1, 0),
			},
		}
		require.Equal(t, expectedValidatorsInfo, validatorsInfo.GetShardValidatorsInfoMap())
	})
}

func TestAuctionListSelector_calcSoftAuctionNodesConfigEdgeCases(t *testing.T) {
	t.Parallel()

	randomness := []byte("pk0")
	args := createAuctionListSelectorArgs(nil)
	als, _ := NewAuctionListSelector(args)

	t.Run("two validators, both have zero top up", func(t *testing.T) {
		t.Parallel()

		v1 := &state.ValidatorInfo{PublicKey: []byte("pk1")}
		v2 := &state.ValidatorInfo{PublicKey: []byte("pk2")}

		owner1 := "owner1"
		owner2 := "owner2"
		ownersData := map[string]*ownerAuctionData{
			owner1: {
				numActiveNodes:           0,
				numAuctionNodes:          1,
				numQualifiedAuctionNodes: 1,
				numStakedNodes:           1,
				totalTopUp:               big.NewInt(0),
				topUpPerNode:             big.NewInt(0),
				qualifiedTopUpPerNode:    big.NewInt(0),
				auctionList:              []state.ValidatorInfoHandler{v1},
			},
			owner2: {
				numActiveNodes:           0,
				numAuctionNodes:          1,
				numQualifiedAuctionNodes: 1,
				numStakedNodes:           1,
				totalTopUp:               big.NewInt(0),
				topUpPerNode:             big.NewInt(0),
				qualifiedTopUpPerNode:    big.NewInt(0),
				auctionList:              []state.ValidatorInfoHandler{v2},
			},
		}

		minTopUp, maxTopUp := als.getMinMaxPossibleTopUp(ownersData)
		require.Equal(t, als.softAuctionConfig.minTopUp, minTopUp)
		require.Equal(t, als.softAuctionConfig.minTopUp, maxTopUp)

		softAuctionConfig := als.calcSoftAuctionNodesConfig(ownersData, 2)
		require.Equal(t, ownersData, softAuctionConfig)
		selectedNodes := als.selectNodes(softAuctionConfig, 2, randomness)
		require.Equal(t, []state.ValidatorInfoHandler{v2, v1}, selectedNodes)

		softAuctionConfig = als.calcSoftAuctionNodesConfig(ownersData, 1)
		require.Equal(t, ownersData, softAuctionConfig)
		selectedNodes = als.selectNodes(softAuctionConfig, 1, randomness)
		require.Equal(t, []state.ValidatorInfoHandler{v2}, selectedNodes)
	})

	t.Run("one validator with zero top up, one with min top up, one with top up", func(t *testing.T) {
		t.Parallel()

		v1 := &state.ValidatorInfo{PublicKey: []byte("pk1")}
		v2 := &state.ValidatorInfo{PublicKey: []byte("pk2")}
		v3 := &state.ValidatorInfo{PublicKey: []byte("pk3")}

		owner1 := "owner1"
		owner2 := "owner2"
		owner3 := "owner3"
		ownersData := map[string]*ownerAuctionData{
			owner1: {
				numActiveNodes:           0,
				numAuctionNodes:          1,
				numQualifiedAuctionNodes: 1,
				numStakedNodes:           1,
				totalTopUp:               big.NewInt(0),
				topUpPerNode:             big.NewInt(0),
				qualifiedTopUpPerNode:    big.NewInt(0),
				auctionList:              []state.ValidatorInfoHandler{v1},
			},
			owner2: {
				numActiveNodes:           0,
				numAuctionNodes:          1,
				numQualifiedAuctionNodes: 1,
				numStakedNodes:           1,
				totalTopUp:               big.NewInt(1),
				topUpPerNode:             big.NewInt(1),
				qualifiedTopUpPerNode:    big.NewInt(1),
				auctionList:              []state.ValidatorInfoHandler{v2},
			},
			owner3: {
				numActiveNodes:           0,
				numAuctionNodes:          1,
				numQualifiedAuctionNodes: 1,
				numStakedNodes:           1,
				totalTopUp:               big.NewInt(1000),
				topUpPerNode:             big.NewInt(1000),
				qualifiedTopUpPerNode:    big.NewInt(1000),
				auctionList:              []state.ValidatorInfoHandler{v3},
			},
		}

		minTopUp, maxTopUp := als.getMinMaxPossibleTopUp(ownersData)
		require.Equal(t, big.NewInt(1), minTopUp)
		require.Equal(t, big.NewInt(1000), maxTopUp)

		softAuctionConfig := als.calcSoftAuctionNodesConfig(ownersData, 3)
		require.Equal(t, ownersData, softAuctionConfig)
		selectedNodes := als.selectNodes(softAuctionConfig, 3, randomness)
		require.Equal(t, []state.ValidatorInfoHandler{v3, v2, v1}, selectedNodes)

		softAuctionConfig = als.calcSoftAuctionNodesConfig(ownersData, 2)
		expectedSoftAuctionConfig := copyOwnersData(softAuctionConfig)
		delete(expectedSoftAuctionConfig, owner1)
		require.Equal(t, expectedSoftAuctionConfig, softAuctionConfig)
		selectedNodes = als.selectNodes(softAuctionConfig, 2, randomness)
		require.Equal(t, []state.ValidatorInfoHandler{v3, v2}, selectedNodes)

		softAuctionConfig = als.calcSoftAuctionNodesConfig(ownersData, 1)
		delete(expectedSoftAuctionConfig, owner2)
		require.Equal(t, expectedSoftAuctionConfig, softAuctionConfig)
		selectedNodes = als.selectNodes(softAuctionConfig, 1, randomness)
		require.Equal(t, []state.ValidatorInfoHandler{v3}, selectedNodes)
	})

	t.Run("two validators, both have same top up", func(t *testing.T) {
		v1 := &state.ValidatorInfo{PublicKey: []byte("pk1")}
		v2 := &state.ValidatorInfo{PublicKey: []byte("pk2")}

		owner1 := "owner1"
		owner2 := "owner2"
		ownersData := map[string]*ownerAuctionData{
			owner1: {
				numActiveNodes:           0,
				numAuctionNodes:          1,
				numQualifiedAuctionNodes: 1,
				numStakedNodes:           1,
				totalTopUp:               big.NewInt(1000),
				topUpPerNode:             big.NewInt(1000),
				qualifiedTopUpPerNode:    big.NewInt(1000),
				auctionList:              []state.ValidatorInfoHandler{v1},
			},
			owner2: {
				numActiveNodes:           0,
				numAuctionNodes:          1,
				numQualifiedAuctionNodes: 1,
				numStakedNodes:           1,
				totalTopUp:               big.NewInt(1000),
				topUpPerNode:             big.NewInt(1000),
				qualifiedTopUpPerNode:    big.NewInt(1000),
				auctionList:              []state.ValidatorInfoHandler{v2},
			},
		}

		minTopUp, maxTopUp := als.getMinMaxPossibleTopUp(ownersData)
		require.Equal(t, big.NewInt(1000), minTopUp)
		require.Equal(t, big.NewInt(1000), maxTopUp)

		softAuctionConfig := als.calcSoftAuctionNodesConfig(ownersData, 2)
		require.Equal(t, ownersData, softAuctionConfig)
		selectedNodes := als.selectNodes(softAuctionConfig, 2, randomness)
		require.Equal(t, []state.ValidatorInfoHandler{v2, v1}, selectedNodes)

		softAuctionConfig = als.calcSoftAuctionNodesConfig(ownersData, 1)
		require.Equal(t, ownersData, softAuctionConfig)
		selectedNodes = als.selectNodes(softAuctionConfig, 1, randomness)
		require.Equal(t, []state.ValidatorInfoHandler{v2}, selectedNodes)
	})

	t.Run("two validators, top up difference less than step", func(t *testing.T) {
		v1 := &state.ValidatorInfo{PublicKey: []byte("pk1")}
		v2 := &state.ValidatorInfo{PublicKey: []byte("pk2")}

		owner1 := "owner1"
		owner2 := "owner2"
		ownersData := map[string]*ownerAuctionData{
			owner1: {
				numActiveNodes:           0,
				numAuctionNodes:          1,
				numQualifiedAuctionNodes: 1,
				numStakedNodes:           1,
				totalTopUp:               big.NewInt(1000),
				topUpPerNode:             big.NewInt(1000),
				qualifiedTopUpPerNode:    big.NewInt(1000),
				auctionList:              []state.ValidatorInfoHandler{v1},
			},
			owner2: {
				numActiveNodes:           0,
				numAuctionNodes:          1,
				numQualifiedAuctionNodes: 1,
				numStakedNodes:           1,
				totalTopUp:               big.NewInt(995),
				topUpPerNode:             big.NewInt(995),
				qualifiedTopUpPerNode:    big.NewInt(995),
				auctionList:              []state.ValidatorInfoHandler{v2},
			},
		}

		minTopUp, maxTopUp := als.getMinMaxPossibleTopUp(ownersData)
		require.Equal(t, big.NewInt(995), minTopUp)
		require.Equal(t, big.NewInt(1000), maxTopUp)

		softAuctionConfig := als.calcSoftAuctionNodesConfig(ownersData, 2)
		require.Equal(t, ownersData, softAuctionConfig)
		selectedNodes := als.selectNodes(softAuctionConfig, 2, randomness)
		require.Equal(t, []state.ValidatorInfoHandler{v1, v2}, selectedNodes)

		softAuctionConfig = als.calcSoftAuctionNodesConfig(ownersData, 1)
		require.Equal(t, ownersData, softAuctionConfig)
		selectedNodes = als.selectNodes(softAuctionConfig, 1, randomness)
		require.Equal(t, []state.ValidatorInfoHandler{v1}, selectedNodes)
	})

	t.Run("three validators, top up difference equal to step", func(t *testing.T) {
		v1 := &state.ValidatorInfo{PublicKey: []byte("pk1")}
		v2 := &state.ValidatorInfo{PublicKey: []byte("pk2")}
		v0 := &state.ValidatorInfo{PublicKey: []byte("pk0")}

		owner1 := "owner1"
		owner2 := "owner2"
		ownersData := map[string]*ownerAuctionData{
			owner1: {
				numActiveNodes:           0,
				numAuctionNodes:          1,
				numQualifiedAuctionNodes: 1,
				numStakedNodes:           1,
				totalTopUp:               big.NewInt(1000),
				topUpPerNode:             big.NewInt(1000),
				qualifiedTopUpPerNode:    big.NewInt(1000),
				auctionList:              []state.ValidatorInfoHandler{v1},
			},
			owner2: {
				numActiveNodes:           0,
				numAuctionNodes:          2,
				numQualifiedAuctionNodes: 2,
				numStakedNodes:           2,
				totalTopUp:               big.NewInt(1980),
				topUpPerNode:             big.NewInt(990),
				qualifiedTopUpPerNode:    big.NewInt(990),
				auctionList:              []state.ValidatorInfoHandler{v2, v0},
			},
		}

		minTopUp, maxTopUp := als.getMinMaxPossibleTopUp(ownersData)
		require.Equal(t, big.NewInt(990), minTopUp)
		require.Equal(t, big.NewInt(1980), maxTopUp)

		softAuctionConfig := als.calcSoftAuctionNodesConfig(ownersData, 3)
		require.Equal(t, ownersData, softAuctionConfig)
		selectedNodes := als.selectNodes(softAuctionConfig, 3, randomness)
		require.Equal(t, []state.ValidatorInfoHandler{v1, v2, v0}, selectedNodes)

		softAuctionConfig = als.calcSoftAuctionNodesConfig(ownersData, 2)
		expectedSoftAuction := copyOwnersData(ownersData)
		expectedSoftAuction[owner2].numQualifiedAuctionNodes = 1
		expectedSoftAuction[owner2].qualifiedTopUpPerNode = big.NewInt(1980)
		require.Equal(t, expectedSoftAuction, softAuctionConfig)
		selectedNodes = als.selectNodes(softAuctionConfig, 2, randomness)
		require.Equal(t, []state.ValidatorInfoHandler{v2, v1}, selectedNodes)

		softAuctionConfig = als.calcSoftAuctionNodesConfig(ownersData, 1)
		delete(expectedSoftAuction, owner1)
		require.Equal(t, expectedSoftAuction, softAuctionConfig)
		selectedNodes = als.selectNodes(softAuctionConfig, 1, randomness)
		require.Equal(t, []state.ValidatorInfoHandler{v2}, selectedNodes)
	})
}

func TestAuctionListSelector_calcSoftAuctionNodesConfig(t *testing.T) {
	t.Parallel()

	randomness := []byte("pk0")
	v1 := &state.ValidatorInfo{PublicKey: []byte("pk1")}
	v2 := &state.ValidatorInfo{PublicKey: []byte("pk2")}
	v3 := &state.ValidatorInfo{PublicKey: []byte("pk3")}
	v4 := &state.ValidatorInfo{PublicKey: []byte("pk4")}
	v5 := &state.ValidatorInfo{PublicKey: []byte("pk5")}
	v6 := &state.ValidatorInfo{PublicKey: []byte("pk6")}
	v7 := &state.ValidatorInfo{PublicKey: []byte("pk7")}
	v8 := &state.ValidatorInfo{PublicKey: []byte("pk8")}

	owner1 := "owner1"
	owner2 := "owner2"
	owner3 := "owner3"
	owner4 := "owner4"
	ownersData := map[string]*ownerAuctionData{
		owner1: {
			numActiveNodes:           2,
			numAuctionNodes:          2,
			numQualifiedAuctionNodes: 2,
			numStakedNodes:           4,
			totalTopUp:               big.NewInt(1500),
			topUpPerNode:             big.NewInt(375),
			qualifiedTopUpPerNode:    big.NewInt(375),
			auctionList:              []state.ValidatorInfoHandler{v1, v2},
		},
		owner2: {
			numActiveNodes:           0,
			numAuctionNodes:          3,
			numQualifiedAuctionNodes: 3,
			numStakedNodes:           3,
			totalTopUp:               big.NewInt(3000),
			topUpPerNode:             big.NewInt(1000),
			qualifiedTopUpPerNode:    big.NewInt(1000),
			auctionList:              []state.ValidatorInfoHandler{v3, v4, v5},
		},
		owner3: {
			numActiveNodes:           1,
			numAuctionNodes:          2,
			numQualifiedAuctionNodes: 2,
			numStakedNodes:           3,
			totalTopUp:               big.NewInt(1000),
			topUpPerNode:             big.NewInt(333),
			qualifiedTopUpPerNode:    big.NewInt(333),
			auctionList:              []state.ValidatorInfoHandler{v6, v7},
		},
		owner4: {
			numActiveNodes:           1,
			numAuctionNodes:          1,
			numQualifiedAuctionNodes: 1,
			numStakedNodes:           2,
			totalTopUp:               big.NewInt(0),
			topUpPerNode:             big.NewInt(0),
			qualifiedTopUpPerNode:    big.NewInt(0),
			auctionList:              []state.ValidatorInfoHandler{v8},
		},
	}

	args := createAuctionListSelectorArgs(nil)
	als, _ := NewAuctionListSelector(args)

	minTopUp, maxTopUp := als.getMinMaxPossibleTopUp(ownersData)
	require.Equal(t, big.NewInt(1), minTopUp)    // owner4 having all nodes in auction
	require.Equal(t, big.NewInt(3000), maxTopUp) // owner2 having only only one node in auction

	softAuctionConfig := als.calcSoftAuctionNodesConfig(ownersData, 9)
	require.Equal(t, ownersData, softAuctionConfig)
	selectedNodes := als.selectNodes(softAuctionConfig, 8, randomness)
	require.Equal(t, []state.ValidatorInfoHandler{v5, v4, v3, v2, v1, v7, v6, v8}, selectedNodes)

	softAuctionConfig = als.calcSoftAuctionNodesConfig(ownersData, 8)
	require.Equal(t, ownersData, softAuctionConfig)
	selectedNodes = als.selectNodes(softAuctionConfig, 8, randomness)
	require.Equal(t, []state.ValidatorInfoHandler{v5, v4, v3, v2, v1, v7, v6, v8}, selectedNodes)

	softAuctionConfig = als.calcSoftAuctionNodesConfig(ownersData, 7)
	expectedConfig := copyOwnersData(ownersData)
	delete(expectedConfig, owner4)
	require.Equal(t, expectedConfig, softAuctionConfig)
	selectedNodes = als.selectNodes(softAuctionConfig, 7, randomness)
	require.Equal(t, []state.ValidatorInfoHandler{v5, v4, v3, v2, v1, v7, v6}, selectedNodes)

	softAuctionConfig = als.calcSoftAuctionNodesConfig(ownersData, 6)
	expectedConfig[owner3].numQualifiedAuctionNodes = 1
	expectedConfig[owner3].qualifiedTopUpPerNode = big.NewInt(500)
	require.Equal(t, expectedConfig, softAuctionConfig)
	selectedNodes = als.selectNodes(softAuctionConfig, 6, randomness)
	require.Equal(t, []state.ValidatorInfoHandler{v5, v4, v3, v7, v2, v1}, selectedNodes)

	softAuctionConfig = als.calcSoftAuctionNodesConfig(ownersData, 5)
	expectedConfig[owner1].numQualifiedAuctionNodes = 1
	expectedConfig[owner1].qualifiedTopUpPerNode = big.NewInt(500)
	require.Equal(t, expectedConfig, softAuctionConfig)
	selectedNodes = als.selectNodes(softAuctionConfig, 5, randomness)
	require.Equal(t, []state.ValidatorInfoHandler{v5, v4, v3, v7, v2}, selectedNodes)

	softAuctionConfig = als.calcSoftAuctionNodesConfig(ownersData, 4)
	require.Equal(t, expectedConfig, softAuctionConfig)
	selectedNodes = als.selectNodes(softAuctionConfig, 4, randomness)
	require.Equal(t, []state.ValidatorInfoHandler{v5, v4, v3, v7}, selectedNodes)

	softAuctionConfig = als.calcSoftAuctionNodesConfig(ownersData, 3)
	delete(expectedConfig, owner3)
	delete(expectedConfig, owner1)
	require.Equal(t, expectedConfig, softAuctionConfig)
	selectedNodes = als.selectNodes(softAuctionConfig, 3, randomness)
	require.Equal(t, []state.ValidatorInfoHandler{v5, v4, v3}, selectedNodes)

	softAuctionConfig = als.calcSoftAuctionNodesConfig(ownersData, 2)
	expectedConfig[owner2].numQualifiedAuctionNodes = 2
	expectedConfig[owner2].qualifiedTopUpPerNode = big.NewInt(1500)
	require.Equal(t, expectedConfig, softAuctionConfig)
	selectedNodes = als.selectNodes(softAuctionConfig, 2, randomness)
	require.Equal(t, []state.ValidatorInfoHandler{v5, v4}, selectedNodes)

	softAuctionConfig = als.calcSoftAuctionNodesConfig(ownersData, 1)
	expectedConfig[owner2].numQualifiedAuctionNodes = 1
	expectedConfig[owner2].qualifiedTopUpPerNode = big.NewInt(3000)
	require.Equal(t, expectedConfig, softAuctionConfig)
	selectedNodes = als.selectNodes(softAuctionConfig, 1, randomness)
	require.Equal(t, []state.ValidatorInfoHandler{v5}, selectedNodes)
}
