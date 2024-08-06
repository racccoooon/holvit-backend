package repos

import (
	"context"
	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
	"github.com/lib/pq"
	"holvit/h"
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

	Issuer   string
	Subject  string
	Audience string
	Scopes   []string
}

type RefreshTokenFilter struct {
	BaseFilter

	HashedToken h.Optional[string]
	ClientId    h.Optional[uuid.UUID]
}

type RefreshTokenRepository interface {
	FindRefreshTokenById(ctx context.Context, id uuid.UUID) h.Optional[RefreshToken]
	FindRefreshTokens(ctx context.Context, filter RefreshTokenFilter) h.Result[FilterResult[RefreshToken]]
	CreateRefreshToken(ctx context.Context, refreshToken *RefreshToken) h.Result[uuid.UUID]
	DeleteRefreshToken(ctx context.Context, id uuid.UUID) error
}

type RefreshTokenRepositoryImpl struct{}

func NewRefreshTokenRepository() RefreshTokenRepository {
	return &RefreshTokenRepositoryImpl{}
}

func (r *RefreshTokenRepositoryImpl) FindRefreshTokenById(ctx context.Context, id uuid.UUID) h.Optional[RefreshToken] {
	return r.FindRefreshTokens(ctx, RefreshTokenFilter{
		BaseFilter: BaseFilter{
			Id: h.Some(id),
		},
	}).Unwrap().FirstOrNone()
}

func (r *RefreshTokenRepositoryImpl) FindRefreshTokens(ctx context.Context, filter RefreshTokenFilter) h.Result[FilterResult[RefreshToken]] {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		return h.Err[FilterResult[RefreshToken]](err)
	}

	sb := sqlbuilder.Select(filter.CountCol(),
		"id", "user_id", "client_id", "realm_id", "hashed_token", "valid_until", "issuer", "subject", "audience", "scopes").
		From("refresh_tokens")

	filter.Id.IfSome(func(x uuid.UUID) {
		sb.Where(sb.Equal("id", x))
	})

	filter.HashedToken.IfSome(func(x string) {
		sb.Where(sb.Equal("hashed_token", x))
	})

	filter.ClientId.IfSome(func(x uuid.UUID) {
		sb.Where(sb.Equal("client_id", x))
	})

	filter.PagingInfo.IfSome(func(x PagingInfo) {
		x.Apply(sb)
	})

	sqlString, args := sb.Build()
	logging.Logger.Debugf("executing sql: %s", sqlString)
	rows, err := tx.Query(sqlString, args...)
	if err != nil {
		return h.Err[FilterResult[RefreshToken]](err)
	}
	defer rows.Close()

	var totalCount int
	var result []RefreshToken
	for rows.Next() {
		var row RefreshToken
		err := rows.Scan(&totalCount,
			&row.Id,
			&row.UserId,
			&row.ClientId,
			&row.RealmId,
			&row.HashedToken,
			&row.ValidUntil,
			&row.Issuer,
			&row.Subject,
			&row.Audience,
			pq.Array(&row.Scopes))
		if err != nil {
			return h.Err[FilterResult[RefreshToken]](err)
		}
		result = append(result, row)
	}

	return h.Ok(NewPagedResult(result, totalCount))
}

func (r *RefreshTokenRepositoryImpl) CreateRefreshToken(ctx context.Context, refreshToken *RefreshToken) h.Result[uuid.UUID] {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	var resultingId uuid.UUID

	tx, err := rcs.GetTx()
	if err != nil {
		return h.Err[uuid.UUID](err)
	}

	sqlString := `insert into "refresh_tokens"
    			("user_id", "client_id", "realm_id", "hashed_token", "valid_until", "issuer", "subject", "audience", "scopes")
    			values ($1, $2, $3, $4, $5, $6, $7, $8, $9)
    			returning "id"`
	logging.Logger.Debugf("executing sql: %s", sqlString)

	err = tx.QueryRow(sqlString,
		refreshToken.UserId,
		refreshToken.ClientId,
		refreshToken.RealmId,
		refreshToken.HashedToken,
		refreshToken.ValidUntil,
		refreshToken.Issuer,
		refreshToken.Subject,
		refreshToken.Audience,
		pq.Array(refreshToken.Scopes)).
		Scan(&resultingId)
	if err != nil {
		return h.Err[uuid.UUID](err)
	}

	return h.Ok(resultingId)
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
