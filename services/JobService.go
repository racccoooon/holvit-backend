package services

import (
	"context"
	"github.com/robfig/cron/v3"
	"holvit/config"
	"holvit/constants"
	"holvit/ioc"
	"holvit/logging"
	"holvit/middlewares"
	"holvit/repositories"
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
	Execute(ctx context.Context, details repositories.QueuedJobDetails) error
}

type JobService interface {
	QueueJob(ctx context.Context, job repositories.QueuedJobDetails) error
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
	defer scope.Close()
	ctx := middlewares.ContextWithNewScope(context.Background(), scope)

	queuedJobRepository := ioc.Get[repositories.QueuedJobRepository](scope)
	queuedJobs, _, err := queuedJobRepository.FindQueuedJobs(ctx, repositories.QueuedJobFilter{
		IgnoreLocked: true,
		Status:       utils.Ptr("pending"),
	})
	if err != nil {
		//TODO: sentry or something?
		logging.Logger.Error(err)
	}

	for _, job := range queuedJobs {
		err = executors[job.Type].Execute(ctx, job.Details)

		if err != nil {
			upd := repositories.QueuedJobUpdate{
				Error:        utils.Ptr(err.Error()),
				FailureCount: utils.Ptr(job.FailureCount + 1),
				Status:       utils.Ptr("pending"),
			}

			if job.FailureCount == 3 { //TODO: maybe configurable
				upd.Status = utils.Ptr("failed")
			}

			err = queuedJobRepository.UpdateQueuedJob(ctx, job.Id, upd)
			if err != nil {
				logging.Logger.Error(err)
				continue
			}
		} else {
			err := queuedJobRepository.UpdateQueuedJob(ctx, job.Id, repositories.QueuedJobUpdate{
				Status: utils.Ptr("completed"),
			})
			if err != nil {
				logging.Logger.Error(err)
				continue
			}
		}
	}

}

type JobServiceImpl struct {
}

func (s *JobServiceImpl) QueueJob(ctx context.Context, job repositories.QueuedJobDetails) error {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	queuedJobRepository := ioc.Get[repositories.QueuedJobRepository](scope)
	_, err := queuedJobRepository.CreateQueuedJob(ctx, &repositories.QueuedJob{
		Status:       "pending",
		Type:         job.Type(),
		Details:      job,
		FailureCount: 0,
		Error:        nil,
	})
	if err != nil {
		return err
	}

	rcs.OnAfterTx(func(args requestContext.AfterTxEventArgs) {
		if args.Commit {
			executeQueuedJobs()
		}
	})

	return nil
}
