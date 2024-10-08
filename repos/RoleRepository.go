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

type Role struct {
	BaseModel

	RealmId  uuid.UUID
	ClientId h.Opt[uuid.UUID]

	DisplayName string
	Name        string
	Description string

	ImpliesCache []uuid.UUID
	Internal     bool
}

type RoleUpdate struct {
	DisplayName h.Opt[string]
	Name        h.Opt[string]
	Description h.Opt[string]

	ImpliesCache h.Opt[[]uuid.UUID]
}

type DuplicateRoleError struct{}

func (e DuplicateRoleError) Error() string {
	return "Duplicate role"
}

type RoleFilter struct {
	BaseFilter
	RealmId uuid.UUID
	RoleIds h.Opt[[]uuid.UUID]
	Name    h.Opt[string]
}

type RoleRepository interface {
	FindRoleById(ctx context.Context, id uuid.UUID) h.Opt[Role]
	FindRoles(ctx context.Context, filter RoleFilter) FilterResult[Role]
	CreateRole(ctx context.Context, role Role) h.Result[uuid.UUID]
	DeleteRoles(ctx context.Context, realmId uuid.UUID, ids []uuid.UUID)
	UpdateRole(ctx context.Context, id uuid.UUID, upd RoleUpdate)
}

func NewRoleRepository() RoleRepository {
	return &roleRepositoryImpl{}
}

type roleRepositoryImpl struct{}

func (r *roleRepositoryImpl) FindRoleById(ctx context.Context, id uuid.UUID) h.Opt[Role] {
	return r.FindRoles(ctx, RoleFilter{
		BaseFilter: BaseFilter{
			Id: h.Some(id),
		},
	}).SingleOrNone()
}

func (r *roleRepositoryImpl) FindRoles(ctx context.Context, filter RoleFilter) FilterResult[Role] {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		panic(err)
	}

	q := sqlb.Select(filter.CountCol(),
		"id", "realm_id", "client_id", "display_name", "name", "description", "implies_cache", "internal").
		From("roles").
		Where("realm_id = ?", filter.RealmId)

	filter.Id.IfSome(func(x uuid.UUID) {
		q.Where("id = ?", x)
	})

	filter.RoleIds.IfSome(func(x []uuid.UUID) {
		q.Where("id IN (?)", pq.Array(x))
	})

	filter.Name.IfSome(func(x string) {
		q.Where("name = ?", x)
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
	var result []Role
	for rows.Next() {
		var row Role
		err := rows.Scan(&totalCount,
			&row.Id,
			&row.RealmId,
			row.ClientId.AsMutPtr(),
			&row.DisplayName,
			&row.Name,
			&row.Description,
			pq.Array(&row.ImpliesCache),
			&row.Internal)
		if err != nil {
			panic(mapCustomErrorCodes(err))
		}
		result = append(result, row)
	}

	return NewPagedResult(result, totalCount)
}

func (r *roleRepositoryImpl) CreateRole(ctx context.Context, role Role) h.Result[uuid.UUID] {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	var resultingId uuid.UUID

	tx, err := rcs.GetTx()
	if err != nil {
		return h.Err[uuid.UUID](err)
	}

	q := sqlb.InsertInto("roles", "realm_id", "client_id", "display_name", "name", "description", "internal").
		Values(role.RealmId, role.ClientId.ToNillablePtr(), role.DisplayName, role.Name, role.Description, role.Internal).
		Returning("id")

	query := q.Build()
	logging.Logger.Debugf("executing sql: %s", query.Sql)
	err = tx.QueryRow(query.Sql, query.Parameters...).Scan(&resultingId)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code.Name() {
			case "unique_violation":
				if pqErr.Constraint == "idx_unique_role_per_realm" {
					return h.Err[uuid.UUID](DuplicateRoleError{})
				}
			}
		}

		panic(mapCustomErrorCodes(err))
	}

	return h.Ok(resultingId)
}

func (r *roleRepositoryImpl) DeleteRoles(ctx context.Context, realmId uuid.UUID, ids []uuid.UUID) {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		panic(err)
	}

	q := sqlb.DeleteFrom("roles").
		Where("realm_id =  ?", realmId).
		Where("id = any(?)", pq.Array(ids))
	query := q.Build()
	logging.Logger.Debugf("executing sql: %s", query.Sql)
	_, err = tx.Exec(query.Sql, query.Parameters...)
	if err != nil {
		panic(mapCustomErrorCodes(err))
	}
}

func (r *roleRepositoryImpl) UpdateRole(ctx context.Context, id uuid.UUID, upd RoleUpdate) {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		panic(err)
	}

	q := sqlb.Update("roles")

	upd.DisplayName.IfSome(func(x string) {
		q.Set("display_name", x)
	})

	upd.Name.IfSome(func(x string) {
		q.Set("name", x)
	})

	upd.Description.IfSome(func(x string) {
		q.Set("description", x)
	})

	upd.ImpliesCache.IfSome(func(x []uuid.UUID) {
		q.Set("implies_cache", pq.Array(x))
	})

	q.Where("id = ?", id)

	query := q.Build()
	logging.Logger.Debugf("executing sql: %s", query.Sql)
	_, err = tx.Exec(query.Sql, query.Parameters...)
	if err != nil {
		panic(mapCustomErrorCodes(err))
	}
}
