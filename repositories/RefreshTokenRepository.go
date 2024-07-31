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
	"time"
)

type RefreshToken struct {
	BaseModel

	UserId   uuid.UUID
	ClientId uuid.UUID
	RealmId  uuid.UUID

	HashedToken string
	ValidUntil  time.Time
}

type RefreshTokenFilter struct {
	BaseFilter

	HashedToken *string
	ClientId    *uuid.UUID
}

type RefreshTokenRepository interface {
	FindRefreshTokenById(ctx context.Context, id uuid.UUID) (*RefreshToken, error)
	FindRefreshTokens(ctx context.Context, filter RefreshTokenFilter) ([]*RefreshToken, int, error)
	CreateRefreshToken(ctx context.Context, refreshToken *RefreshToken) (uuid.UUID, error)
	DeleteRefreshToken(ctx context.Context, id uuid.UUID) error
}

type RefreshTokenRepositoryImpl struct{}

func NewRefreshTokenRepository() RefreshTokenRepository {
	return &RefreshTokenRepositoryImpl{}
}

func (r *RefreshTokenRepositoryImpl) FindRefreshTokenById(ctx context.Context, id uuid.UUID) (*RefreshToken, error) {
	result, resultCount, err := r.FindRefreshTokens(ctx, RefreshTokenFilter{
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

func (r *RefreshTokenRepositoryImpl) FindRefreshTokens(ctx context.Context, filter RefreshTokenFilter) ([]*RefreshToken, int, error) {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		return nil, 0, err
	}

	sb := sqlbuilder.Select("count(*) over()",
		"id", "user_id", "client_id", "realm_id", "hashed_token", "valid_until").
		From("refresh_tokens")

	if filter.HashedToken != nil {
		sb.Where(sb.Equal("hashed_token", *filter.HashedToken))
	}

	if filter.ClientId != nil {
		sb.Where(sb.Equal("client_id", *filter.ClientId))
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
	var result []*RefreshToken
	for rows.Next() {
		var row RefreshToken
		err := rows.Scan(&totalCount,
			&row.Id,
			&row.UserId,
			&row.ClientId,
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

func (r *RefreshTokenRepositoryImpl) CreateRefreshToken(ctx context.Context, refreshToken *RefreshToken) (uuid.UUID, error) {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	var resultingId uuid.UUID

	tx, err := rcs.GetTx()
	if err != nil {
		return resultingId, err
	}

	err = tx.QueryRow(`insert into "refresh_tokens"
    			("user_id", "client_id", "realm_id", "hashed_token", "valid_until")
    			values ($1, $2, $3)
    			returning "id"`,
		refreshToken.UserId,
		refreshToken.ClientId,
		refreshToken.RealmId,
		refreshToken.HashedToken,
		refreshToken.ValidUntil).
		Scan(&resultingId)

	return resultingId, err
}

func (r *RefreshTokenRepositoryImpl) DeleteRefreshToken(ctx context.Context, id uuid.UUID) error {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		return err
	}

	sb := sqlbuilder.DeleteFrom("refresh_tokens")
	sb.Where(sb.Equal("id", id))

	sqlString, args := sb.Build()
	logging.Logger.Debugf("executing sql: %s", sqlString)
	_, err = tx.Exec(sqlString, args...)
	if err != nil {
		return err
	}

	return nil
}
