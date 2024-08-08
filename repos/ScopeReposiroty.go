package repos

import (
	"context"
	"fmt"
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

type Scope struct {
	BaseModel

	RealmId     uuid.UUID
	Name        string
	DisplayName string
	Description string

	SortIndex int

	Grant *Grant
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
	Names   h.Optional[[]string]

	IncludeGrants bool
	OnlyGranted   bool

	UserId   h.Optional[uuid.UUID]
	ClientId h.Optional[uuid.UUID]
}

type ScopeRepository interface {
	FindScopeById(ctx context.Context, id uuid.UUID) h.Optional[Scope]
	FindScopes(ctx context.Context, filter ScopeFilter) FilterResult[Scope]
	CreateScope(ctx context.Context, scope Scope) h.Result[uuid.UUID]
	CreateGrants(ctx context.Context, userId uuid.UUID, clientId uuid.UUID, scopeIds []uuid.UUID)
}

type ScopeRepositoryImpl struct{}

func NewScopeReposiroty() ScopeRepository {
	return &ScopeRepositoryImpl{}
}

func (s *ScopeRepositoryImpl) FindScopeById(ctx context.Context, id uuid.UUID) h.Optional[Scope] {
	return s.FindScopes(ctx, ScopeFilter{
		BaseFilter: BaseFilter{
			Id: h.Some(id),
		},
	}).FirstOrNone()
}

func (s *ScopeRepositoryImpl) FindScopes(ctx context.Context, filter ScopeFilter) FilterResult[Scope] {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		panic(err)
	}

	sql := "select " + filter.CountCol() + ", s.id, s.realm_id, s.name, s.display_name, s.description"

	if filter.IncludeGrants {
		sql += ", g.id, g.scope_id, g.user_id, g.client_id"
	} else {
		sql += ", null, null, null, null"
	}

	sql += " from scopes s"

	if filter.IncludeGrants {
		if filter.OnlyGranted {
			sql += " inner join"
		} else {
			sql += " left outer join"
		}
		sql += " grants g on g.scope_id = s.id"
	}

	parameters := make([]interface{}, 0)
	parameters = append(parameters, filter.RealmId)
	sql += fmt.Sprintf(" where s.realm_id = $%d", len(parameters))

	filter.Id.IfSome(func(x uuid.UUID) {
		parameters = append(parameters, x)
		sql += fmt.Sprintf(" and s.id = $%d", len(parameters))
	})

	filter.Names.IfSome(func(x []string) {
		parameters = append(parameters, pq.Array(x))
		sql += fmt.Sprintf(" and s.name = any($%d::text[])", len(parameters))
	})

	filter.ClientId.IfSome(func(x uuid.UUID) {
		parameters = append(parameters, x)
		sql += fmt.Sprintf(" and (g.client_id = $%d or g.client_id is null)", len(parameters))
	})

	filter.PagingInfo.IfSome(func(x PagingInfo) {
		sql += x.SqlString()
	})

	sql += ` order by "sort_index" asc`

	logging.Logger.Debugf("executing sql: %s", sql)
	rows, err := tx.Query(sql, parameters...)
	if err != nil {
		panic(err)
	}
	defer utils.PanicOnErr(rows.Close)

	var totalCount int
	var result []Scope
	for rows.Next() {
		var row Scope
		var grant Grant
		var grantId *uuid.UUID
		err := rows.Scan(&totalCount,
			&row.Id,
			&row.RealmId,
			&row.Name,
			&row.DisplayName,
			&row.Description,
			&grantId,
			&grant.ScopeId,
			&grant.UserId,
			&grant.ClientId)
		if err != nil {
			panic(err)
		}

		if filter.IncludeGrants && grantId != nil {
			grant.Id = *grantId
			row.Grant = &grant
		}

		result = append(result, row)
	}

	return NewPagedResult(result, totalCount)
}

func (s *ScopeRepositoryImpl) CreateScope(ctx context.Context, scope Scope) h.Result[uuid.UUID] {
	iocScope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](iocScope)

	var resultingId uuid.UUID

	tx, err := rcs.GetTx()
	if err != nil {
		return h.Err[uuid.UUID](err)
	}

	err = tx.QueryRow(`insert into "scopes"
    			("realm_id", "name", "display_name", "description", "sort_index")
    			values ($1, $2, $3, $4, $5)
    			returning "id"`,
		scope.RealmId,
		scope.Name,
		scope.DisplayName,
		scope.Description,
		scope.SortIndex).
		Scan(&resultingId)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code.Name() {
			case "unique_violation":
				if pqErr.Constraint == "idx_unique_scope_name_in_realm" {
					return h.Err[uuid.UUID](DuplicateScopeError{})
				}
				break
			}
		} else {
			panic(err)
		}
	}

	return h.Ok(resultingId)
}

func (s *ScopeRepositoryImpl) CreateGrants(ctx context.Context, userId uuid.UUID, clientId uuid.UUID, scopeIds []uuid.UUID) {
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
		panic(err)
	}
}
