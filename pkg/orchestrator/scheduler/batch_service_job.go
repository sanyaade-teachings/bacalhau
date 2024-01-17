package scheduler

import (
	"context"
	"fmt"
	gomath "math"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/lib/math"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/bacalhau-project/bacalhau/pkg/util/idgen"
)

// BatchServiceJobScheduler is a scheduler for:
// - batch jobs that run until completion on N number of nodes
// - service jobs than run until stopped on N number of nodes
type BatchServiceJobScheduler struct {
	jobStore      jobstore.Store
	planner       orchestrator.Planner
	nodeSelector  orchestrator.NodeSelector
	retryStrategy orchestrator.RetryStrategy
}

type BatchServiceJobSchedulerParams struct {
	JobStore      jobstore.Store
	Planner       orchestrator.Planner
	NodeSelector  orchestrator.NodeSelector
	RetryStrategy orchestrator.RetryStrategy
}

func NewBatchServiceJobScheduler(params BatchServiceJobSchedulerParams) *BatchServiceJobScheduler {
	return &BatchServiceJobScheduler{
		jobStore:      params.JobStore,
		planner:       params.Planner,
		nodeSelector:  params.NodeSelector,
		retryStrategy: params.RetryStrategy,
	}
}

func (b *BatchServiceJobScheduler) Process(ctx context.Context, evaluation *models.Evaluation) error {
	ctx = log.Ctx(ctx).With().Str("JobID", evaluation.JobID).Str("EvalID", evaluation.ID).Logger().WithContext(ctx)

	job, err := b.jobStore.GetJob(ctx, evaluation.JobID)
	if err != nil {
		return fmt.Errorf("failed to retrieve job %s: %w", evaluation.JobID, err)
	}
	// Retrieve the job state
	jobExecutions, err := b.jobStore.GetExecutions(ctx, jobstore.GetExecutionsOptions{
		JobID: evaluation.JobID,
	})
	if err != nil {
		return fmt.Errorf("failed to retrieve job state for job %s when evaluating %s: %w",
			evaluation.JobID, evaluation, err)
	}

	// Plan to hold the actions to be taken
	plan := models.NewPlan(evaluation, &job)

	existingExecs := execSetFromSliceOfValues(jobExecutions)
	nonTerminalExecs := existingExecs.filterNonTerminal()

	// early exit if the job is stopped
	if job.IsTerminal() {
		nonTerminalExecs.markStopped(execNotNeeded, plan)
		return b.planner.Process(ctx, plan)
	}

	// Retrieve the info for all the nodes that have executions for this job
	nodeInfos, err := existingNodeInfos(ctx, b.nodeSelector, nonTerminalExecs)
	if err != nil {
		return err
	}

	// Mark executions that are running on nodes that are not healthy as failed
	nonTerminalExecs, lost := nonTerminalExecs.filterByNodeHealth(nodeInfos)
	lost.markStopped(execLost, plan)

	// Calculate remaining job count
	// Service jobs run until the user stops the job, and would be a bug if an execution is marked completed. So the desired
	// remaining count equals the count specified in the job spec.
	// Batch jobs on the other hand run until completion and the desired remaining count excludes the completed executions
	desiredRemainingCount := job.Count
	if job.Type == models.JobTypeBatch {
		desiredRemainingCount = math.Max(0, job.Count-existingExecs.countCompleted())
	}

	// Approve/Reject nodes
	execsByApprovalStatus := nonTerminalExecs.filterByApprovalStatus(desiredRemainingCount)
	execsByApprovalStatus.toApprove.markApproved(plan)
	execsByApprovalStatus.toReject.markStopped(execRejected, plan)

	// How many executions failed due to compute nodes rejecting bids?
	rejectedExecutions := 0

	for _, execution := range jobExecutions {
		if execution.ComputeState.StateType == models.ExecutionStateAskForBidRejected {
			// This execution was rejected by its compute node
			rejectedExecutions = rejectedExecutions + 1
		}
	}

	if (rejectedExecutions > 0) && evaluation.TriggeredBy != models.EvalTriggerDefer {
		// If we had failed executions due to bid rejections in the
		// past, then we should retry. This causes the scheduler to be
		// nivoked again after the retry delay; when that happens,
		// evaluation.TriggeredBy will equal models.EvalTriggerDefer so
		// the test above checks for that to make us not retry *again*
		// after we've retried.
		b.handleRetry(plan, &job, rejectedExecutions)
		return b.planner.Process(ctx, plan)
	} else {
		// create new executions if needed
		remainingExecutionCount := desiredRemainingCount - execsByApprovalStatus.activeCount()
		if remainingExecutionCount > 0 {
			allFailed := existingExecs.filterFailed().union(lost)
			var placementErr error
			if len(allFailed) > 0 && !b.retryStrategy.ShouldRetry(ctx, orchestrator.RetryRequest{Job: &job}) {
				placementErr = fmt.Errorf("exceeded max retries for job %s", job.ID)
				b.handleFailure(nonTerminalExecs, allFailed, plan, placementErr)
				return b.planner.Process(ctx, plan)
			} else {
				_, placementErr = b.createMissingExecs(ctx, remainingExecutionCount, &job, plan)
			}
			if placementErr != nil {
				b.handleFailure(nonTerminalExecs, allFailed, plan, placementErr)
				return b.planner.Process(ctx, plan)
			}
		}
	}

	// stop executions if we over-subscribed and exceeded the desired number of executions
	_, overSubscriptions := execsByApprovalStatus.running.filterByOverSubscriptions(desiredRemainingCount)
	overSubscriptions.markStopped(execNotNeeded, plan)

	// Check the job's state and update it accordingly.
	if desiredRemainingCount <= 0 {
		// If there are no remaining tasks to be done, mark the job as completed.
		plan.MarkJobCompleted()
	}

	plan.MarkJobRunningIfEligible()
	return b.planner.Process(ctx, plan)
}

