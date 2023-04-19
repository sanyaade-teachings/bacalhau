package spec

import (
	"bytes"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/model"
)

// TODO these are duplicated across the docker executor package and here to avoid dep hell, need a better solution.
const (
	DockerEngineType          = 2
	DockerEngineImageKey      = "Image"
	DockerEngineEntrypointKey = "Entrypoint"
	DockerEngineWorkDirKey    = "WorkingDirectory"
	DockerEngineEnvVarKey     = "EnvironmentVariables"
)

// JobSpecDocker is for VM style executors.
type JobSpecDocker struct {
	// Image is the docker image to run. This must be pull-able by docker.
	Image string `json:"Image,omitempty"`

	// Entrypoint is an optional override for the default container entrypoint.
	Entrypoint []string `json:"Entrypoint,omitempty"`

	// EnvironmentVariables is a map of env to run the container with.
	EnvironmentVariables []string `json:"EnvironmentVariables,omitempty"`

	// WorkingDirectory is the working directory inside the container.
	WorkingDirectory string `json:"WorkingDirectory,omitempty"`
}

func (ds *JobSpecDocker) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := ds.MarshalCBOR(buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func DecodeJobSpecDocker(b []byte) (*JobSpecDocker, error) {
	var spec JobSpecDocker
	if err := spec.UnmarshalCBOR(bytes.NewReader(b)); err != nil {
		return nil, err
	}
	return &spec, nil
}

func (ds *JobSpecDocker) AsEngineSpec() model.EngineSpec {
	data, err := ds.Serialize()
	if err != nil {
		// TODO probs should return this to the caller.
		panic(err)
	}
	return model.EngineSpec{
		Type: DockerEngineType,
		Spec: data,
	}
}

func WithImage(image string) func(*JobSpecDocker) error {
	return func(docker *JobSpecDocker) error {
		docker.Image = image
		return nil
	}
}

func WithEntrypoint(entrypoint ...string) func(*JobSpecDocker) error {
	return func(docker *JobSpecDocker) error {
		docker.Entrypoint = entrypoint
		return nil
	}
}

func MutateDockerEngineSpec(e model.EngineSpec, mutate ...func(docker *JobSpecDocker) error) (model.EngineSpec, error) {
	dockerSpec, err := AsJobSpecDocker(e)
	if err != nil {
		return model.EngineSpec{}, err
	}

	for _, m := range mutate {
		if err := m(dockerSpec); err != nil {
			return model.EngineSpec{}, err
		}
	}
	return dockerSpec.AsEngineSpec(), nil
}

func AsJobSpecDocker(e model.EngineSpec) (*JobSpecDocker, error) {
	if e.Type != DockerEngineType {
		return nil, fmt.Errorf("EngineSpec is Type %s, expected %d", e.Type, DockerEngineType)
	}

	if e.Spec == nil {
		return nil, fmt.Errorf("EngineSpec is uninitalized")
	}

	return DecodeJobSpecDocker(e.Spec)
}
