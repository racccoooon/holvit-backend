package repos

import (
	"context"
	"github.com/google/uuid"
	"holvit/h"
	"holvit/ioc"
	"holvit/logging"
	"holvit/middlewares"
	"holvit/requestContext"
	"holvit/sqlb"
	"holvit/utils"
	"time"
)

type Session struct {
	BaseModel

	UserId       uuid.UUID
	UserDeviceId uuid.UUID
	RealmId      uuid.UUID

	ValidUntil  time.Time
	HashedToken string
}

type SessionFilter struct {
	BaseFilter

	RealmId h.Opt[uuid.UUID]
	UserId  h.Opt[uuid.UUID]

	HashedToken h.Opt[string]
}

type SessionRepository interface {
	FindSessionById(ctx context.Context, id uuid.UUID) h.Opt[Session]
	FindSessions(ctx context.Context, filter SessionFilter) FilterResult[Session]
	CreateSession(ctx context.Context, session Session) uuid.UUID
	DeleteOldSessions(ctx context.Context)
}

type sessionRepositoryImpl struct{}

func NewSessionRepository() SessionRepository {
	return &sessionRepositoryImpl{}
}

func (s *sessionRepositoryImpl) DeleteOldSessions(ctx context.Context) {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		panic(err)
	}

	clockService := ioc.Get[utils.ClockService](scope)
	now := clockService.Now()

	q := sqlb.DeleteFrom("sessions").Where("valid_until < ", now)

	query := q.Build()
	logging.Logger.Debugf("executing sql: %s", query.Sql)
	_, err = tx.Exec(query.Sql, query.Parameters...)
	if err != nil {
		panic(mapCustomErrorCodes(err))
	}
}

func (s *sessionRepositoryImpl) FindSessionById(ctx context.Context, id uuid.UUID) h.Opt[Session] {
	return s.FindSessions(ctx, SessionFilter{
		BaseFilter: BaseFilter{
			Id: h.Some(id),
		},
	}).FirstOrNone()
}

func (s *sessionRepositoryImpl) FindSessions(ctx context.Context, filter SessionFilter) FilterResult[Session] {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		panic(err)
	}

	q := sqlb.Select(filter.CountCol(),
		"id", "user_id", "user_device_id", "realm_id", "hashed_token", "valid_until").
		From("sessions")

	filter.RealmId.IfSome(func(x uuid.UUID) {
		q.Where("realm_id = ?", x)
	})

	filter.UserId.IfSome(func(x uuid.UUID) {
		q.Where("user_id = ?", x)
	})

	filter.PagingInfo.IfSome(func(x PagingInfo) {
		x.Apply(q)
	})

	filter.SortInfo.IfSome(func(x SortInfo) {
		x.Apply(q)
	})

	query := q.Build()
	logging.Logger.Debugf("executing sql: %s", query.Sql)
	rows, err := tx.Query(query.Sql, query.Parameters...)
	if err != nil {
		panic(mapCustomErrorCodes(err))
	}
	defer utils.PanicOnErr(rows.Close)

	var totalCount int
	var result []Session
	for rows.Next() {
		var row Session
		err := rows.Scan(&totalCount,
			&row.Id,
			&row.UserId,
			&row.UserDeviceId,
			&row.RealmId,
			&row.HashedToken,
			&row.ValidUntil)
		if err != nil {
			panic(mapCustomErrorCodes(err))
		}
		result = append(result, row)
	}

	return NewPagedResult(result, totalCount)
}

func (s *sessionRepositoryImpl) CreateSession(ctx context.Context, session Session) uuid.UUID {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	var resultingId uuid.UUID

	tx, err := rcs.GetTx()
	if err != nil {
		panic(err)
	}

	q := sqlb.InsertInto("sessions", "user_id", "user_device_id", "realm_id", "hashed_token", "valid_until").
		Values(session.UserId,
			session.UserDeviceId,
			session.RealmId,
			session.HashedToken,
			session.ValidUntil).
		Returning("id")

	query := q.Build()
	logging.Logger.Debugf("Executing sql: %s", query.Sql)
	err = tx.QueryRow(query.Sql, query.Parameters...).Scan(&resultingId)
	if err != nil {
		panic(mapCustomErrorCodes(err))
	}

	return resultingId
}
