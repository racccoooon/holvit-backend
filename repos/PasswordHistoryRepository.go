package repos

import (
	"context"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"holvit/ioc"
	"holvit/logging"
	"holvit/middlewares"
	"holvit/requestContext"
	"holvit/sqlb"
	"holvit/utils"
	"time"
)

type PasswordHistoryEntry struct {
	BaseModel

	UserId         uuid.UUID
	HashedPassword string
	CreatedAt      time.Time
}

type PasswordHistoryFilter struct {
	UserId uuid.UUID
}

type PasswordHistoryRepository interface {
	GetHistory(ctx context.Context, filter PasswordHistoryFilter) []PasswordHistoryEntry
	CreateEntry(ctx context.Context, entry PasswordHistoryEntry)
	DeleteEntries(ctx context.Context, ids []uuid.UUID)
}

func NewPasswordHistoryRepository() PasswordHistoryRepository {
	return &passwordHistoryRepositoryImpl{}
}

type passwordHistoryRepositoryImpl struct{}

func (p *passwordHistoryRepositoryImpl) DeleteEntries(ctx context.Context, ids []uuid.UUID) {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		panic(err)
	}

	q := sqlb.DeleteFrom("password_history").
		Where("user_id IN (?)", pq.Array(ids))

	query := q.Build()
	logging.Logger.Debugf("executing sql: %s", query.Query)
	_, err = tx.Exec(query.Query, query.Parameters...)
	if err != nil {
		panic(err)
	}
}

func (p *passwordHistoryRepositoryImpl) CreateEntry(ctx context.Context, entry PasswordHistoryEntry) {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		panic(err)
	}

	q := sqlb.InsertInto("password_history", "user_id", "hashed_password", "created_at").
		Values(entry.UserId, entry.HashedPassword, entry.CreatedAt)

	query := q.Build()
	logging.Logger.Debugf("executing sql: %s", query.Query)
	_, err = tx.Exec(query.Query, query.Parameters...)
	if err != nil {
		panic(err)
	}
}

func (p *passwordHistoryRepositoryImpl) GetHistory(ctx context.Context, filter PasswordHistoryFilter) []PasswordHistoryEntry {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		panic(err)
	}

	q := sqlb.Select("id", "user_id", "hashed_password", "created_at").
		From("password_history")

	q.Where("user_id = ?", filter.UserId)

	query := q.Build()
	logging.Logger.Debugf("executing sql: %s", query.Query)
	rows, err := tx.Query(query.Query, query.Parameters...)
	if err != nil {
		panic(err)
	}
	defer utils.PanicOnErr(rows.Close)

	var result []PasswordHistoryEntry
	for rows.Next() {
		var row PasswordHistoryEntry
		err := rows.Scan(&row.Id,
			&row.UserId,
			&row.HashedPassword,
			&row.CreatedAt)
		if err != nil {
			panic(err)
		}
		result = append(result, row)
	}

	return result
}
