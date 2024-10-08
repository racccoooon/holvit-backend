package repos

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
	"github.com/lib/pq"
	"holvit/constants"
	"holvit/h"
	"holvit/ioc"
	"holvit/logging"
	"holvit/middlewares"
	"holvit/requestContext"
	"holvit/sqlb"
	"holvit/utils"
)

type Credential struct {
	BaseModel

	UserId uuid.UUID

	Type    string
	Details interface{}
}

type DuplicatePasswordOnUserError struct{}

func (DuplicatePasswordOnUserError) Error() string {
	return "Duplicate password"
}

type CredentialPasswordDetails struct {
	HashedPassword string `json:"hashedPassword"`
	Temporary      bool   `json:"temporary"`
}

func (d CredentialPasswordDetails) Value() (driver.Value, error) {
	return json.Marshal(d)
}

func (d *CredentialPasswordDetails) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, &d)
}

type CredentialTotpDetails struct {
	DisplayName           string `json:"displayName"`
	EncryptedSecretBase64 string `json:"encryptedSecretBase64"`
}

func (d CredentialTotpDetails) Value() (driver.Value, error) {
	return json.Marshal(d)
}

func (d *CredentialTotpDetails) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, &d)
}

type CredentialFilter struct {
	BaseFilter
	UserId h.Opt[uuid.UUID]
	Type   h.Opt[string]
}

type CredentialRepository interface {
	CreateCredential(ctx context.Context, credential Credential) h.Result[uuid.UUID]
	FindCredentialById(ctx context.Context, id uuid.UUID) h.Opt[Credential]
	FindCredentials(ctx context.Context, filter CredentialFilter) FilterResult[Credential]
	DeleteCredential(ctx context.Context, id uuid.UUID)
}

type credentialRepositoryImpl struct{}

func NewCredentialRepository() CredentialRepository {
	return &credentialRepositoryImpl{}
}

func (c *credentialRepositoryImpl) CreateCredential(ctx context.Context, credential Credential) h.Result[uuid.UUID] {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	var resultingId uuid.UUID

	tx, err := rcs.GetTx()
	if err != nil {
		return h.Err[uuid.UUID](err)
	}

	q := sqlb.InsertInto("credentials", "user_id", "type", "details").
		Values(credential.UserId,
			credential.Type,
			credential.Details).
		Returning("id")

	query := q.Build()
	logging.Logger.Debugf("executing sql: %s", query.Sql)

	err = tx.QueryRow(query.Sql, query.Parameters...).Scan(&resultingId)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code.Name() {
			case "unique_violation":
				if pqErr.Constraint == "idx_only_one_password_per_user" {
					return h.Err[uuid.UUID](DuplicatePasswordOnUserError{})
				}
			}
		}

		panic(mapCustomErrorCodes(err))
	}

	return h.Ok(resultingId)
}

func (c *credentialRepositoryImpl) FindCredentialById(ctx context.Context, id uuid.UUID) h.Opt[Credential] {
	return c.FindCredentials(ctx, CredentialFilter{
		BaseFilter: BaseFilter{
			Id: h.Some(id),
		},
	}).SingleOrNone()
}

func (c *credentialRepositoryImpl) FindCredentials(ctx context.Context, filter CredentialFilter) FilterResult[Credential] {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		panic(err)
	}

	q := sqlb.Select(filter.CountCol(), "id", "user_id", "type", "details").
		From("credentials")

	filter.Id.IfSome(func(x uuid.UUID) {
		q.Where("id = ?", x)
	})

	filter.UserId.IfSome(func(x uuid.UUID) {
		q.Where("user_id = ?", x)
	})

	filter.Type.IfSome(func(x string) {
		q.Where("type = ?", x)
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
	var result []Credential
	for rows.Next() {
		var row Credential
		var detailsRaw json.RawMessage
		err := rows.Scan(&totalCount,
			&row.Id,
			&row.UserId,
			&row.Type,
			&detailsRaw)
		if err != nil {
			panic(mapCustomErrorCodes(err))
		}

		switch row.Type {
		case constants.CredentialTypePassword:
			row.Details = utils.FromRawMessage[CredentialPasswordDetails](detailsRaw).Unwrap()
		case constants.CredentialTypeTotp:
			row.Details = utils.FromRawMessage[CredentialTotpDetails](detailsRaw).Unwrap()
		default:
			logging.Logger.Fatalf("Unsupported hash algorithm '%v' in password credential '%v'", row.Type, row.Id.String())
		}

		result = append(result, row)
	}

	return NewPagedResult(result, totalCount)
}

func (c *credentialRepositoryImpl) DeleteCredential(ctx context.Context, id uuid.UUID) {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		panic(err)
	}

	sb := sqlbuilder.DeleteFrom("credentials")
	sb.Where(sb.Equal("id", id))

	sqlString, args := sb.Build()
	logging.Logger.Debugf("executing sql: %s", sqlString)
	_, err = tx.Exec(sqlString, args...)
	if err != nil {
		panic(mapCustomErrorCodes(err))
	}
}
