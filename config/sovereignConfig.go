package config

// SovereignConfig holds sovereign config
type SovereignConfig struct {
	ExtendedShardHdrNonceHashStorage StorageConfig
	ExtendedShardHeaderStorage       StorageConfig
	MainChainNotarization            MainChainNotarization    `toml:"MainChainNotarization"`
	OutgoingSubscribedEvents         OutgoingSubscribedEvents `toml:"OutgoingSubscribedEvents"`
	OutGoingBridge                   OutGoingBridge           `toml:"OutGoingBridge"`
}

// OutgoingSubscribedEvents holds config for outgoing subscribed events
type OutgoingSubscribedEvents struct {
	SubscribedEvents                                   []SubscribedEvent `toml:"SubscribedEvents"`
	TimeToWaitForUnconfirmedOutGoingOperationInSeconds uint32            `toml:"TimeToWaitForUnconfirmedOutGoingOperationInSeconds"`
}

// SubscribedEvent holds subscribed events config
type SubscribedEvent struct {
	Identifier string   `toml:"Identifier"`
	Addresses  []string `toml:"Addresses"`
}

// MainChainNotarization defines necessary data to start main chain notarization on a sovereign shard
type MainChainNotarization struct {
	MainChainNotarizationStartRound uint64 `toml:"MainChainNotarizationStartRound"`
}

// OutGoingBridge holds config for grpc client to send outgoing bridge txs
type OutGoingBridge struct {
	GRPCHost string `toml:"GRPCHost"`
	GRPCPort string `toml:"GRPCPort"`
}
