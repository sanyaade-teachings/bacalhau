//go:generate mockgen --source types.go --destination mocks.go --package jobstore
package jobstore

import (
	"context"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"k8s.io/apimachinery/pkg/labels"
)

type JobQuery struct {
	Namespace string

	// IncludeTags and ExcludeTags are used primarily by the requester's list API.
	// In the orchestrator API, we insted use the Selector field to filter jobs.
	IncludeTags []string
	ExcludeTags []string
	Limit       uint32
	Offset      uint32
	ReturnAll   bool
	SortBy      string
	SortReverse bool
	Selector    labels.Selector
}

type JobQueryResponse struct {
	Jobs       []models.Job
	Offset     uint32 // Offset into the filtered results of the first returned record
	Limit      uint32 // The number of records to return, 0 means all
	NextOffset uint32 // Offset + Limit of the next page of results, 0 means no more results
}

// A Store will persist jobs and their state to the underlying storage.
// It also gives an efficient way to retrieve jobs using queries.
type Store interface {
	// Watch returns a channel from which the caller can read specific events
	// as they are transmitted. When called the combination of parameters
	// will determine which events are sent.  Both the StoreWatcherType and
	// StoreEventType parameters can be a bitmask of entries, so to listen
	// for Create and Delete events for Jobs and Executions you would set
	//   types = JobWatcher | ExecutionWatcher
	//   events = CreateEvent | DeleteEvent
	//
	// The structure sent down the channel when one of these events occurs
	// will contain a timestamp, but also the StoreWatcherType and
	// StoreEventType that triggered the event. A json encoded `[]byte`
	// of the related object will also be included in the [WatchEvent].
	Watch(ctx context.Context, types StoreWatcherType, events StoreEventType) chan WatchEvent

	// GetJob returns a job, identified by the id parameter, or an error if
	// it does not exist.
	GetJob(ctx context.Context, id string) (models.Job, error)

	// GetJobs retrieves a slice of jobs defined by the contents of the
	// [JobQuery]. If it fails, it will return an error
	GetJobs(ctx context.Context, query JobQuery) (*JobQueryResponse, error)

	// GetInProgressJobs retrieves all jobs that have a state that can be
	// considered, 'in progress'. Failure generates an error.
	GetInProgressJobs(ctx context.Context) ([]models.Job, error)

	// GetJobHistory retrieves the history for the specified job.  The
	// history returned is filtered by the contents of the provided
	// [JobHistoryFilterOptions].
	GetJobHistory(ctx context.Context, jobID string, options JobHistoryFilterOptions) ([]models.JobHistory, error)

	// CreateJob will create a new job and persist it in the store.
	CreateJob(ctx context.Context, j models.Job) error

	// GetExecutions retrieves all executions for the specified job.
	GetExecutions(ctx context.Context, options GetExecutionsOptions) ([]models.Execution, error)

	// UpdateJobState updates the state for the job identified in the
	// [UpdateJobStateRequest].
	UpdateJobState(ctx context.Context, request UpdateJobStateRequest) error

	// CreateExecution creates a new execution
	CreateExecution(ctx context.Context, execution models.Execution) error

	// UpdateExecution updates the execution state according to the values
	// within [UpdateExecutionRequest].
	UpdateExecution(ctx context.Context, request UpdateExecutionRequest) error

	// Record that further scheduling of executions for a job has been deferred until the specified time
	RecordJobDeferral(ctx context.Context, jobID string, waitUntil time.Time, comment string) error

	// DeleteJob removes all trace of the provided job from storage
	DeleteJob(ctx context.Context, jobID string) error

	// CreateEvaluation creates a new evaluation
	CreateEvaluation(ctx context.Context, eval models.Evaluation) error

	// GetEvaluation retrieves the specified evaluation
	GetEvaluation(ctx context.Context, id string) (models.Evaluation, error)

	// DeleteEvaluation deletes the specified evaluation
	DeleteEvaluation(ctx context.Context, id string) error

	// Close provides an interface to cleanup any resources in use when the
	// store is no longer required
	Close(ctx context.Context) error
}

type UpdateJobStateRequest struct {
	JobID     string
	Condition UpdateJobCondition
	NewState  models.JobStateType
	Comment   string
}

type UpdateExecutionRequest struct {
	ExecutionID string
	Condition   UpdateExecutionCondition
	NewValues   models.Execution
	Comment     string
}

type UpdateJobCondition struct {
	ExpectedState    models.JobStateType
	UnexpectedStates []models.JobStateType
	ExpectedRevision uint64
}

// Validate checks if the condition matches the given job
func (condition UpdateJobCondition) Validate(job models.Job) error {
	if !condition.ExpectedState.IsUndefined() && condition.ExpectedState != job.State.StateType {
		return NewErrInvalidJobState(job.ID, job.State.StateType, condition.ExpectedState)
	}
	if condition.ExpectedRevision != 0 && condition.ExpectedRevision != job.Revision {
		return NewErrInvalidJobVersion(job.ID, job.Revision, condition.ExpectedRevision)
	}
	if len(condition.UnexpectedStates) > 0 {
		for _, s := range condition.UnexpectedStates {
			if s == job.State.StateType {
				return NewErrInvalidJobState(job.ID, job.State.StateType, models.JobStateTypeUndefined)
			}
		}
	}
	return nil
}

type UpdateExecutionCondition struct {
	ExpectedStates   []models.ExecutionStateType
	ExpectedRevision uint64
	UnexpectedStates []models.ExecutionStateType
}

// Validate checks if the condition matches the given execution
func (condition UpdateExecutionCondition) Validate(execution models.Execution) error {
	if len(condition.ExpectedStates) > 0 {
		validState := false
		for _, s := range condition.ExpectedStates {
			if s == execution.ComputeState.StateType {
				validState = true
				break
			}
		}
		if !validState {
			return NewErrInvalidExecutionState(execution.ID, execution.ComputeState.StateType, condition.ExpectedStates...)
		}
	}

	if condition.ExpectedRevision != 0 && condition.ExpectedRevision != execution.Revision {
		return NewErrInvalidExecutionVersion(execution.ID, execution.Revision, condition.ExpectedRevision)
	}
	if len(condition.UnexpectedStates) > 0 {
		for _, s := range condition.UnexpectedStates {
			if s == execution.ComputeState.StateType {
				return NewErrInvalidExecutionState(execution.ID, execution.ComputeState.StateType)
			}
		}
	}
	return nil
}

type JobHistoryFilterOptions struct {
	Since                 int64  `json:"since"`
	ExcludeExecutionLevel bool   `json:"exclude_execution_level"`
	ExcludeJobLevel       bool   `json:"exclude_job_level"`
	ExecutionID           string `json:"execution_id"`
	NodeID                string `json:"node_id"`
}

type GetExecutionsOptions struct {
	JobID      string `json:"job_id"`
	IncludeJob bool   `json:"include_job"`
}
