package factory

import (
	"os"
	"time"

	indexerFactory "github.com/ElrondNetwork/elastic-indexer-go/process/factory"
	wsDriverFactory "github.com/ElrondNetwork/elrond-go-core/websocketOutportDriver/factory"
	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/outport"
	"github.com/ElrondNetwork/elrond-go/outport/firehose"
)

// WrappedOutportDriverWebSocketSenderFactoryArgs extends the wsDriverFactory.OutportDriverWebSocketSenderFactoryArgs structure with the Enabled field
type WrappedOutportDriverWebSocketSenderFactoryArgs struct {
	Enabled bool
	wsDriverFactory.OutportDriverWebSocketSenderFactoryArgs
}

// OutportFactoryArgs holds the factory arguments of different outport drivers
type OutportFactoryArgs struct {
	RetrialInterval                  time.Duration
	ElasticIndexerFactoryArgs        indexerFactory.ArgsIndexerFactory
	EventNotifierFactoryArgs         *EventNotifierFactoryArgs
	WebSocketSenderDriverFactoryArgs WrappedOutportDriverWebSocketSenderFactoryArgs
	FireHoseIndexerConfig            config.FireHoseConfig
}

// CreateOutport will create a new instance of OutportHandler
func CreateOutport(args *OutportFactoryArgs) (outport.OutportHandler, error) {
	err := checkArguments(args)
	if err != nil {
		return nil, err
	}

	outportHandler, err := outport.NewOutport(args.RetrialInterval)
	if err != nil {
		return nil, err
	}

	err = createAndSubscribeDrivers(outportHandler, args)
	if err != nil {
		return nil, err
	}

	return outportHandler, nil
}

func createAndSubscribeDrivers(outport outport.OutportHandler, args *OutportFactoryArgs) error {
	err := createAndSubscribeElasticDriverIfNeeded(outport, args.ElasticIndexerFactoryArgs)
	if err != nil {
		return err
	}

	err = createAndSubscribeEventNotifierIfNeeded(outport, args.EventNotifierFactoryArgs)
	if err != nil {
		return err
	}

	err = createAndSubscribeWebSocketDriver(outport, args.WebSocketSenderDriverFactoryArgs)
	if err != nil {
		return err
	}

	return createAndSubscribeFirehoseIndexerDriver(outport, args.FireHoseIndexerConfig)
}

func createAndSubscribeElasticDriverIfNeeded(
	outport outport.OutportHandler,
	args indexerFactory.ArgsIndexerFactory,
) error {
	if !args.Enabled {
		return nil
	}

	elasticDriver, err := indexerFactory.NewIndexer(args)
	if err != nil {
		return err
	}

	return outport.SubscribeDriver(elasticDriver)
}

func createAndSubscribeEventNotifierIfNeeded(
	outport outport.OutportHandler,
	args *EventNotifierFactoryArgs,
) error {
	if !args.Enabled {
		return nil
	}

	eventNotifier, err := CreateEventNotifier(args)
	if err != nil {
		return err
	}

	return outport.SubscribeDriver(eventNotifier)
}

func checkArguments(args *OutportFactoryArgs) error {
	if args == nil {
		return outport.ErrNilArgsOutportFactory
	}

	return nil
}

func createAndSubscribeWebSocketDriver(
	outport outport.OutportHandler,
	args WrappedOutportDriverWebSocketSenderFactoryArgs,
) error {
	if !args.Enabled {
		return nil
	}

	wsFactory, err := wsDriverFactory.NewOutportDriverWebSocketSenderFactory(args.OutportDriverWebSocketSenderFactoryArgs)
	if err != nil {
		return err
	}

	wsDriver, err := wsFactory.Create()
	if err != nil {
		return err
	}

	return outport.SubscribeDriver(wsDriver)
}

func createAndSubscribeFirehoseIndexerDriver(
	outport outport.OutportHandler,
	args config.FireHoseConfig,
) error {
	if !args.Enabled {
		return nil
	}

	fireHoseIndexer, err := firehose.NewFirehoseIndexer(os.Stdout)
	if err != nil {
		return err
	}

	return outport.SubscribeDriver(fireHoseIndexer)
}
