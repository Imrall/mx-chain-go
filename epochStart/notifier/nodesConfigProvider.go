package notifier

import (
	"sort"
	"sync"

	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/epochStart"
	"github.com/ElrondNetwork/elrond-go/process"
)

type nodesConfigProvider struct {
	mutex              sync.Mutex
	currentNodesConfig config.MaxNodesChangeConfig
	allNodesConfigs    []config.MaxNodesChangeConfig
}

// NewNodesConfigProvider returns a new instance of nodesConfigProvider, which provides the current
// config.MaxNodesChangeConfig based on the current epoch
func NewNodesConfigProvider(
	epochNotifier process.EpochNotifier,
	maxNodesEnableConfig []config.MaxNodesChangeConfig,
) (*nodesConfigProvider, error) {
	if check.IfNil(epochNotifier) {
		return nil, epochStart.ErrNilEpochNotifier
	}

	ncp := &nodesConfigProvider{
		allNodesConfigs: make([]config.MaxNodesChangeConfig, len(maxNodesEnableConfig)),
	}
	copy(ncp.allNodesConfigs, maxNodesEnableConfig)
	ncp.sortConfigs()
	epochNotifier.RegisterNotifyHandler(ncp)

	return ncp, nil
}

func (ncp *nodesConfigProvider) sortConfigs() {
	ncp.mutex.Lock()
	defer ncp.mutex.Unlock()

	sort.Slice(ncp.allNodesConfigs, func(i, j int) bool {
		return ncp.allNodesConfigs[i].EpochEnable < ncp.allNodesConfigs[j].EpochEnable
	})
}

func (ncp *nodesConfigProvider) GetAllNodesConfig() []config.MaxNodesChangeConfig {
	ncp.mutex.Lock()
	defer ncp.mutex.Unlock()

	return ncp.allNodesConfigs
}

func (ncp *nodesConfigProvider) GetCurrentNodesConfig() config.MaxNodesChangeConfig {
	ncp.mutex.Lock()
	defer ncp.mutex.Unlock()

	return ncp.currentNodesConfig
}

// EpochConfirmed is called whenever a new epoch is confirmed
func (ncp *nodesConfigProvider) EpochConfirmed(epoch uint32, _ uint64) {
	ncp.mutex.Lock()
	defer ncp.mutex.Unlock()

	for _, maxNodesConfig := range ncp.allNodesConfigs {
		if epoch >= maxNodesConfig.EpochEnable {
			ncp.currentNodesConfig = maxNodesConfig
		}
	}
}

// IsInterfaceNil checks if the underlying pointer is nil
func (ncp *nodesConfigProvider) IsInterfaceNil() bool {
	return ncp == nil
}
