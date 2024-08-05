package repos

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
	"holvit/constants"
	"holvit/h"
	"holvit/ioc"
	"holvit/logging"
	"holvit/middlewares"
	"holvit/requestContext"
)

type QueuedJob struct {
	BaseModel

	Status string

	Type    string
	Details QueuedJobDetails

	FailureCount int
	Error        *string
}

type QueuedJobDetails interface {
	Type() string
}

type SendMailJobDetails struct {
	To  []string `json:"to"`
	Cc  []string `json:"cc"`
	Bcc []string `json:"bcc"`

	Subject string `json:"subject"`
	Body    string `json:"body"`
}

func (d SendMailJobDetails) Type() string {
	return constants.QueuedJobSendMail
}

func (d SendMailJobDetails) Value() (driver.Value, error) {
	return json.Marshal(d)
}

func (d *SendMailJobDetails) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, &d)
}

type QueuedJobFilter struct {
	BaseFilter

	IgnoreLocked bool
	Status       h.Optional[string]
}

type QueuedJobUpdate struct {
	Status       h.Optional[string]
	FailureCount h.Optional[int]
	Error        h.Optional[string]
}

type QueuedJobRepository interface {
	FindQueuedJobById(ctx context.Context, id uuid.UUID) h.Optional[QueuedJob]
	FindQueuedJobs(ctx context.Context, filter QueuedJobFilter) h.Result[FilterResult[QueuedJob]]
	CreateQueuedJob(ctx context.Context, job *QueuedJob) h.Result[uuid.UUID]
	UpdateQueuedJob(ctx context.Context, id uuid.UUID, upd QueuedJobUpdate) error
}

type QueuedJobRepositoryImpl struct{}

func NewQueuedJobRepository() QueuedJobRepository {
	return &QueuedJobRepositoryImpl{}
}

func (r *QueuedJobRepositoryImpl) FindQueuedJobById(ctx context.Context, id uuid.UUID) h.Optional[QueuedJob] {
	return r.FindQueuedJobs(ctx, QueuedJobFilter{
		BaseFilter: BaseFilter{
			Id: h.Some(id),
		},
	}).Unwrap().First()

}

func (c *QueuedJobRepositoryImpl) FindQueuedJobs(ctx context.Context, filter QueuedJobFilter) h.Result[FilterResult[QueuedJob]] {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		return h.Err[FilterResult[QueuedJob]](err)
	}

	selectCount := "count(*) over ()"
	if filter.IgnoreLocked {
		selectCount = "-1"
	}

	sb := sqlbuilder.Select(selectCount, "id", "status", "type", "details", "failure_count", "error").
		From("queued_jobs")

	filter.Id.IfSome(func(x uuid.UUID) {
		sb.Where(sb.Equal("id", x))
	})

	filter.Status.IfSome(func(x string) {
		sb.Where(sb.Equal("status", x))
	})

	if filter.PagingInfo.IsSome() {
		sb.Limit(filter.PagingInfo.Unwrap().PageSize).
			Offset(filter.PagingInfo.Unwrap().PageSize * (filter.PagingInfo.Unwrap().PageNumber - 1))
	}

	if filter.IgnoreLocked {
		sb.SQL("for update skip locked")
	}

	sqlString, args := sb.Build()
	logging.Logger.Debugf("executing sql: %s", sqlString)
	rows, err := tx.Query(sqlString, args...)
	if err != nil {
		return h.Err[FilterResult[QueuedJob]](err)
	}
	defer rows.Close()

	var totalCount int
	var result []QueuedJob
	for rows.Next() {
		var row QueuedJob
		var detailsRaw json.RawMessage
		err := rows.Scan(&totalCount,
			&row.Id,
			&row.Status,
			&row.Type,
			&detailsRaw,
			&row.FailureCount,
			&row.Error)
		if err != nil {
			return h.Err[FilterResult[QueuedJob]](err)
		}

		switch row.Type {
		case constants.QueuedJobSendMail:
			var details SendMailJobDetails
			err := json.Unmarshal(detailsRaw, &details)
			if err != nil {
				return h.Err[FilterResult[QueuedJob]](err)
			}
			row.Details = details
			break
		default:
			logging.Logger.Fatalf("Unsupported job type '%v' in queud job '%v'", row.Type, row.Id.String())
		}

		result = append(result, row)
	}

	return h.Ok(pagedResult[QueuedJob]{
		values: result,
		count:  totalCount,
	}.ToResult())
}

func (c *QueuedJobRepositoryImpl) CreateQueuedJob(ctx context.Context, job *QueuedJob) h.Result[uuid.UUID] {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	var resultingId uuid.UUID

	tx, err := rcs.GetTx()
	if err != nil {
		return h.Err[uuid.UUID](err)
	}

	sqlString := `insert into "queued_jobs"
				("status", "type", "details", "failure_count", "error")
				values ($1, $2, $3, $4, $5)
				returning "id"`
	logging.Logger.Debugf("executing sql: %s", sqlString)

	err = tx.QueryRow(sqlString,
		job.Status,
		job.Type,
		job.Details,
		job.FailureCount,
		job.Error).Scan(&resultingId)
	if err != nil {
		return h.Err[uuid.UUID](err)
	}

	return h.Ok(resultingId)
}

func (c *QueuedJobRepositoryImpl) UpdateQueuedJob(ctx context.Context, id uuid.UUID, upd QueuedJobUpdate) error {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		return err
	}

	sb := sqlbuilder.Update("queued_jobs")

	upd.Status.IfSome(func(x string) {
		sb.Set(sb.Assign("status", x))
	})

	upd.Error.IfSome(func(x string) {
		sb.Set(sb.Assign("error", x))
	})

	upd.FailureCount.IfSome(func(x int) {
		sb.Set(sb.Assign("failure_count", x))
	})

	sb.Where(sb.Equal("id", id))

	sqlString, args := sb.Build()
	logging.Logger.Debugf("executing sql: %s", sqlString)
	_, err = tx.Exec(sqlString, args...)
	if err != nil {
		return err
	}

	return nil
}
