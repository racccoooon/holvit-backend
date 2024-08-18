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

type RoleImplication struct {
	BaseModel

	RoleId        uuid.UUID
	ImpliedRoleId uuid.UUID
}

type RoleImplicationFilter struct {
	RealmId uuid.UUID
}

type RoleImplicationRepository interface {
	FindRoleImplications(ctx context.Context, filter RoleImplicationFilter) []RoleImplication
	DeleteImplicationsForRole(ctx context.Context, id uuid.UUID)
	CreateImplications(ctx context.Context, implications []RoleImplication)
}

func NewRoleImplicationRepository() RoleImplicationRepository {
	return &roleImplicationRepositoryImpl{}
}

type roleImplicationRepositoryImpl struct{}

func (r *roleImplicationRepositoryImpl) CreateImplications(ctx context.Context, implications []RoleImplication) {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		panic(err)
	}

	q := sqlb.InsertInto("role_implications", "role_id", "implied_role_id")

	for _, implication := range implications {
		q.Values(implication.RoleId, implication.ImpliedRoleId)
	}

	query := q.Build()
	logging.Logger.Debugf("executing sql: %s", query.Sql)
	_, err = tx.Exec(query.Sql, query.Parameters...)
	if err != nil {
		panic(mapCustomErrorCodes(err))
	}
}

func (r *roleImplicationRepositoryImpl) DeleteImplicationsForRole(ctx context.Context, id uuid.UUID) {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		panic(err)
	}

	q := sqlb.DeleteFrom("role_implications").
		Where("role_id = ?", id)

	query := q.Build()
	logging.Logger.Debugf("executing sql: %s", query.Sql)
	_, err = tx.Exec(query.Sql, query.Parameters...)
	if err != nil {
		panic(mapCustomErrorCodes(err))
	}
}

func (r *roleImplicationRepositoryImpl) FindRoleImplications(ctx context.Context, filter RoleImplicationFilter) []RoleImplication {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		panic(err)
	}

	q := sqlb.Select("i.id", "i.role_id", "i.implied_role_id").
		From("role_implications i").
		Join("roles r", "r.id = i.role_id").
		Where("r.realm_id = ?", filter.RealmId)

	query := q.Build()
	logging.Logger.Debugf("executing sql: %s", query.Sql)
	rows, err := tx.Query(query.Sql, query.Parameters...)
	if err != nil {
		panic(mapCustomErrorCodes(err))
	}
	defer utils.PanicOnErr(rows.Close)

	var result []RoleImplication
	for rows.Next() {
		var row RoleImplication
		err := rows.Scan(&row.Id,
			row.RoleId,
			&row.ImpliedRoleId)
		if err != nil {
			panic(mapCustomErrorCodes(err))
		}
		result = append(result, row)
	}

	return result
}
