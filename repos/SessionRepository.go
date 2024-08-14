package repos

import (
	"context"
	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
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

type SessionRepositoryImpl struct{}

func NewSessionRepository() SessionRepository {
	return &SessionRepositoryImpl{}
}

func (s *SessionRepositoryImpl) DeleteOldSessions(ctx context.Context) {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		panic(err)
	}

	clockService := ioc.Get[utils.ClockService](scope)
	now := clockService.Now()

	sb := sqlbuilder.DeleteFrom("sessions")
	sb.Where(sb.LessThan("valid_until", now))

	sqlString, args := sb.Build()
	logging.Logger.Debugf("executing sql: %s", sqlString)
	_, err = tx.Exec(sqlString, args...)
	if err != nil {
		panic(err)
	}
}

func (s *SessionRepositoryImpl) FindSessionById(ctx context.Context, id uuid.UUID) h.Opt[Session] {
	return s.FindSessions(ctx, SessionFilter{
		BaseFilter: BaseFilter{
			Id: h.Some(id),
		},
	}).FirstOrNone()
}

func (s *SessionRepositoryImpl) FindSessions(ctx context.Context, filter SessionFilter) FilterResult[Session] {
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
		x.Apply2(q)
	})

	filter.SortInfo.IfSome(func(x SortInfo) {
		x.Apply2(q)
	})

	query := q.Build()
	logging.Logger.Debugf("executing sql: %s", query.Query)
	rows, err := tx.Query(query.Query, query.Parameters...)
	if err != nil {
		panic(err)
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
			panic(err)
		}
		result = append(result, row)
	}

	return NewPagedResult(result, totalCount)
}

func (s *SessionRepositoryImpl) CreateSession(ctx context.Context, session Session) uuid.UUID {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	var resultingId uuid.UUID

	tx, err := rcs.GetTx()
	if err != nil {
		panic(err)
	}

	sqlString := `insert into "sessions"
    			("user_id", "user_device_id", "realm_id", "hashed_token", "valid_until")
    			values ($1, $2, $3, $4, $5)
    			returning "id"`
	logging.Logger.Debugf("Executing sql: %s", sqlString)

	err = tx.QueryRow(sqlString,
		session.UserId,
		session.UserDeviceId,
		session.RealmId,
		session.HashedToken,
		session.ValidUntil).Scan(&resultingId)
	if err != nil {
		panic(err)
	}

	return resultingId
}
