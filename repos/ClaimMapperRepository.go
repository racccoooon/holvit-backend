package repos

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"holvit/constants"
	"holvit/h"
	"holvit/ioc"
	"holvit/logging"
	"holvit/middlewares"
	"holvit/requestContext"
	"holvit/utils"
)

type ClaimMapper struct {
	BaseModel

	RealmId uuid.UUID

	DisplayName string
	Description string

	Type    string
	Details interface{}
}

type UserInfoClaimMapperDetails struct {
	ClaimName string
	Property  string
}

func (c UserInfoClaimMapperDetails) Value() (driver.Value, error) {
	return json.Marshal(c)
}

func (c *UserInfoClaimMapperDetails) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, &c)
}

type ClaimMapperFilter struct {
	BaseFilter

	RealmId  h.Opt[uuid.UUID]
	ScopeIds h.Opt[[]uuid.UUID]
}

type AssociateScopeClaimRequest struct {
	ClaimMapperId uuid.UUID
	ScopeId       uuid.UUID
}

type ClaimMapperRepository interface {
	FindClaimMapperById(ctx context.Context, id uuid.UUID) h.Opt[ClaimMapper]
	FindClaimMappers(ctx context.Context, filter ClaimMapperFilter) FilterResult[ClaimMapper]
	CreateClaimMapper(ctx context.Context, claimMapper ClaimMapper) uuid.UUID
	AssociateClaimMapper(ctx context.Context, request AssociateScopeClaimRequest) uuid.UUID
}

type ClaimMapperRepositoryImpl struct{}

func NewClaimMapperRepository() ClaimMapperRepository {
	return &ClaimMapperRepositoryImpl{}
}

func (c *ClaimMapperRepositoryImpl) FindClaimMapperById(ctx context.Context, id uuid.UUID) h.Opt[ClaimMapper] {
	return c.FindClaimMappers(ctx, ClaimMapperFilter{
		BaseFilter: BaseFilter{
			Id: h.Some(id),
		},
	}).FirstOrNone()
}

func (c *ClaimMapperRepositoryImpl) FindClaimMappers(ctx context.Context, filter ClaimMapperFilter) FilterResult[ClaimMapper] {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		panic(err)
	}

	sqlString := `select ` + filter.CountCol() + `, c.id, c.realm_id, c.display_name, c.description, c.type, c.details from claim_mappers c where true`

	args := make([]interface{}, 0)
	filter.Id.IfSome(func(x uuid.UUID) {
		args = append(args, x)
		sqlString += fmt.Sprintf(" and c.realm_id = $%d", len(args))
	})

	filter.Id.IfSome(func(x uuid.UUID) {
		args = append(args, x)
		sqlString += fmt.Sprintf(" and c.id = $%d", len(args))
	})

	filter.ScopeIds.IfSome(func(x []uuid.UUID) {
		args = append(args, pq.Array(x))
		sqlString += fmt.Sprintf(" and exists (select 1 from scope_claims sc where sc.claim_mapper_id = c.id and sc.scope_id = any($%d::uuid[]))", len(args))
	})

	filter.PagingInfo.IfSome(func(x PagingInfo) {
		sqlString += x.SqlString()
	})

	filter.SortInfo.IfSome(func(x SortInfo) {
		sqlString += x.SqlString()
	})

	logging.Logger.Debugf("executing sql: %s", sqlString)
	rows, err := tx.Query(sqlString, args...)
	if err != nil {
		panic(err)
	}
	defer utils.PanicOnErr(rows.Close)

	var totalCount int
	var result []ClaimMapper
	for rows.Next() {
		var row ClaimMapper
		var detailsRaw json.RawMessage
		err := rows.Scan(&totalCount,
			&row.Id,
			&row.RealmId,
			&row.DisplayName,
			&row.Description,
			&row.Type,
			&detailsRaw)
		if err != nil {
			panic(err)
		}

		switch row.Type {
		case constants.ClaimMapperUserInfo:
			row.Details = utils.FromRawMessage[UserInfoClaimMapperDetails](detailsRaw).Unwrap()
		default:
			logging.Logger.Fatalf("Unsupported mapper type '%v' in claims mapper '%v'", row.Type, row.Id.String())
		}

		result = append(result, row)
	}

	return NewPagedResult(result, totalCount)
}

func (c *ClaimMapperRepositoryImpl) CreateClaimMapper(ctx context.Context, claimMapper ClaimMapper) uuid.UUID {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	var resultingId uuid.UUID

	tx, err := rcs.GetTx()
	if err != nil {
		panic(err)
	}

	sqlString := `insert into "claim_mappers"
				("realm_id", "display_name", "description", "type", "details")
				values ($1, $2, $3, $4, $5)
				returning "id"`
	logging.Logger.Debugf("executing sql: %s", sqlString)

	err = tx.QueryRow(sqlString,
		claimMapper.RealmId,
		claimMapper.DisplayName,
		claimMapper.Description,
		claimMapper.Type,
		claimMapper.Details).Scan(&resultingId)
	if err != nil {
		panic(err)
	}

	return resultingId
}

func (c *ClaimMapperRepositoryImpl) AssociateClaimMapper(ctx context.Context, request AssociateScopeClaimRequest) uuid.UUID {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	var resultingId uuid.UUID

	tx, err := rcs.GetTx()
	if err != nil {
		panic(err)
	}

	sqlString := `insert into "scope_claims"
				("scope_id", "claim_mapper_id")
				values ($1, $2)
				returning "id"`
	logging.Logger.Debugf("executing sql: %s", sqlString)

	err = tx.QueryRow(sqlString,
		request.ScopeId,
		request.ClaimMapperId).Scan(&resultingId)
	if err != nil {
		panic(err)
	}

	return resultingId
}