func (b *BatchServiceJobScheduler) createMissingExecs(
	ctx context.Context, remainingExecutionCount int, job *models.Job, plan *models.Plan) (execSet, error) {
	newExecs := execSet{}
	for i := 0; i < remainingExecutionCount; i++ {
		execution := &models.Execution{
			JobID:        job.ID,
			Job:          job,
			ID:           idgen.ExecutionIDPrefix + uuid.NewString(),
			EvalID:       plan.EvalID,
			Namespace:    job.Namespace,
			ComputeState: models.NewExecutionState(models.ExecutionStateNew),
			DesiredState: models.NewExecutionDesiredState(models.ExecutionDesiredStatePending),
		}
		execution.Normalize()
		newExecs[execution.ID] = execution
	}
	if len(newExecs) > 0 {
		err := b.placeExecs(ctx, newExecs, job)
		if err != nil {
			return newExecs, err
		}
	}
	for _, exec := range newExecs {
		plan.AppendExecution(exec)
	}
	return newExecs, nil
}

// placeExecs places the executions
func (b *BatchServiceJobScheduler) placeExecs(ctx context.Context, execs execSet, job *models.Job) error {
	if len(execs) > 0 {
		selectedNodes, err := b.nodeSelector.TopMatchingNodes(ctx, job, len(execs))
		if err != nil {
			return err
		}
		i := 0
		for _, exec := range execs {
			exec.NodeID = selectedNodes[i].ID()
			i++
		}
	}
	return nil
}

func (b *BatchServiceJobScheduler) handleRetry(plan *models.Plan, job *models.Job, failures int) {
	// delay = RetryDelay * RetryDelayGrowthFactor ^ failures
	// ...but never more than MaximumRetryDelay
	delay := gomath.Min(
		float64(job.RetryDelay)*gomath.Pow(job.RetryDelayGrowthFactor, float64(failures)),
		float64(job.MaximumRetryDelay))

	log.Debug().Msgf("Deferring job execution for %f seconds (%d * %f ^ %d max %d)",
		delay,
		job.RetryDelay, job.RetryDelayGrowthFactor, failures,
		job.MaximumRetryDelay)

	// Schedule a new evaluation
	plan.DeferEvaluation(time.Duration(delay * float64(time.Second)))
}

func (b *BatchServiceJobScheduler) handleFailure(nonTerminalExecs execSet, failed execSet, plan *models.Plan, err error) {
	// mark all non-terminal executions as failed
	nonTerminalExecs.markStopped(jobFailed, plan)

	// mark the job as failed, using the error message of the latest failed execution, if any, or use
	// the error message passed by the scheduler
	latestErr := err.Error()
	if len(failed) > 0 {
		latestErr = failed.latest().ComputeState.Message
	}
	plan.MarkJobFailed(latestErr)
}

// compile-time assertion that BatchServiceJobScheduler satisfies the Scheduler interface
var _ orchestrator.Scheduler = &BatchServiceJobScheduler{}
