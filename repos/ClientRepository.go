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
)

type Client struct {
	BaseModel

	RealmId uuid.UUID

	DisplayName string

	ClientId     string
	ClientSecret string

	RedirectUris []string
}

type ClientFilter struct {
	BaseFilter

	RealmId  h.Optional[uuid.UUID]
	ClientId h.Optional[string]
}

type ClientUpdate struct {
	DisplayName  h.Optional[string]
	RedirectUris h.Optional[[]string]
}

type ClientRepository interface {
	FindClientById(ctx context.Context, id uuid.UUID) h.Optional[Client]
	FindClients(ctx context.Context, filter ClientFilter) h.Result[FilterResult[Client]]
	CreateClient(ctx context.Context, client *Client) h.Result[uuid.UUID]
	UpdateClient(ctx context.Context, id uuid.UUID, upd *ClientUpdate) error
}

type ClientRepositoryImpl struct{}

func NewClientRepository() ClientRepository {
	return &ClientRepositoryImpl{}
}

func (c *ClientRepositoryImpl) FindClientById(ctx context.Context, id uuid.UUID) h.Optional[Client] {
	return c.FindClients(ctx, ClientFilter{
		BaseFilter: BaseFilter{
			Id:         h.Some(id),
			PagingInfo: h.Some(NewPagingInfo(1, 0)),
		},
	}).Unwrap().First()
}

func (c *ClientRepositoryImpl) FindClients(ctx context.Context, filter ClientFilter) h.Result[FilterResult[Client]] {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		return h.Err[FilterResult[Client]](err)
	}

	sb := sqlbuilder.Select("count(*) over()",
		"id", "realm_id", "display_name", "client_id", "hashed_client_secret", "redirect_uris").
		From("clients")

	filter.Id.IfSome(func(x uuid.UUID) {
		sb.Where(sb.Equal("id", x))
	})

	filter.RealmId.IfSome(func(x uuid.UUID) {
		sb.Where(sb.Equal("realm_id", x))
	})

	filter.ClientId.IfSome(func(x string) {
		sb.Where(sb.Equal("client_id", x))
	})

	filter.PagingInfo.IfSome(func(x PagingInfo) {
		x.Apply(sb)
	})

	sqlString, args := sb.Build()
	logging.Logger.Debugf("executing sql: %s", sqlString)
	rows, err := tx.Query(sqlString, args...)
	if err != nil {
		return h.Err[FilterResult[Client]](err)
	}
	defer rows.Close()

	var totalCount int
	var result []Client
	for rows.Next() {
		var row Client
		err := rows.Scan(&totalCount,
			&row.Id,
			&row.RealmId,
			&row.DisplayName,
			&row.ClientId,
			&row.ClientSecret,
			pq.Array(&row.RedirectUris))
		if err != nil {
			return h.Err[FilterResult[Client]](err)
		}
		result = append(result, row)
	}

	return h.Ok(NewPagedResult(result, totalCount))
}

func (c *ClientRepositoryImpl) CreateClient(ctx context.Context, client *Client) h.Result[uuid.UUID] {
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
		client.ClientSecret,
		pq.Array(client.RedirectUris)).Scan(&resultingId)
	if err != nil {
		return h.Err[uuid.UUID](err)
	}

	return h.Ok(resultingId)
}

func (c *ClientRepositoryImpl) UpdateClient(ctx context.Context, id uuid.UUID, upd *ClientUpdate) error {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		return err
	}

	sb := sqlbuilder.Update("clients")

	upd.DisplayName.IfSome(func(x string) {
		sb.Set(sb.Assign("display_name", x))
	})

	upd.RedirectUris.IfSome(func(x []string) {
		sb.Set(sb.Assign("redirect_uris", x))
	})

	sb.Where(sb.Equal("id", id))

	sqlString, args := sb.Build()
	logging.Logger.Debugf("executing sql: %s", sqlString)
	_, err = tx.Exec(sqlString, args...)
	if err != nil {
		return err
	}

	return nil
}
