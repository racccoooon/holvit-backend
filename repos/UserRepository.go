package repos

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"holvit/h"
	"holvit/ioc"
	"holvit/logging"
	"holvit/middlewares"
	"holvit/requestContext"
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

type UserRepositoryImpl struct{}

func NewUserRepository() UserRepository {
	return &UserRepositoryImpl{}
}

func (u *UserRepositoryImpl) FindUserById(ctx context.Context, id uuid.UUID) h.Opt[User] {
	return u.FindUsers(ctx, UserFilter{
		BaseFilter: BaseFilter{
			Id: h.Some(id),
		},
	}).FirstOrNone()
}

func (u *UserRepositoryImpl) FindUsers(ctx context.Context, filter UserFilter) FilterResult[User] {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		panic(err)
	}

	sqlString := `select ` + filter.CountCol() + `, "id", "realm_id", "username", "email", "email_verified" from users where true`

	args := make([]interface{}, 0)
	filter.Id.IfSome(func(x uuid.UUID) {
		args = append(args, x)
		sqlString += fmt.Sprintf(" and id = $%d", len(args))
	})

	filter.RealmId.IfSome(func(x uuid.UUID) {
		args = append(args, x)
		sqlString += fmt.Sprintf(" and realm_id = $%d", len(args))
	})

	filter.Username.IfSome(func(x string) {
		args = append(args, x)
		sqlString += fmt.Sprintf(" and username = lower($%d)", len(args))
	})

	filter.PagingInfo.IfSome(func(x PagingInfo) {
		sqlString += x.SqlString()
	})

	filter.SortInfo.IfSome(func(x SortInfo) {
		sqlString += x.SqlString()
	})

	logging.Logger.Debugf("executing sql: %s", sqlString)
	rows, err := tx.Query(sqlString, args...)
	if err != nil {
		panic(err)
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
			panic(err)
		}

		result = append(result, row)
	}

	return NewPagedResult(result, totalCount)
}

func (u *UserRepositoryImpl) CreateUser(ctx context.Context, user User) h.Result[uuid.UUID] {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	var resultingId uuid.UUID

	tx, err := rcs.GetTx()
	if err != nil {
		return h.Err[uuid.UUID](err)
	}

	err = tx.QueryRow(`insert into "users"
    			("realm_id", "username", "email", "email_verified")
    			values ($1, $2, $3, $4)
    			returning "id"`,
		user.RealmId,
		user.Username,
		user.Email.ToNillablePtr(),
		user.EmailVerified).Scan(&resultingId)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code.Name() {
			case "unique_violation":
				if pqErr.Constraint == "idx_unique_username_per_realm" {
					return h.Err[uuid.UUID](DuplicateUsernameError{})
				}
			}
		} else {
			panic(err)
		}
	}

	return h.Ok(resultingId)
}
