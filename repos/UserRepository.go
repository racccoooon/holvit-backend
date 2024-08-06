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
)

type User struct {
	BaseModel

	RealmId uuid.UUID

	Username      string
	Email         h.Optional[string]
	EmailVerified bool
}

type DuplicateUsernameError struct {
	RealmId  uuid.UUID
	Username string
}

func (e DuplicateUsernameError) Error() string {
	return fmt.Sprintf("Username '%s' already in use in realm '%s'", e.Username, e.RealmId.String())
}

type UserFilter struct {
	BaseFilter

	RealmId  h.Optional[uuid.UUID]
	Username h.Optional[string]
}

type UserRepository interface {
	FindUserById(ctx context.Context, id uuid.UUID) h.Optional[User]
	FindUsers(ctx context.Context, filter UserFilter) h.Result[FilterResult[User]]
	CreateUser(ctx context.Context, user *User) h.Result[uuid.UUID]
}

type UserRepositoryImpl struct{}

func NewUserRepository() UserRepository {
	return &UserRepositoryImpl{}
}

func (u *UserRepositoryImpl) FindUserById(ctx context.Context, id uuid.UUID) h.Optional[User] {
	return u.FindUsers(ctx, UserFilter{
		BaseFilter: BaseFilter{
			Id: h.Some(id),
		},
	}).Unwrap().FirstOrNone()
}

func (u *UserRepositoryImpl) FindUsers(ctx context.Context, filter UserFilter) h.Result[FilterResult[User]] {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		return h.Err[FilterResult[User]](err)
	}

	sqlString := `select ` + filter.CountCol() + `, "id", "realm_id", "username", "email", "email_verified" from users where true`

	args := []interface{}{}
	filter.Id.IfSome(func(x uuid.UUID) {
		args = append(args, x)
		sqlString += fmt.Sprintf(" id = $%d", len(args))
	})

	filter.RealmId.IfSome(func(x uuid.UUID) {
		args = append(args, x)
		sqlString += fmt.Sprintf(" realm_id = $%d", len(args))
	})

	filter.Username.IfSome(func(x string) {
		args = append(args, x)
		sqlString += fmt.Sprintf(" username = lower($%d)", len(args))
	})

	filter.PagingInfo.IfSome(func(x PagingInfo) {
		sqlString += x.SqlString()
	})

	logging.Logger.Debugf("executing sql: %s", sqlString)
	rows, err := tx.Query(sqlString, args...)
	if err != nil {
		return h.Err[FilterResult[User]](err)
	}
	defer rows.Close()

	var totalCount int
	var result []User
	for rows.Next() {
		var row User
		err := rows.Scan(&totalCount,
			&row.Id,
			&row.RealmId,
			&row.Username,
			row.Email.ToNillablePtr(),
			&row.EmailVerified)
		if err != nil {
			return h.Err[FilterResult[User]](err)
		}

		result = append(result, row)
	}

	return h.Ok(NewPagedResult(result, totalCount))
}

func (u *UserRepositoryImpl) CreateUser(ctx context.Context, user *User) h.Result[uuid.UUID] {
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
					return h.Err[uuid.UUID](DuplicateUsernameError{
						Username: user.Username,
						RealmId:  user.RealmId,
					})
				}
				break
			}
		}

		return h.Err[uuid.UUID](err)
	}

	return h.Ok(resultingId)
}
