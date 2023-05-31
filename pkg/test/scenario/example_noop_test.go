package scenario

import (
	"context"
	"strings"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/executor/noop"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	enginetesting "github.com/bacalhau-project/bacalhau/pkg/model/spec/engine/testing"

	"github.com/stretchr/testify/suite"
)

func noopScenario(t testing.TB) Scenario {
	return Scenario{
		Stack: &StackConfig{
			ExecutorConfig: noop.ExecutorConfig{
				ExternalHooks: noop.ExecutorConfigExternalHooks{
					JobHandler: func(ctx context.Context, job model.Job, resultsDir string) (*model.RunCommandResult, error) {
						return executor.WriteJobResults(resultsDir, strings.NewReader("hello, world!\n"), nil, 0, nil)
					},
				},
			},
		},
		Spec: model.Spec{
			Engine: enginetesting.NoopMakeEngine(t, "noop"),
		},
		ResultsChecker: FileEquals(model.DownloadFilenameStdout, "hello, world!\n"),
		JobCheckers:    WaitUntilSuccessful(1),
	}
}

type NoopTest struct {
	ScenarioRunner
}

func Example_noop() {
	// In a real example, use the testing.T passed to the TestXxx method.
	suite.Run(&testing.T{}, new(NoopTest))
}

func (suite *NoopTest) TestRunNoop() {
	suite.RunScenario(noopScenario(suite.T()))
}
