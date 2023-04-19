//go:build unit || !integration

package model

import (
	"testing"

	"github.com/ipld/go-ipld-prime/codec/json"
	"github.com/stretchr/testify/require"
)

func TestUnmarshalWasm(t *testing.T) {
	bytes, err := tests.ReadFile("tasks/wasm_task.json")
	require.NoError(t, err)

	task, err := UnmarshalIPLD[Task](bytes, json.Decode, UCANTaskSchema)
	require.NoError(t, err)

	spec, err := task.ToSpec()
	require.NoError(t, err)

	engine, err := AsJobSpecWasm(spec.EngineSpec)
	require.NoError(t, err)

	require.Equal(t, EngineWasm, spec.EngineSpec.Type)
	require.Equal(t, "_start", engine.EntryPoint)
	require.Equal(t, []string{"/inputs/data.tar.gz"}, engine.Parameters)
	require.Equal(t, []KV{{
		Key:   "HELLO",
		Value: "world",
	}}, engine.EnvironmentVariables)
	require.Equal(t, []StorageSpec{
		{Path: "/job", StorageSource: StorageSourceIPFS, CID: "bafybeig7mdkzcgpacpozamv7yhhaelztfrnb6ozsupqqh7e5uyqdkijegi"},
		{Path: "/inputs", StorageSource: StorageSourceURLDownload, URL: "https://www.example.com/data.tar.gz"},
	}, spec.Inputs)
	require.Equal(t, []StorageSpec{
		{Path: "/outputs", Name: "outputs"},
	}, spec.Outputs)
}
