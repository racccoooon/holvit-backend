package services

import (
	"context"
	"github.com/robfig/cron/v3"
	"holvit/config"
	"holvit/constants"
	"holvit/h"
	"holvit/ioc"
	"holvit/logging"
	"holvit/middlewares"
	"holvit/repos"
	"holvit/requestContext"
	"holvit/services/jobs"
	"holvit/utils"
)

var (
	executors = map[string]JobExecutor{
		constants.QueuedJobSendMail: &jobs.SendMailExecutor{},
	}
)

type JobExecutor interface {
	Execute(ctx context.Context, details repos.QueuedJobDetails) h.Result[h.Unit]
}

type JobService interface {
	QueueJob(ctx context.Context, job repos.QueuedJobDetails)
}

func NewJobService(c *cron.Cron) JobService {
	_, err := c.AddFunc(config.C.Crons.JobScheduler, executeQueuedJobs)
	if err != nil {
		logging.Logger.Fatal(err)
	}

	return &JobServiceImpl{}
}

func executeQueuedJobs() {
	logging.Logger.Debug("Scheduler is executing queued jobs")

	scope := ioc.RootScope.NewScope()
	defer utils.PanicOnErr(scope.Close)
	ctx := middlewares.ContextWithNewScope(context.Background(), scope)

	queuedJobRepository := ioc.Get[repos.QueuedJobRepository](scope)
	queuedJobs := queuedJobRepository.FindQueuedJobs(ctx, repos.QueuedJobFilter{
		IgnoreLocked: true,
		Status:       h.Some("pending"),
	})

	for _, job := range queuedJobs.Values() {
		result := executors[job.Type].Execute(ctx, job.Details)

		if result.IsErr() {
			upd := repos.QueuedJobUpdate{
				Error:        h.Some(result.UnwrapErr().Error()),
				FailureCount: h.Some(job.FailureCount + 1),
				Status:       h.Some("pending"),
			}

			if job.FailureCount == 3 { //TODO: maybe configurable
				upd.Status = h.Some("failed")
			}

			queuedJobRepository.UpdateQueuedJob(ctx, job.Id, upd)
		} else {
			queuedJobRepository.UpdateQueuedJob(ctx, job.Id, repos.QueuedJobUpdate{
				Status: h.Some("completed"),
			})
		}
	}
}

type JobServiceImpl struct {
}

func (s *JobServiceImpl) QueueJob(ctx context.Context, job repos.QueuedJobDetails) {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	queuedJobRepository := ioc.Get[repos.QueuedJobRepository](scope)
	queuedJobRepository.CreateQueuedJob(ctx, repos.QueuedJob{
		Status:       "pending",
		Type:         job.Type(),
		Details:      job,
		FailureCount: 0,
		Error:        h.None[string](),
	})

	rcs.OnAfterTx(func(args requestContext.AfterTxEventArgs) {
		if args.Commit {
			executeQueuedJobs()
		}
	})
}
