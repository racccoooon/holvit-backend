package repos

import (
	"context"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"holvit/h"
	"holvit/ioc"
	"holvit/logging"
	"holvit/middlewares"
	"holvit/requestContext"
	"holvit/sqlb"
	"holvit/utils"
)

type User struct {
	BaseModel

	RealmId uuid.UUID

	Username      string
	Email         h.Opt[string]
	EmailVerified bool
}

type DuplicateUsernameError struct{}

func (e DuplicateUsernameError) Error() string {
	return "Duplicate username"
}

type UserFilter struct {
	BaseFilter

	RealmId  h.Opt[uuid.UUID]
	Username h.Opt[string]
}

type UserRepository interface {
	FindUserById(ctx context.Context, id uuid.UUID) h.Opt[User]
	FindUsers(ctx context.Context, filter UserFilter) FilterResult[User]
	CreateUser(ctx context.Context, user User) h.Result[uuid.UUID]
}

type userRepositoryImpl struct{}

func NewUserRepository() UserRepository {
	return &userRepositoryImpl{}
}

func (u *userRepositoryImpl) FindUserById(ctx context.Context, id uuid.UUID) h.Opt[User] {
	return u.FindUsers(ctx, UserFilter{
		BaseFilter: BaseFilter{
			Id: h.Some(id),
		},
	}).FirstOrNone()
}

func (u *userRepositoryImpl) FindUsers(ctx context.Context, filter UserFilter) FilterResult[User] {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		panic(err)
	}

	q := sqlb.Select(filter.CountCol(), "id", "realm_id", "username", "email", "email_verified").
		From("users")

	filter.Id.IfSome(func(x uuid.UUID) {
		q.Where("id = ?", x)
	})

	filter.RealmId.IfSome(func(x uuid.UUID) {
		q.Where("realm_id = ?", x)
	})

	filter.Username.IfSome(func(x string) {
		q.Where("username = lower($%d)", x)
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
	var result []User
	for rows.Next() {
		var row User
		err := rows.Scan(&totalCount,
			&row.Id,
			&row.RealmId,
			&row.Username,
			row.Email.AsMutPtr(),
			&row.EmailVerified)
		if err != nil {
			panic(mapCustomErrorCodes(err))
		}

		result = append(result, row)
	}

	return NewPagedResult(result, totalCount)
}

func (u *userRepositoryImpl) CreateUser(ctx context.Context, user User) h.Result[uuid.UUID] {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	var resultingId uuid.UUID

	tx, err := rcs.GetTx()
	if err != nil {
		return h.Err[uuid.UUID](err)
	}

	q := sqlb.InsertInto("users", "realm_id", "username", "email", "email_verified").
		Values(user.RealmId,
			user.Username,
			user.Email.ToNillablePtr(),
			user.EmailVerified).
		Returning("id")

	query := q.Build()
	logging.Logger.Debugf("executing sql: %s", query.Sql)
	err = tx.QueryRow(query.Sql, query.Parameters...).Scan(&resultingId)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code.Name() {
			case "unique_violation":
				if pqErr.Constraint == "idx_unique_username_per_realm" {
					return h.Err[uuid.UUID](DuplicateUsernameError{})
				}
			}
		}

		panic(mapCustomErrorCodes(err))
	}

	return h.Ok(resultingId)
}
