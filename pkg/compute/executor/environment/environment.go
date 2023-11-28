package environment

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/lib/provider"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/rs/zerolog/log"
)

type Environment struct {
	inputs  []Mount
	outputs []Mount

	variables map[string]string

	storage *provider.Provider[storage.Storage]

	stdoutFile string
	stderrFile string

	cleanupPolicy *CleanupPolicy
	cleanupFuncs  []func(context.Context) error
}

type CleanupPolicy struct {
	delayTime          *time.Duration
	awaitJobCompletion bool
}

func New(options ...Option) *Environment {
	e := &Environment{}

	for _, opt := range options {
		opt(e)
	}

	return e
}

func (e *Environment) Build(ctx context.Context, execution *models.Execution, rootDirectory string) error {

	task := execution.Job.Task()

	executionRoot := filepath.Join(rootDirectory, execution.JobID, execution.ID)
	executionLogs := filepath.Join(executionRoot, "logs")
	executionInput := filepath.Join(executionRoot, "input")
	executionOutput := filepath.Join(executionRoot, "output")

	if err := e.buildInputs(ctx, task, executionInput); err != nil {
		return err
	}

	if err := e.buildOutputs(ctx, task, executionOutput); err != nil {
		return err
	}

	if err := e.buildLogOutput(ctx, executionLogs); err != nil {
		return err
	}

	return nil
}

func (e *Environment) buildInputs(ctx context.Context, task *models.Task, inputPath string) error {
	if e.storage == nil {
		// TODO: Not this
		return nil
	}

	// We want for the inputs to be written underneath a specific directory, unfortunatelyROSS

	inputVolumes, inputCleanup, err := prepareInputVolumes(ctx, *e.storage, inputPath, task.InputSources...)
	if err != nil {
		return err
	}
	e.cleanupFuncs = append(e.cleanupFuncs, inputCleanup)

	for _, input := range inputVolumes {
		log.Ctx(ctx).Trace().Msgf("Input Volume: %+v %+v", input.InputSource, input.Volume)

		e.inputs = append(e.inputs, Mount{
			Local:    input.Volume.Source,
			Hosted:   input.Volume.Target,
			ReadOnly: input.Volume.ReadOnly,
		})
	}

	return nil
}

func (e *Environment) buildOutputs(ctx context.Context, task *models.Task, executionOutputPath string) error {
	// Generate results folder for the task

	if err := os.MkdirAll(executionOutputPath, os.ModePerm); err != nil {
		return err
	}

	for _, respath := range task.ResultPaths {
		m := Mount{
			Local:  filepath.Join(executionOutputPath, respath.Name),
			Hosted: respath.Path,
		}

		if err := os.MkdirAll(m.Local, os.ModePerm); err != nil {
			return err
		}

		e.outputs = append(e.outputs, m)
	}

	return nil
}

func (e *Environment) buildLogOutput(ctx context.Context, executionLogsPath string) error {

	if err := os.MkdirAll(executionLogsPath, os.ModePerm); err != nil {
		return err
	}

	// Create the execution log files ready for the execution
	e.stdoutFile = filepath.Join(executionLogsPath, "stdout")
	if err := makeExist(e.stdoutFile); err != nil {
		return err
	}

	e.stderrFile = filepath.Join(executionLogsPath, "stderr")
	if err := makeExist(e.stderrFile); err != nil {
		return err
	}

	return nil
}

func makeExist(fileName string) error {
	if _, err := os.Stat(fileName); err != nil {
		if os.IsNotExist(err) {
			file, err := os.Create(fileName)
			if err != nil {
				return err
			}
			defer file.Close()
		}
	}
	return nil
}

func (e *Environment) VariablesAsArray() []string {
	vars := make([]string, len(e.variables))
	if len(e.variables) == 0 {
		return vars
	}

	for k, v := range e.variables {
		// Skip anything that looks like it was generated by
		// bacalhau
		if strings.HasPrefix(k, "BACALHAU_") {
			continue
		}

		vars = append(vars, fmt.Sprintf("%s=%s", k, v))
	}
	return vars
}

func (e *Environment) Destroy() {
}
