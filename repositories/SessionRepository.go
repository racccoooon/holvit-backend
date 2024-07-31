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
)

type Session struct {
	BaseModel

	UserId   uuid.UUID
	ClientId uuid.UUID
	RealmId  uuid.UUID

	Token []byte
}

type SessionFilter struct {
	BaseFilter

	RealmId  *uuid.UUID
	UserId   *uuid.UUID
	ClientId *uuid.UUID
}

type SessionRepository interface {
	FindSessionById(ctx context.Context, id uuid.UUID) (*Session, error)
	FindSessions(ctx context.Context, filter SessionFilter) ([]*Session, int, error)
	CreateSession(ctx context.Context, session *Session) (uuid.UUID, error)
}

type SessionRepositoryImpl struct{}

func NewSessionRepository() SessionRepository {
	return &SessionRepositoryImpl{}
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
		"id", "user_id", "client_id", "realm_id", "token").
		From("sessions")

	if filter.RealmId != nil {
		sb.Where(sb.Equal("realm_id", filter.RealmId))
	}
	if filter.UserId != nil {
		sb.Where(sb.Equal("user_id", filter.UserId))
	}
	if filter.ClientId != nil {
		sb.Where(sb.Equal("client_id", filter.ClientId))
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
			&row.ClientId,
			&row.RealmId,
			&row.Token)
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

	err = tx.QueryRow(`insert into "sessions"
    			("user_id", "client_id", "realm_id", "token")
    			values ($1, $2, $3, $4)
    			returning "id"`,
		session.UserId,
		session.ClientId,
		session.RealmId,
		session.Token).
		Scan(&resultingId)

	return resultingId, err
}
