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
	"holvit/utils"
)

type Role struct {
	BaseModel

	RealmId  uuid.UUID
	ClientId h.Opt[uuid.UUID]

	DisplayName string
	Name        string
	Description string
}

type DuplicateRoleError struct{}

func (e DuplicateRoleError) Error() string {
	return "Duplicate role"
}

type RoleFilter struct {
	BaseFilter
}

type RolesRepository interface {
	FindRoleById(ctx context.Context, id uuid.UUID) h.Opt[Role]
	FindRoles(ctx context.Context, filter RoleFilter) FilterResult[Role]
	CreateRole(ctx context.Context, role Role) h.Result[uuid.UUID]
}

func NewRolesRepository() RolesRepository {
	return &RoleRepositoryImpl{}
}

type RoleRepositoryImpl struct{}

func (r *RoleRepositoryImpl) FindRoleById(ctx context.Context, id uuid.UUID) h.Opt[Role] {
	return r.FindRoles(ctx, RoleFilter{
		BaseFilter: BaseFilter{
			Id: h.Some(id),
		},
	}).FirstOrNone()
}

func (r *RoleRepositoryImpl) FindRoles(ctx context.Context, filter RoleFilter) FilterResult[Role] {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		panic(err)
	}

	sb := sqlbuilder.Select(filter.CountCol(),
		"id", "realm_id", "client_id", "display_name", "name", "description").
		From("roles")

	filter.Id.IfSome(func(x uuid.UUID) {
		sb.Where(sb.Equal("id", x))
	})

	filter.PagingInfo.IfSome(func(x PagingInfo) {
		x.Apply(sb)
	})

	filter.SortInfo.IfSome(func(x SortInfo) {
		x.Apply(sb)
	})

	sqlString, args := sb.Build()
	logging.Logger.Debugf("executing sql: %s", sqlString)
	rows, err := tx.Query(sqlString, args...)
	if err != nil {
		panic(err)
	}
	defer utils.PanicOnErr(rows.Close)

	var totalCount int
	var result []Role
	for rows.Next() {
		var row Role
		err := rows.Scan(&totalCount,
			&row.Id,
			&row.RealmId,
			row.ClientId.AsMutPtr(),
			&row.DisplayName,
			&row.Name,
			&row.Description)
		if err != nil {
			panic(err)
		}
		result = append(result, row)
	}

	return NewPagedResult(result, totalCount)
}

func (r *RoleRepositoryImpl) CreateRole(ctx context.Context, role Role) h.Result[uuid.UUID] {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	var resultingId uuid.UUID

	tx, err := rcs.GetTx()
	if err != nil {
		return h.Err[uuid.UUID](err)
	}

	sqlString := `insert into "roles"
				("realm_id", "client_id", "display_name", "name", "description")
				values ($1, $2, $3, $4, $5)
				returning "id"`

	logging.Logger.Debugf("executing sql: %s", sqlString)
	err = tx.QueryRow(sqlString,
		role.RealmId,
		role.ClientId.ToNillablePtr(),
		role.DisplayName,
		role.Name,
		role.Description).Scan(&resultingId)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code.Name() {
			case "unique_violation":
				if pqErr.Constraint == "idx_unique_role_per_realm" {
					return h.Err[uuid.UUID](DuplicateRoleError{})
				}
			}
		} else {
			panic(err)
		}
	}

	return h.Ok(resultingId)
}
