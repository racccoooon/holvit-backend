package repositories

import (
	"context"
	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
	"holvit/httpErrors"
	"holvit/ioc"
	"holvit/logging"
	"holvit/middlewares"
	"holvit/requestContext"
	"holvit/services"
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

	RealmId *uuid.UUID
	UserId  *uuid.UUID

	HashedToken *string
}

type SessionRepository interface {
	FindSessionById(ctx context.Context, id uuid.UUID) (*Session, error)
	FindSessions(ctx context.Context, filter SessionFilter) ([]*Session, int, error)
	CreateSession(ctx context.Context, session *Session) (uuid.UUID, error)
	DeleteOldSessions(ctx context.Context) error
}

type SessionRepositoryImpl struct{}

func NewSessionRepository() SessionRepository {
	return &SessionRepositoryImpl{}
}

func (s *SessionRepositoryImpl) DeleteOldSessions(ctx context.Context) error {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		return err
	}

	clockService := ioc.Get[services.ClockService](scope)
	now := clockService.Now()

	sb := sqlbuilder.DeleteFrom("sessions")
	sb.Where(sb.LessThan("valid_until", now))

	sqlString, args := sb.Build()
	logging.Logger.Debugf("executing sql: %s", sqlString)
	_, err = tx.Exec(sqlString, args...)
	if err != nil {
		return err
	}

	return nil
}

func (s *SessionRepositoryImpl) FindSessionById(ctx context.Context, id uuid.UUID) (*Session, error) {
	result, resultCount, err := s.FindSessions(ctx, SessionFilter{
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
	if resultCount == 0 {
		return nil, httpErrors.NotFound()
	}
	return result[0], nil
}

func (s *SessionRepositoryImpl) FindSessions(ctx context.Context, filter SessionFilter) ([]*Session, int, error) {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		return nil, 0, err
	}

	sb := sqlbuilder.Select("count(*) over()",
		"id", "user_id", "user_device_id", "realm_id", "hashed_token", "valid_until").
		From("sessions")

	if filter.RealmId != nil {
		sb.Where(sb.Equal("realm_id", filter.RealmId))
	}
	if filter.UserId != nil {
		sb.Where(sb.Equal("user_id", filter.UserId))
	}

	if filter.PagingInfo.PageSize > 0 {
		sb.Limit(filter.PagingInfo.PageSize).
			Offset(filter.PagingInfo.PageSize * (filter.PagingInfo.PageNumber - 1))
	}

	sqlString, args := sb.Build()
	logging.Logger.Debugf("executing sql: %s", sqlString)
	rows, err := tx.Query(sqlString, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var totalCount int
	var result []*Session
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
			return nil, 0, err
		}
		result = append(result, &row)
	}

	return result, totalCount, nil
}

func (s *SessionRepositoryImpl) CreateSession(ctx context.Context, session *Session) (uuid.UUID, error) {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	var resultingId uuid.UUID

	tx, err := rcs.GetTx()
	if err != nil {
		return resultingId, err
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

	return resultingId, err
}
