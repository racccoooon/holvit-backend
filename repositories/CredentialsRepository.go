package repositories

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
	"holvit/constants"
	"holvit/httpErrors"
	"holvit/ioc"
	"holvit/logging"
	"holvit/middlewares"
	"holvit/requestContext"
)

type Credential struct {
	BaseModel

	UserId uuid.UUID

	Type    string
	Details interface{}
}

type CredentialPasswordDetails struct {
	HashedPassword string `json:"hashed_password"`
	Temporary      bool   `json:"temporary"`
}

func (c CredentialPasswordDetails) Value() (driver.Value, error) {
	return json.Marshal(c)
}

func (c *CredentialPasswordDetails) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, &c)
}

type CredentialFilter struct {
	BaseFilter
	UserId *uuid.UUID
	Type   *string
}

type CredentialUpdate struct {
	Type    string
	Details interface{}
}

type CredentialRepository interface {
	CreateCredential(ctx context.Context, credential *Credential) (uuid.UUID, error)
	FindCredentialById(ctx context.Context, id uuid.UUID) (*Credential, error)
	FindCredentials(ctx context.Context, filter CredentialFilter) ([]*Credential, int, error)
	DeleteCredential(ctx context.Context, id uuid.UUID) error
}

type CredentialRepositoryImpl struct{}

func NewCredentialRepository() CredentialRepository {
	return &CredentialRepositoryImpl{}
}

func (c *CredentialRepositoryImpl) CreateCredential(ctx context.Context, credential *Credential) (uuid.UUID, error) {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	var resultingId uuid.UUID

	tx, err := rcs.GetTx()
	if err != nil {
		return resultingId, err
	}

	err = tx.QueryRow(`insert into "credentials"
    			("user_id", "type", "details")
    			values ($1, $2, $3)
    			returning "id"`,
		credential.UserId,
		credential.Type,
		credential.Details).
		Scan(&resultingId)

	return resultingId, err
}

func (c *CredentialRepositoryImpl) FindCredentialById(ctx context.Context, id uuid.UUID) (*Credential, error) {
	credentials, resultCount, err := c.FindCredentials(ctx, CredentialFilter{
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
	if resultCount != len(credentials) {
		return nil, httpErrors.NotFound()
	}
	return credentials[0], nil
}

func (c *CredentialRepositoryImpl) FindCredentials(ctx context.Context, filter CredentialFilter) ([]*Credential, int, error) {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		return nil, 0, err
	}

	sb := sqlbuilder.Select("count(*) over()", "id", "user_id", "type", "details").
		From("credentials")

	if filter.UserId != nil {
		sb.Where(sb.Equal("user_id", *filter.UserId))
	}

	if filter.Type != nil {
		sb.Where(sb.Equal("type", *filter.Type))
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
	var result []*Credential
	for rows.Next() {
		var row Credential
		var detailsRaw json.RawMessage
		err := rows.Scan(&totalCount,
			&row.Id,
			&row.UserId,
			&row.Type,
			&detailsRaw)
		if err != nil {
			return nil, 0, err
		}

		switch row.Type {
		case constants.CredentialTypePassword:
			var passwordDetails CredentialPasswordDetails
			err := json.Unmarshal(detailsRaw, &passwordDetails)
			if err != nil {
				return nil, 0, err
			}
			row.Details = passwordDetails
			break
		default:
			logging.Logger.Fatalf("Unsupported hash algorithm %v in password credential %v", row.Type, row.Id.String())
		}

		result = append(result, &row)
	}

	return result, totalCount, nil
}

func (c *CredentialRepositoryImpl) DeleteCredential(ctx context.Context, id uuid.UUID) error {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		return err
	}

	sb := sqlbuilder.DeleteFrom("credentials")
	sb.Where(sb.Equal("id", id))

	sqlString, args := sb.Build()
	logging.Logger.Debugf("executing sql: %s", sqlString)
	_, err = tx.Exec(sqlString, args...)
	if err != nil {
		return err
	}

	return nil
}
