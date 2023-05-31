package executor

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	_ "github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/test/scenario"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
)

const testNodeCount = 1

func RunTestCase(
	t *testing.T,
	testCase scenario.Scenario,
) {
	ctx := context.Background()
	testSpec := testCase.Spec

	stack, _ := testutils.SetupTest(ctx, t, testNodeCount, 0, false,
		node.NewComputeConfigWithDefaults(),
		node.NewRequesterConfigWithDefaults(),
	)
	executor, err := stack.Nodes[0].ComputeNode.Executors.Get(ctx, testSpec.Engine.Schema)
	require.NoError(t, err)

	isInstalled, err := executor.IsInstalled(ctx)
	require.NoError(t, err)
	require.True(t, isInstalled)

	prepareStorage := func(getStorage scenario.SetupStorage) []spec.Storage {
		if getStorage == nil {
			return []spec.Storage{}
		}

		storageList, stErr := getStorage(ctx, stack.IPFSClients()[:testNodeCount]...)
		require.NoError(t, stErr)

		for _, storageSpec := range storageList {
			hasStorage, stErr := executor.HasStorageLocally(
				ctx, storageSpec)
			require.NoError(t, stErr)
			require.True(t, hasStorage)
		}

		return storageList
	}

	testSpec.Inputs = prepareStorage(testCase.Inputs)
	testSpec.Outputs = testCase.Outputs
	testSpec.Deal = model.Deal{Concurrency: testNodeCount}

	job := model.Job{
		Metadata: model.Metadata{
			ID:        "test-job",
			ClientID:  "test-client",
			CreatedAt: time.Now(),
			Requester: model.JobRequester{
				RequesterNodeID: "test-owner",
			},
		},
		Spec: testSpec,
	}

	resultsDirectory := t.TempDir()
	runnerOutput, err := executor.Run(ctx, "test-execution", job, resultsDirectory)
	require.NoError(t, err)
	require.Empty(t, runnerOutput.ErrorMsg)

	err = testCase.ResultsChecker(resultsDirectory)
	require.NoError(t, err)
}
