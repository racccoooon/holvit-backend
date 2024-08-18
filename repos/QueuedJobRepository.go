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
	"holvit/sqlb"
	"holvit/utils"
)

type QueuedJob struct {
	BaseModel

	Status string

	Type    string
	Details QueuedJobDetails

	FailureCount int
	Error        h.Opt[string]
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
	Status       h.Opt[string]
}

type QueuedJobUpdate struct {
	Status       h.Opt[string]
	FailureCount h.Opt[int]
	Error        h.Opt[string]
}

type QueuedJobRepository interface {
	FindQueuedJobById(ctx context.Context, id uuid.UUID) h.Opt[QueuedJob]
	FindQueuedJobs(ctx context.Context, filter QueuedJobFilter) FilterResult[QueuedJob]
	CreateQueuedJob(ctx context.Context, job QueuedJob) uuid.UUID
	UpdateQueuedJob(ctx context.Context, id uuid.UUID, upd QueuedJobUpdate)
}

type queuedJobRepositoryImpl struct{}

func NewQueuedJobRepository() QueuedJobRepository {
	return &queuedJobRepositoryImpl{}
}

func (r *queuedJobRepositoryImpl) FindQueuedJobById(ctx context.Context, id uuid.UUID) h.Opt[QueuedJob] {
	return r.FindQueuedJobs(ctx, QueuedJobFilter{
		BaseFilter: BaseFilter{
			Id: h.Some(id),
		},
	}).FirstOrNone()

}

func (c *queuedJobRepositoryImpl) FindQueuedJobs(ctx context.Context, filter QueuedJobFilter) FilterResult[QueuedJob] {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		panic(err)
	}

	selectCount := filter.CountCol()
	if filter.IgnoreLocked {
		selectCount = "-1"
	}

	q := sqlb.Select(selectCount, "id", "status", "type", "details", "failure_count", "error").
		From("queued_jobs")

	filter.Id.IfSome(func(x uuid.UUID) {
		q.Where("id = ?", x)
	})

	filter.Status.IfSome(func(x string) {
		q.Where("status = ?", x)
	})

	filter.PagingInfo.IfSome(func(x PagingInfo) {
		x.Apply(q)
	})

	filter.SortInfo.IfSome(func(x SortInfo) {
		x.Apply(q)
	})

	if filter.IgnoreLocked {
		q.LockForUpdate(true)
	}

	query := q.Build()
	logging.Logger.Debugf("executing sql: %s", query.Sql)
	rows, err := tx.Query(query.Sql, query.Parameters...)
	if err != nil {
		panic(mapCustomErrorCodes(err))
	}
	defer utils.PanicOnErr(rows.Close)

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
			row.Error.AsMutPtr())
		if err != nil {
			panic(mapCustomErrorCodes(err))
		}

		switch row.Type {
		case constants.QueuedJobSendMail:
			row.Details = utils.FromRawMessage[SendMailJobDetails](detailsRaw).Unwrap()
		default:
			logging.Logger.Fatalf("Unsupported job type '%v' in queud job '%v'", row.Type, row.Id.String())
		}

		result = append(result, row)
	}

	return NewPagedResult(result, totalCount)
}

func (c *queuedJobRepositoryImpl) CreateQueuedJob(ctx context.Context, job QueuedJob) uuid.UUID {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	var resultingId uuid.UUID

	tx, err := rcs.GetTx()
	if err != nil {
		panic(err)
	}

	q := sqlb.InsertInto("queued_jobs", "status", "type", "details", "failure_count", "error").
		Values(job.Status,
			job.Type,
			job.Details,
			job.FailureCount,
			job.Error.ToNillablePtr()).
		Returning("id")

	query := q.Build()
	logging.Logger.Debugf("executing sql: %s", query.Sql)
	err = tx.QueryRow(query.Sql, query.Parameters...).Scan(&resultingId)
	if err != nil {
		panic(mapCustomErrorCodes(err))
	}

	return resultingId
}

func (c *queuedJobRepositoryImpl) UpdateQueuedJob(ctx context.Context, id uuid.UUID, upd QueuedJobUpdate) {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		panic(err)
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
		panic(mapCustomErrorCodes(err))
	}
}
