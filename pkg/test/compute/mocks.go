package compute

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/models/requests"
	"github.com/bacalhau-project/bacalhau/pkg/node/heartbeat"
)

type ManagementEndpointMock struct {
	RegisterHandler        func(ctx context.Context, request requests.RegisterRequest) (*requests.RegisterResponse, error)
	UpdateInfoHandler      func(ctx context.Context, request requests.UpdateInfoRequest) (*requests.UpdateInfoResponse, error)
	UpdateResourcesHandler func(ctx context.Context, request requests.UpdateResourcesRequest) (*requests.UpdateResourcesResponse, error)
}

func (m ManagementEndpointMock) Register(ctx context.Context, request requests.RegisterRequest) (*requests.RegisterResponse, error) {
	if m.RegisterHandler != nil {
		return m.RegisterHandler(ctx, request)
	}
	return &requests.RegisterResponse{Accepted: true}, nil
}

func (m ManagementEndpointMock) UpdateInfo(ctx context.Context, request requests.UpdateInfoRequest) (*requests.UpdateInfoResponse, error) {
	if m.UpdateInfoHandler != nil {
		return m.UpdateInfoHandler(ctx, request)
	}
	return &requests.UpdateInfoResponse{Accepted: true}, nil
}

func (m ManagementEndpointMock) UpdateResources(
	ctx context.Context, request requests.UpdateResourcesRequest) (*requests.UpdateResourcesResponse, error) {
	if m.UpdateResourcesHandler != nil {
		return m.UpdateResourcesHandler(ctx, request)
	}
	return &requests.UpdateResourcesResponse{}, nil
}

// compile time check if ManagementEndpointMock implements ManagementEndpoint
var _ compute.ManagementEndpoint = ManagementEndpointMock{}

// HeartbeatClientMock is a mock implementation of the HeartbeatClient interface
type HeartbeatClientMock struct {
	SendHeartbeatHandler func(ctx context.Context, sequence uint64) error
}

func (h HeartbeatClientMock) SendHeartbeat(ctx context.Context, sequence uint64) error {
	if h.SendHeartbeatHandler != nil {
		return h.SendHeartbeatHandler(ctx, sequence)
	}
	return nil
}

func (h HeartbeatClientMock) Close(ctx context.Context) error {
	return nil
}

// compile time check if HeartbeatClientMock implements HeartbeatClient
var _ heartbeat.Client = HeartbeatClientMock{}
