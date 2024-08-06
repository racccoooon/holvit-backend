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
)

type User struct {
	BaseModel

	RealmId uuid.UUID

	Username      h.Optional[string]
	Email         h.Optional[string]
	EmailVerified bool
}

type UserFilter struct {
	BaseFilter

	RealmId         h.Optional[uuid.UUID]
	UsernameOrEmail h.Optional[string]
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

	sb := sqlbuilder.Select("count(*) over()", "id", "realm_id", "username", "email", "email_verified").
		From("users")

	filter.Id.IfSome(func(x uuid.UUID) {
		sb.Where(sb.Equal("id", x))
	})

	filter.RealmId.IfSome(func(x uuid.UUID) {
		sb.Where(sb.Equal("realm_id", x))
	})

	filter.UsernameOrEmail.IfSome(func(x string) {
		sb.Where(
			sb.Or(
				sb.Equal("username", x),
				sb.Equal("email", x),
			))

	})

	filter.PagingInfo.IfSome(func(x PagingInfo) {
		x.Apply(sb)
	})

	sqlString, args := sb.Build()
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
		var username *string
		var email *string
		err := rows.Scan(&totalCount,
			&row.Id,
			&row.RealmId,
			&username,
			&email,
			&row.EmailVerified)
		if err != nil {
			return h.Err[FilterResult[User]](err)
		}

		//TODO: implement valuer for optionals?
		row.Username = h.FromPtr(username)
		row.Email = h.FromPtr(email)

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
    			("realm_id", "username", "email")
    			values ($1, $2, $3)
    			returning "id"`,
		user.RealmId,
		user.Username.ToNillablePtr(),
		user.Email.ToNillablePtr()).
		Scan(&resultingId)
	if err != nil {
		return h.Err[uuid.UUID](err)
	}

	return h.Ok(resultingId)
}
