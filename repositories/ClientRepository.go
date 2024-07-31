package repositories

import (
	"context"
	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
	"github.com/lib/pq"
	"holvit/httpErrors"
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
	ClientSecret []byte

	RedirectUris []string
}

type ClientFilter struct {
	BaseFilter

	RealmId  *uuid.UUID
	ClientId *string
}

type ClientUpdate struct {
	DisplayName  *string
	RedirectUris *[]string
}

type ClientRepository interface {
	FindClientById(ctx context.Context, id uuid.UUID) (*Client, error)
	FindClients(ctx context.Context, filter ClientFilter) ([]*Client, int, error)
	CreateClient(ctx context.Context, client *Client) (uuid.UUID, error)
	UpdateClient(ctx context.Context, id uuid.UUID, upd *ClientUpdate) error
}

type ClientRepositoryImpl struct{}

func NewClientRepository() ClientRepository {
	return &ClientRepositoryImpl{}
}

func (c *ClientRepositoryImpl) FindClientById(ctx context.Context, id uuid.UUID) (*Client, error) {
	result, resultCount, err := c.FindClients(ctx, ClientFilter{
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

func (c *ClientRepositoryImpl) FindClients(ctx context.Context, filter ClientFilter) ([]*Client, int, error) {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		return nil, 0, err
	}

	sb := sqlbuilder.Select("count(*) over()",
		"id", "realm_id", "display_name", "client_id", "client_secret", "redirect_uris").
		From("clients")

	if filter.ClientId != nil {
		sb.Where(sb.Equal("client_id", *filter.ClientId))
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
	var result []*Client
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
			return nil, 0, err
		}
		result = append(result, &row)
	}

	return result, totalCount, nil
}

func (c *ClientRepositoryImpl) CreateClient(ctx context.Context, client *Client) (uuid.UUID, error) {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	var resultingId uuid.UUID

	tx, err := rcs.GetTx()
	if err != nil {
		return resultingId, err
	}

	err = tx.QueryRow(`insert into "clients"
    			("realm_id", "display_name", "client_id", "client_secret", "redirect_uris")
    			values ($1, $2, $3, $4, $5)
    			returning "id"`,
		client.RealmId,
		client.DisplayName,
		client.ClientId,
		client.ClientSecret,
		pq.Array(client.RedirectUris)).
		Scan(&resultingId)

	return resultingId, err
}

func (c *ClientRepositoryImpl) UpdateClient(ctx context.Context, id uuid.UUID, upd *ClientUpdate) error {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		return err
	}

	sb := sqlbuilder.Update("clients")

	if upd.DisplayName != nil {
		sb.Set(sb.Assign("display_name", *upd.DisplayName))
	}

	if upd.RedirectUris != nil {
		sb.Set(sb.Assign("redirect_uris", *upd.RedirectUris))
	}

	sb.Where(sb.Equal("id", id))

	sqlString, args := sb.Build()
	logging.Logger.Debugf("executing sql: %s", sqlString)
	_, err = tx.Exec(sqlString, args...)
	if err != nil {
		return err
	}

	return nil
}
