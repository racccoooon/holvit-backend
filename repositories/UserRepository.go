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

type User struct {
	BaseModel

	RealmId uuid.UUID

	Username      *string
	Email         *string
	EmailVerified bool
}

type UserFilter struct {
	BaseFilter

	RealmId            uuid.UUID
	UsernameOrPassword *string
}

type UserRepository interface {
	FindUserById(ctx context.Context, id uuid.UUID) (*User, error)
	FindUsers(ctx context.Context, filter UserFilter) ([]*User, int, error)
	CreateUser(ctx context.Context, user *User) (uuid.UUID, error)
}

type UserRepositoryImpl struct{}

func NewUserRepository() UserRepository {
	return &UserRepositoryImpl{}
}

func (u *UserRepositoryImpl) FindUserById(ctx context.Context, id uuid.UUID) (*User, error) {
	users, userCount, err := u.FindUsers(ctx, UserFilter{
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
	if userCount == 0 {
		return nil, httpErrors.NotFound()
	}
	return users[0], nil
}

func (u *UserRepositoryImpl) FindUsers(ctx context.Context, filter UserFilter) ([]*User, int, error) {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		return nil, 0, err
	}

	sb := sqlbuilder.Select("count(*) over()", "id", "realm_id", "username", "email", "email_verified").
		From("users")

	sb.Where(sb.Equal("realm_id", filter.RealmId))

	if filter.UsernameOrPassword != nil {
		sb.Where(
			sb.Or(
				sb.Equal("username", *filter.UsernameOrPassword),
				sb.Equal("email", *filter.UsernameOrPassword),
			))
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
	var result []*User
	for rows.Next() {
		var row User
		err := rows.Scan(&totalCount,
			&row.Id,
			&row.RealmId,
			&row.Username,
			&row.Email,
			&row.EmailVerified)
		if err != nil {
			return nil, 0, err
		}
		result = append(result, &row)
	}

	return result, totalCount, nil
}

func (u *UserRepositoryImpl) CreateUser(ctx context.Context, user *User) (uuid.UUID, error) {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	var resultingId uuid.UUID

	tx, err := rcs.GetTx()
	if err != nil {
		return resultingId, err
	}

	err = tx.QueryRow(`insert into "users"
    			("realm_id", "username", "email")
    			values ($1, $2, $3)
    			returning "id"`,
		user.RealmId,
		user.Username,
		user.Email).
		Scan(&resultingId)

	return resultingId, err
}
