package repositories

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
	"github.com/lib/pq"
	"holvit/httpErrors"
	"holvit/ioc"
	"holvit/logging"
	"holvit/middlewares"
	"holvit/requestContext"
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

type Grant struct {
	BaseModel

	ScopeId  uuid.UUID
	UserId   uuid.UUID
	ClientId uuid.UUID
}

type ScopeFilter struct {
	BaseFilter

	RealmId uuid.UUID
	Names   []string

	IncludeGrants bool
	OnlyGranted   bool

	UserId   *uuid.UUID
	ClientId *uuid.UUID
}

type ScopeRepository interface {
	FindScopeById(ctx context.Context, id uuid.UUID) (*Scope, error)
	FindScopes(ctx context.Context, filter ScopeFilter) ([]*Scope, int, error)
	CreateScope(ctx context.Context, scope *Scope) (uuid.UUID, error)
	CreateGrants(ctx context.Context, userId uuid.UUID, clientId uuid.UUID, scopeIds []uuid.UUID) error
}

type ScopeRepositoryImpl struct{}

func NewScopeReposiroty() ScopeRepository {
	return &ScopeRepositoryImpl{}
}

func (s *ScopeRepositoryImpl) FindScopeById(ctx context.Context, id uuid.UUID) (*Scope, error) {
	result, resultCount, err := s.FindScopes(ctx, ScopeFilter{
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
	if resultCount == 0 {
		return nil, httpErrors.NotFound()
	}
	return result[0], nil
}

func (s *ScopeRepositoryImpl) FindScopes(ctx context.Context, filter ScopeFilter) ([]*Scope, int, error) {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		return nil, 0, err
	}

	sql := "select count(*) over() as total_count, s.id, s.realm_id, s.name, s.display_name, s.description"

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

	if filter.Names != nil {
		parameters = append(parameters, pq.Array(filter.Names))
		sql += fmt.Sprintf(" and s.name = any($%d::text[])", len(parameters))
	}

	if filter.UserId != nil {
		parameters = append(parameters, filter.UserId)
		sql += fmt.Sprintf(" and (g.user_id = $%d or g.user_id is null)", len(parameters))
	}

	if filter.ClientId != nil {
		parameters = append(parameters, filter.ClientId)
		sql += fmt.Sprintf(" and (g.client_id = $%d or g.client_id is null)", len(parameters))
	}

	sql += ` order by "sort_index" asc`

	logging.Logger.Debugf("executing sql: %s", sql)
	rows, err := tx.Query(sql, parameters...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var totalCount int
	var result []*Scope
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
			return nil, 0, err
		}

		if filter.IncludeGrants && grantId != nil {
			grant.Id = *grantId
			row.Grant = &grant
		}

		result = append(result, &row)
	}

	return result, totalCount, nil
}

func (s *ScopeRepositoryImpl) CreateScope(ctx context.Context, scope *Scope) (uuid.UUID, error) {
	iocScope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](iocScope)

	var resultingId uuid.UUID

	tx, err := rcs.GetTx()
	if err != nil {
		return resultingId, err
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

	return resultingId, err
}

func (s *ScopeRepositoryImpl) CreateGrants(ctx context.Context, userId uuid.UUID, clientId uuid.UUID, scopeIds []uuid.UUID) error {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		return nil
	}

	sb := sqlbuilder.InsertInto("grants").
		Cols("scope_id", "user_id", "client_id")

	for _, scopeId := range scopeIds {
		sb.Values(scopeId, userId, clientId)
	}

	sqlString, args := sb.Build()
	logging.Logger.Debugf("executing sql: %s", sqlString)
	_, err = tx.Exec(sqlString, args...)

	return err
}
