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

type Client struct {
	BaseModel

	RealmId uuid.UUID

	DisplayName string

	ClientId     string
	ClientSecret h.Opt[string]

	RedirectUris []string
}

type DuplicateClientIdError struct{}

func (e DuplicateClientIdError) Error() string {
	return "Duplicate client id"
}

type ClientFilter struct {
	BaseFilter

	RealmId  h.Opt[uuid.UUID]
	ClientId h.Opt[string]
}

type ClientUpdate struct {
	DisplayName  h.Opt[string]
	RedirectUris h.Opt[[]string]
	ClientSecret h.Opt[string]
}

type ClientRepository interface {
	FindClientById(ctx context.Context, id uuid.UUID) h.Opt[Client]
	FindClients(ctx context.Context, filter ClientFilter) FilterResult[Client]
	CreateClient(ctx context.Context, client Client) h.Result[uuid.UUID]
	UpdateClient(ctx context.Context, id uuid.UUID, upd ClientUpdate) h.UResult
}

type clientRepositoryImpl struct{}

func NewClientRepository() ClientRepository {
	return &clientRepositoryImpl{}
}

func (c *clientRepositoryImpl) FindClientById(ctx context.Context, id uuid.UUID) h.Opt[Client] {
	return c.FindClients(ctx, ClientFilter{
		BaseFilter: BaseFilter{
			Id: h.Some(id),
		},
	}).SingleOrNone()
}

func (c *clientRepositoryImpl) FindClients(ctx context.Context, filter ClientFilter) FilterResult[Client] {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		panic(err)
	}

	q := sqlb.Select(filter.CountCol(),
		"id", "realm_id", "display_name", "client_id", "hashed_client_secret", "redirect_uris").
		From("clients")

	filter.Id.IfSome(func(x uuid.UUID) {
		q.Where("id = ?", x)
	})

	filter.RealmId.IfSome(func(x uuid.UUID) {
		q.Where("realm_id = ?", x)
	})

	filter.ClientId.IfSome(func(x string) {
		q.Where("client_id = ?", x)
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
	var result []Client
	for rows.Next() {
		var row Client
		err := rows.Scan(&totalCount,
			&row.Id,
			&row.RealmId,
			&row.DisplayName,
			&row.ClientId,
			row.ClientSecret.AsMutPtr(),
			pq.Array(&row.RedirectUris))
		if err != nil {
			panic(mapCustomErrorCodes(err))
		}
		result = append(result, row)
	}

	return NewPagedResult(result, totalCount)
}

func (c *clientRepositoryImpl) CreateClient(ctx context.Context, client Client) h.Result[uuid.UUID] {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	var resultingId uuid.UUID

	tx, err := rcs.GetTx()
	if err != nil {
		return h.Err[uuid.UUID](err)
	}

	err = tx.QueryRow(`insert into "clients"
    			("realm_id", "display_name", "client_id", "hashed_client_secret", "redirect_uris")
    			values ($1, $2, $3, $4, $5)
    			returning "id"`,
		client.RealmId,
		client.DisplayName,
		client.ClientId,
		client.ClientSecret.AsMutPtr(),
		pq.Array(client.RedirectUris)).Scan(&resultingId)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code.Name() {
			case "unique_violation":
				if pqErr.Constraint == "idx_unique_client_id_per_realm" {
					return h.Err[uuid.UUID](DuplicateClientIdError{})
				}
			}
		}

		panic(mapCustomErrorCodes(err))
	}

	return h.Ok(resultingId)
}

func (c *clientRepositoryImpl) UpdateClient(ctx context.Context, id uuid.UUID, upd ClientUpdate) h.UResult {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		panic(err)
	}

	sb := sqlbuilder.Update("clients")

	upd.DisplayName.IfSome(func(x string) {
		sb.Set(sb.Assign("display_name", x))
	})

	upd.RedirectUris.IfSome(func(x []string) {
		sb.Set(sb.Assign("redirect_uris", x))
	})

	upd.ClientSecret.IfSome(func(x string) {
		sb.Set(sb.Assign("hashed_client_secret", x))
	})

	sb.Where(sb.Equal("id", id))

	sqlString, args := sb.Build()
	logging.Logger.Debugf("executing sql: %s", sqlString)
	_, err = tx.Exec(sqlString, args...)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code.Name() {
			case "unique_violation":
				if pqErr.Constraint == "idx_unique_client_id_per_realm" {
					return h.UErr(DuplicateClientIdError{})
				}
			}
		}

		panic(mapCustomErrorCodes(err))
	}

	return h.UOk()
}
