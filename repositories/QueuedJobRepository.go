package repositories

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
	"holvit/constants"
	"holvit/httpErrors"
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
	Status       *string
}

type QueuedJobUpdate struct {
	Status       *string
	FailureCount *int
	Error        *string
}

type QueuedJobRepository interface {
	FindQueuedJobById(ctx context.Context, id uuid.UUID) (*QueuedJob, error)
	FindQueuedJobs(ctx context.Context, filter QueuedJobFilter) ([]*QueuedJob, int, error)
	CreateQueuedJob(ctx context.Context, job *QueuedJob) (uuid.UUID, error)
	UpdateQueuedJob(ctx context.Context, id uuid.UUID, upd QueuedJobUpdate) error
}

type QueuedJobRepositoryImpl struct{}

func NewQueuedJobRepository() QueuedJobRepository {
	return &QueuedJobRepositoryImpl{}
}

func (r *QueuedJobRepositoryImpl) FindQueuedJobById(ctx context.Context, id uuid.UUID) (*QueuedJob, error) {
	credentials, resultCount, err := r.FindQueuedJobs(ctx, QueuedJobFilter{
		BaseFilter: BaseFilter{
			Id: id,
			PagingInfo: PagingInfo{
				PageSize:   1,
				PageNumber: 0,
			},
		},
	})

	if err != nil {
		return nil, err
	}
	if resultCount != len(credentials) {
		return nil, httpErrors.NotFound()
	}
	return credentials[0], nil
}

func (c *QueuedJobRepositoryImpl) FindQueuedJobs(ctx context.Context, filter QueuedJobFilter) ([]*QueuedJob, int, error) {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		return nil, 0, err
	}

	selectCount := "count(*) over ()"
	if filter.IgnoreLocked {
		selectCount = "-1"
	}

	sb := sqlbuilder.Select(selectCount, "id", "status", "type", "details", "failure_count", "error").
		From("queued_jobs")

	if filter.Status != nil {
		sb.Where(sb.Equal("status", *filter.Status))
	}

	if filter.PagingInfo.PageSize > 0 {
		sb.Limit(filter.PagingInfo.PageSize).
			Offset(filter.PagingInfo.PageSize * (filter.PagingInfo.PageNumber - 1))
	}

	if filter.IgnoreLocked {
		sb.SQL("for update skip locked")
	}

	sqlString, args := sb.Build()
	logging.Logger.Debugf("executing sql: %s", sqlString)
	rows, err := tx.Query(sqlString, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var totalCount int
	var result []*QueuedJob
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
			return nil, 0, err
		}

		switch row.Type {
		case constants.QueuedJobSendMail:
			var details SendMailJobDetails
			err := json.Unmarshal(detailsRaw, &details)
			if err != nil {
				return nil, 0, err
			}
			row.Details = details
			break
		default:
			logging.Logger.Fatalf("Unsupported job type '%v' in queud job '%v'", row.Type, row.Id.String())
		}

		result = append(result, &row)
	}

	return result, totalCount, nil
}

func (c *QueuedJobRepositoryImpl) CreateQueuedJob(ctx context.Context, job *QueuedJob) (uuid.UUID, error) {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	var resultingId uuid.UUID

	tx, err := rcs.GetTx()
	if err != nil {
		return resultingId, err
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

	return resultingId, err
}

func (c *QueuedJobRepositoryImpl) UpdateQueuedJob(ctx context.Context, id uuid.UUID, upd QueuedJobUpdate) error {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		return err
	}

	sb := sqlbuilder.Update("queued_jobs")

	if upd.Status != nil {
		sb.Set(sb.Assign("status", *upd.Status))
	}

	if upd.Error != nil {
		sb.Set(sb.Assign("error", *upd.Error))
	}

	if upd.FailureCount != nil {
		sb.Set(sb.Assign("failure_count", *upd.FailureCount))
	}

	sb.Where(sb.Equal("id", id))

	sqlString, args := sb.Build()
	logging.Logger.Debugf("executing sql: %s", sqlString)
	_, err = tx.Exec(sqlString, args...)
	if err != nil {
		return err
	}

	return nil
}
