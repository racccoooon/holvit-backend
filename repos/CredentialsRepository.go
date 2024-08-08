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
	UserId h.Optional[uuid.UUID]
	Type   h.Optional[string]
}

type CredentialRepository interface {
	CreateCredential(ctx context.Context, credential *Credential) h.Result[uuid.UUID]
	FindCredentialById(ctx context.Context, id uuid.UUID) h.Optional[Credential]
	FindCredentials(ctx context.Context, filter CredentialFilter) FilterResult[Credential]
	DeleteCredential(ctx context.Context, id uuid.UUID) h.Result[h.Unit]
}

type CredentialRepositoryImpl struct{}

func NewCredentialRepository() CredentialRepository {
	return &CredentialRepositoryImpl{}
}

func (c *CredentialRepositoryImpl) CreateCredential(ctx context.Context, credential *Credential) h.Result[uuid.UUID] {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	var resultingId uuid.UUID

	tx, err := rcs.GetTx()
	if err != nil {
		return h.Err[uuid.UUID](err)
	}

	sqlString := `insert into "credentials"
    			("user_id", "type", "details")
    			values ($1, $2, $3)
    			returning "id"`
	logging.Logger.Debugf("executing sql: %s", sqlString)

	err = tx.QueryRow(sqlString,
		credential.UserId,
		credential.Type,
		credential.Details).Scan(&resultingId)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code.Name() {
			case "unique_violation":
				if pqErr.Constraint == "idx_only_one_password_per_user" {
					return h.Err[uuid.UUID](DuplicatePasswordOnUserError{})
				}
				break
			}
		} else {
			panic(err)
		}
	}

	return h.Ok(resultingId)
}

func (c *CredentialRepositoryImpl) FindCredentialById(ctx context.Context, id uuid.UUID) h.Optional[Credential] {
	return c.FindCredentials(ctx, CredentialFilter{
		BaseFilter: BaseFilter{
			Id: h.Some(id),
		},
	}).FirstOrNone()
}

func (c *CredentialRepositoryImpl) FindCredentials(ctx context.Context, filter CredentialFilter) FilterResult[Credential] {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		panic(err)
	}

	sb := sqlbuilder.Select(filter.CountCol(), "id", "user_id", "type", "details").
		From("credentials")

	filter.Id.IfSome(func(x uuid.UUID) {
		sb.Where(sb.Equal("id", x))
	})

	filter.UserId.IfSome(func(x uuid.UUID) {
		sb.Where(sb.Equal("user_id", x))
	})

	filter.Type.IfSome(func(x string) {
		sb.Where(sb.Equal("type", x))
	})

	filter.PagingInfo.IfSome(func(x PagingInfo) {
		x.Apply(sb)
	})

	sqlString, args := sb.Build()
	logging.Logger.Debugf("executing sql: %s", sqlString)
	rows, err := tx.Query(sqlString, args...)
	if err != nil {
		panic(err)
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
			panic(err)
		}

		switch row.Type {
		case constants.CredentialTypePassword:
			row.Details = utils.FromRawMessage[CredentialPasswordDetails](detailsRaw).Unwrap()
			break
		case constants.CredentialTypeTotp:
			row.Details = utils.FromRawMessage[CredentialTotpDetails](detailsRaw).Unwrap()
			break
		default:
			logging.Logger.Fatalf("Unsupported hash algorithm '%v' in password credential '%v'", row.Type, row.Id.String())
		}

		result = append(result, row)
	}

	return NewPagedResult(result, totalCount)
}

func (c *CredentialRepositoryImpl) DeleteCredential(ctx context.Context, id uuid.UUID) h.Result[h.Unit] {
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
	result, err := tx.Exec(sqlString, args...)
	if err != nil {
		panic(err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		panic(err)
	}
	if affected == 0 {
		return h.UErr(DbNotFoundError{})
	}

	return h.UOk()
}
