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
	"holvit/sqlb"
	"holvit/utils"
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

	HashedToken h.Opt[string]
	ClientId    h.Opt[uuid.UUID]
}

type RefreshTokenRepository interface {
	FindRefreshTokenById(ctx context.Context, id uuid.UUID) h.Opt[RefreshToken]
	FindRefreshTokens(ctx context.Context, filter RefreshTokenFilter) FilterResult[RefreshToken]
	CreateRefreshToken(ctx context.Context, refreshToken RefreshToken) uuid.UUID
	DeleteRefreshToken(ctx context.Context, id uuid.UUID)
}

type refreshTokenRepositoryImpl struct{}

func NewRefreshTokenRepository() RefreshTokenRepository {
	return &refreshTokenRepositoryImpl{}
}

func (r *refreshTokenRepositoryImpl) FindRefreshTokenById(ctx context.Context, id uuid.UUID) h.Opt[RefreshToken] {
	return r.FindRefreshTokens(ctx, RefreshTokenFilter{
		BaseFilter: BaseFilter{
			Id: h.Some(id),
		},
	}).SingleOrNone()
}

func (r *refreshTokenRepositoryImpl) FindRefreshTokens(ctx context.Context, filter RefreshTokenFilter) FilterResult[RefreshToken] {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		panic(err)
	}

	q := sqlb.Select(filter.CountCol(),
		"id", "user_id", "client_id", "realm_id", "hashed_token", "valid_until", "issuer", "subject", "audience", "scopes").
		From("refresh_tokens")

	filter.Id.IfSome(func(x uuid.UUID) {
		q.Where("id = ?", x)
	})

	filter.HashedToken.IfSome(func(x string) {
		q.Where("hashed_token = ?", x)
	})

	filter.ClientId.IfSome(func(x uuid.UUID) {
		q.Where("client_id = ?", x)
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
			panic(err)
		}
		result = append(result, row)
	}

	return NewPagedResult(result, totalCount)
}

func (r *refreshTokenRepositoryImpl) CreateRefreshToken(ctx context.Context, refreshToken RefreshToken) uuid.UUID {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	var resultingId uuid.UUID

	tx, err := rcs.GetTx()
	if err != nil {
		panic(err)
	}

	q := sqlb.InsertInto("refresh_tokens", "user_id", "client_id", "realm_id", "hashed_token", "valid_until", "issuer", "subject", "audience", "scopes").
		Values(refreshToken.UserId,
			refreshToken.ClientId,
			refreshToken.RealmId,
			refreshToken.HashedToken,
			refreshToken.ValidUntil,
			refreshToken.Issuer,
			refreshToken.Subject,
			refreshToken.Audience,
			pq.Array(refreshToken.Scopes)).
		Returning("id")

	query := q.Build()
	logging.Logger.Debugf("executing sql: %s", query.Sql)
	err = tx.QueryRow(query.Sql, query.Parameters...).Scan(&resultingId)
	if err != nil {
		panic(mapCustomErrorCodes(err))
	}

	return resultingId
}

func (r *refreshTokenRepositoryImpl) DeleteRefreshToken(ctx context.Context, id uuid.UUID) {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		panic(err)
	}

	sb := sqlbuilder.DeleteFrom("refresh_tokens")
	sb.Where(sb.Equal("id", id))

	sqlString, args := sb.Build()
	logging.Logger.Debugf("executing sql: %s", sqlString)
	result, err := tx.Exec(sqlString, args...)
	if err != nil {
		panic(err)
	}
	_, err = result.RowsAffected()
	if err != nil {
		panic(err)
	}
}
