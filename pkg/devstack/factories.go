package devstack

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/executor"
	noop_executor "github.com/bacalhau-project/bacalhau/pkg/executor/noop"
	executor_util "github.com/bacalhau-project/bacalhau/pkg/executor/util"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
	noop_publisher "github.com/bacalhau-project/bacalhau/pkg/publisher/noop"
	publisher_util "github.com/bacalhau-project/bacalhau/pkg/publisher/util"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	noop_storage "github.com/bacalhau-project/bacalhau/pkg/storage/noop"
)

// Noop implementations of node factories used to mock certain components, which is useful for testing.
func NewNoopNodeDependencyInjector() node.NodeDependencyInjector {
	return node.NodeDependencyInjector{
		StorageProvidersFactory: NewNoopStorageProvidersFactory(),
		ExecutorsFactory:        NewNoopExecutorsFactory(),
		PublishersFactory:       NewNoopPublishersFactory(),
	}
}

func NewNoopStorageProvidersFactory() node.StorageProvidersFactory {
	return NewNoopStorageProvidersFactoryWithConfig(noop_storage.StorageConfig{})
}

func NewNoopStorageProvidersFactoryWithConfig(config noop_storage.StorageConfig) node.StorageProvidersFactory {
	return node.StorageProvidersFactoryFunc(
		func(ctx context.Context, nodeConfig node.NodeConfig) (storage.StorageProvider, error) {
			return executor_util.NewNoopStorageProvider(ctx, nodeConfig.CleanupManager, config)
		})
}

func NewNoopExecutorsFactory() node.ExecutorsFactory {
	return NewNoopExecutorsFactoryWithConfig(noop_executor.ExecutorConfig{})
}

func NewNoopExecutorsFactoryWithConfig(config noop_executor.ExecutorConfig) node.ExecutorsFactory {
	return node.ExecutorsFactoryFunc(
		func(ctx context.Context, nodeConfig node.NodeConfig) (executor.ExecProvider, error) {
			return executor_util.NewNoopExecutors(config), nil
		})
}

func NewNoopPublishersFactory() node.PublishersFactory {
	return NewNoopPublishersFactoryWithConfig(noop_publisher.PublisherConfig{})
}

func NewNoopPublishersFactoryWithConfig(config noop_publisher.PublisherConfig) node.PublishersFactory {
	return node.PublishersFactoryFunc(
		func(ctx context.Context, nodeConfig node.NodeConfig) (publisher.PublisherProvider, error) {
			return publisher_util.NewNoopPublishers(ctx, nodeConfig.CleanupManager, config)
		})
}
