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
)

type Scope struct {
	BaseModel

	RealmId     uuid.UUID
	Name        string
	DisplayName string
	Description string

	SortIndex int

	Grant h.Opt[Grant]
}

type DuplicateScopeError struct{}

func (e DuplicateScopeError) Error() string {
	return "Duplicate Scope"
}

type Grant struct {
	BaseModel

	ScopeId  uuid.UUID
	UserId   uuid.UUID
	ClientId uuid.UUID
}

type ScopeFilter struct {
	BaseFilter

	RealmId uuid.UUID
	Names   h.Opt[[]string]

	IncludeGrants bool
	OnlyGranted   bool

	UserId   h.Opt[uuid.UUID]
	ClientId h.Opt[uuid.UUID]
}

type ScopeRepository interface {
	FindScopeById(ctx context.Context, id uuid.UUID) h.Opt[Scope]
	FindScopes(ctx context.Context, filter ScopeFilter) FilterResult[Scope]
	CreateScope(ctx context.Context, scope Scope) h.Result[uuid.UUID]
	CreateGrants(ctx context.Context, userId uuid.UUID, clientId uuid.UUID, scopeIds []uuid.UUID)
}

type scopeRepositoryImpl struct{}

func NewScopeRepository() ScopeRepository {
	return &scopeRepositoryImpl{}
}

func (s *scopeRepositoryImpl) FindScopeById(ctx context.Context, id uuid.UUID) h.Opt[Scope] {
	return s.FindScopes(ctx, ScopeFilter{
		BaseFilter: BaseFilter{
			Id: h.Some(id),
		},
	}).FirstOrNone()
}

func (s *scopeRepositoryImpl) FindScopes(ctx context.Context, filter ScopeFilter) FilterResult[Scope] {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		panic(err)
	}

	q := sqlb.Select(filter.CountCol(), "s.id", "s.realm_id", "s.name", "s.display_name", "s.description").
		From("scopes s")

	if filter.IncludeGrants {
		q.Select("g.id", "g.scope_id", "g.user_id", "g.client_id")
	}

	if filter.IncludeGrants {
		if filter.OnlyGranted {
			q.InnerJoin("grants g", "g.scope_id = s.id")
		} else {
			q.LeftJoin("grants g", "g.scope_id = s.id")
		}
	}

	q.Where("s.realm_id = ?", filter.RealmId)

	filter.Id.IfSome(func(x uuid.UUID) {
		q.Where("s.id = ?", x)
	})

	filter.Names.IfSome(func(x []string) {
		q.Where("s.name = any(?::text[])", pq.Array(x))
	})

	filter.ClientId.IfSome(func(x uuid.UUID) {
		q.Where("(g.client_id = ? or g.client_id is null)", x)
	})

	filter.PagingInfo.IfSome(func(x PagingInfo) {
		x.Apply(q)
	})

	if x, ok := filter.SortInfo.Get(); ok {
		x.Apply(q)
	}
	q.OrderBy("sort_index asc")

	query := q.Build()
	logging.Logger.Debugf("executing sql: %s", query.Query)
	rows, err := tx.Query(query.Query, query.Parameters...)
	if err != nil {
		panic(mapCustomErrorCodes(err))
	}
	defer utils.PanicOnErr(rows.Close)

	var totalCount int
	var result []Scope
	for rows.Next() {
		var row Scope
		var grantId h.Opt[uuid.UUID]
		var grantScopeId h.Opt[uuid.UUID]
		var grantUserId h.Opt[uuid.UUID]
		var grantClientId h.Opt[uuid.UUID]
		scan := []any{&totalCount,
			&row.Id,
			&row.RealmId,
			&row.Name,
			&row.DisplayName,
			&row.Description}
		if filter.IncludeGrants {
			scan = append(scan,
				grantId.AsMutPtr(),
				grantScopeId.AsMutPtr(),
				grantUserId.AsMutPtr(),
				grantClientId.AsMutPtr())
		}
		err := rows.Scan(scan)
		if err != nil {
			panic(mapCustomErrorCodes(err))
		}

		if filter.IncludeGrants && grantId.IsSome() {
			row.Grant = h.Some(Grant{
				BaseModel: BaseModel{
					Id: grantId.Unwrap(),
				},
				ScopeId:  grantScopeId.Unwrap(),
				UserId:   grantUserId.Unwrap(),
				ClientId: grantClientId.Unwrap(),
			})
		}

		result = append(result, row)
	}

	return NewPagedResult(result, totalCount)
}

func (s *scopeRepositoryImpl) CreateScope(ctx context.Context, scope Scope) h.Result[uuid.UUID] {
	iocScope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](iocScope)

	var resultingId uuid.UUID

	tx, err := rcs.GetTx()
	if err != nil {
		return h.Err[uuid.UUID](err)
	}

	q := sqlb.InsertInto("scopes", "realm_id", "name", "display_name", "description", "sort_index").
		Values(scope.RealmId,
			scope.Name,
			scope.DisplayName,
			scope.Description,
			scope.SortIndex).
		Returning("id")

	query := q.Build()
	logging.Logger.Debugf("executing sql: %s", query.Query)
	err = tx.QueryRow(query.Query, query.Parameters...).Scan(&resultingId)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code.Name() {
			case "unique_violation":
				if pqErr.Constraint == "idx_unique_scope_name_in_realm" {
					return h.Err[uuid.UUID](DuplicateScopeError{})
				}
			}
		}

		panic(mapCustomErrorCodes(err))
	}

	return h.Ok(resultingId)
}

func (s *scopeRepositoryImpl) CreateGrants(ctx context.Context, userId uuid.UUID, clientId uuid.UUID, scopeIds []uuid.UUID) {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		panic(err)
	}

	sb := sqlbuilder.InsertInto("grants").
		Cols("scope_id", "user_id", "client_id")

	for _, scopeId := range scopeIds {
		sb.Values(scopeId, userId, clientId)
	}

	sqlString, args := sb.Build()
	logging.Logger.Debugf("executing sql: %s", sqlString)
	_, err = tx.Exec(sqlString, args...)

	if err != nil {
		panic(mapCustomErrorCodes(err))
	}
}
