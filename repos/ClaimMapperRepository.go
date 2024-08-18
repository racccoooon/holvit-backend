package repos

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
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

type ClaimMapper struct {
	BaseModel

	RealmId uuid.UUID

	DisplayName string
	Description string

	Type    string
	Details interface{}
}

type RolesClaimMapperDetails struct {
	ClaimName string
}

func (c RolesClaimMapperDetails) Value() (driver.Value, error) {
	return json.Marshal(c)
}

func (c *RolesClaimMapperDetails) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, &c)
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

type claimMapperRepositoryImpl struct{}

func NewClaimMapperRepository() ClaimMapperRepository {
	return &claimMapperRepositoryImpl{}
}

func (c *claimMapperRepositoryImpl) FindClaimMapperById(ctx context.Context, id uuid.UUID) h.Opt[ClaimMapper] {
	return c.FindClaimMappers(ctx, ClaimMapperFilter{
		BaseFilter: BaseFilter{
			Id: h.Some(id),
		},
	}).SingleOrNone()
}

func (c *claimMapperRepositoryImpl) FindClaimMappers(ctx context.Context, filter ClaimMapperFilter) FilterResult[ClaimMapper] {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		panic(err)
	}

	q := sqlb.Select(filter.CountCol(), "c.id", "c.realm_id", "c.display_name", "c.description", "c.type", "c.details").From("claim_mappers c")

	filter.Id.IfSome(func(x uuid.UUID) {
		q.Where("c.realm_id = ?", x)
	})

	filter.Id.IfSome(func(x uuid.UUID) {
		q.Where("c.id = ?", x)
	})

	filter.ScopeIds.IfSome(func(x []uuid.UUID) {
		q.Where(sqlb.Exists(sqlb.Select("1").
			From("scope_claims sc").
			Where("sc.claim_mapper_id = c.id").
			Where("sc.scope_id = any(?::uuid[])", pq.Array(x))))
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
			panic(mapCustomErrorCodes(err))
		}

		switch row.Type {
		case constants.ClaimMapperUserInfo:
			row.Details = utils.FromRawMessage[UserInfoClaimMapperDetails](detailsRaw).Unwrap()
		case constants.ClaimMapperRoles:
			row.Details = utils.FromRawMessage[RolesClaimMapperDetails](detailsRaw).Unwrap()
		default:
			logging.Logger.Fatalf("Unsupported mapper type '%v' in claims mapper '%v'", row.Type, row.Id.String())
		}

		result = append(result, row)
	}

	return NewPagedResult(result, totalCount)
}

func (c *claimMapperRepositoryImpl) CreateClaimMapper(ctx context.Context, claimMapper ClaimMapper) uuid.UUID {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	var resultingId uuid.UUID

	tx, err := rcs.GetTx()
	if err != nil {
		panic(err)
	}

	q := sqlb.InsertInto("claim_mappers", "realm_id", "display_name", "description", "type", "details").
		Values(claimMapper.RealmId,
			claimMapper.DisplayName,
			claimMapper.Description,
			claimMapper.Type,
			claimMapper.Details).
		Returning("id")

	query := q.Build()
	logging.Logger.Debugf("executing sql: %s", query.Sql)
	err = tx.QueryRow(query.Sql, query.Parameters...).Scan(&resultingId)
	if err != nil {
		panic(mapCustomErrorCodes(err))
	}

	return resultingId
}

func (c *claimMapperRepositoryImpl) AssociateClaimMapper(ctx context.Context, request AssociateScopeClaimRequest) uuid.UUID {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	var resultingId uuid.UUID

	tx, err := rcs.GetTx()
	if err != nil {
		panic(err)
	}

	q := sqlb.InsertInto("scope_claims", "scope_id", "claim_mapper_id").
		Values(request.ScopeId,
			request.ClaimMapperId).
		Returning("id")

	query := q.Build()
	logging.Logger.Debugf("executing sql: %s", query.Sql)
	err = tx.QueryRow(query.Sql, query.Parameters...).Scan(&resultingId)
	if err != nil {
		panic(mapCustomErrorCodes(err))
	}

	return resultingId
}
