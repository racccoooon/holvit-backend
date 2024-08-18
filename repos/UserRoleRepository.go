package repos

import (
	"context"
	"github.com/google/uuid"
	"holvit/ioc"
	"holvit/logging"
	"holvit/middlewares"
	"holvit/requestContext"
	"holvit/sqlb"
	"holvit/utils"
)

type UserRole struct {
	BaseModel

	UserId uuid.UUID
	RoleId uuid.UUID
}

type UserRoleFilter struct {
	UserId uuid.UUID
}

type UserRoleRepository interface {
	CreateUserRoles(ctx context.Context, userRoles []UserRole)
	DeleteUserRole(ctx context.Context, id uuid.UUID)
	FindUserRoles(ctx context.Context, filter UserRoleFilter) []UserRole
}

func NewUserRoleRepository() UserRoleRepository {
	return &userRoleRepositoryImpl{}
}

type userRoleRepositoryImpl struct{}

func (u *userRoleRepositoryImpl) CreateUserRoles(ctx context.Context, userRoles []UserRole) {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		panic(err)
	}

	q := sqlb.InsertInto("user_roles", "user_id", "role_id")

	for _, userRole := range userRoles {
		q.Values(userRole.UserId, userRole.RoleId)
	}

	q.OnConflict().DoNothing()

	query := q.Build()
	logging.Logger.Debugf("executing sql: %s", query.Sql)
	_, err = tx.Exec(query.Sql, query.Parameters...)
	if err != nil {
		panic(mapCustomErrorCodes(err))
	}
}

func (u *userRoleRepositoryImpl) DeleteUserRole(ctx context.Context, id uuid.UUID) {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		panic(err)
	}

	q := sqlb.DeleteFrom("user_roles").
		Where("id = ?", id)

	query := q.Build()
	logging.Logger.Debugf("executing sql: %s", query.Sql)
	_, err = tx.Exec(query.Sql, query.Parameters...)
	if err != nil {
		panic(mapCustomErrorCodes(err))
	}
}

func (u *userRoleRepositoryImpl) FindUserRoles(ctx context.Context, filter UserRoleFilter) []UserRole {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		panic(err)
	}

	q := sqlb.Select("id", "user_id", "role_id").
		From("user_roles")

	q.Where("user_id = ?", filter.UserId)

	query := q.Build()
	logging.Logger.Debugf("executing sql: %s", query.Sql)
	rows, err := tx.Query(query.Sql, query.Parameters...)
	if err != nil {
		panic(mapCustomErrorCodes(err))
	}
	defer utils.PanicOnErr(rows.Close)

	var totalCount int
	var result []UserRole
	for rows.Next() {
		var row UserRole
		err := rows.Scan(&totalCount,
			&row.Id,
			&row.UserId,
			&row.RoleId)
		if err != nil {
			panic(mapCustomErrorCodes(err))
		}
		result = append(result, row)
	}

	return result
}
