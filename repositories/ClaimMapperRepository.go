package repositories

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"holvit/constants"
	"holvit/httpErrors"
	"holvit/ioc"
	"holvit/logging"
	"holvit/middlewares"
	"holvit/requestContext"
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

	RealmId  *uuid.UUID
	ScopeIds []uuid.UUID
}

type AssociateScopeClaimRequest struct {
	ClaimMapperId uuid.UUID
	ScopeId       uuid.UUID
}

type ClaimMapperRepository interface {
	FindClaimMapperById(ctx context.Context, id uuid.UUID) (*ClaimMapper, error)
	FindClaimMappers(ctx context.Context, filter ClaimMapperFilter) ([]*ClaimMapper, int, error)
	CreateClaimMapper(ctx context.Context, claimMapper *ClaimMapper) (uuid.UUID, error)
	AssociateClaimMapper(ctx context.Context, request AssociateScopeClaimRequest) (uuid.UUID, error)
}

type ClaimMapperRepositoryImpl struct{}

func NewClaimMapperRepository() ClaimMapperRepository {
	return &ClaimMapperRepositoryImpl{}
}

func (c *ClaimMapperRepositoryImpl) FindClaimMapperById(ctx context.Context, id uuid.UUID) (*ClaimMapper, error) {
	result, resultCount, err := c.FindClaimMappers(ctx, ClaimMapperFilter{
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

func (c *ClaimMapperRepositoryImpl) FindClaimMappers(ctx context.Context, filter ClaimMapperFilter) ([]*ClaimMapper, int, error) {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		return nil, 0, err
	}

	sqlString := `select count(*) over(), c.id, c.realm_id, c.display_name, c.description, c.type, c.details from claim_mappers c where true`

	args := make([]interface{}, 0)
	if filter.RealmId != nil {
		args = append(args, *filter.RealmId)
		sqlString += fmt.Sprintf(" and c.realm_id = $%d", len(args))
	}

	if len(filter.ScopeIds) > 0 {
		args = append(args, pq.Array(filter.ScopeIds))
		sqlString += fmt.Sprintf(" and exists (select 1 from scope_claims sc where sc.claim_mapper_id = c.id and sc.scope_id = any($%d::uuid[]))", len(args))
	}

	if filter.PagingInfo.PageSize > 0 {
		sqlString += fmt.Sprintf(" limit %d offset %d", filter.PagingInfo.PageSize, filter.PagingInfo.PageSize*(filter.PagingInfo.PageNumber-1))
	}

	logging.Logger.Debugf("executing sql: %s", sqlString)
	rows, err := tx.Query(sqlString, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var totalCount int
	var result []*ClaimMapper
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
			return nil, 0, err
		}

		switch row.Type {
		case constants.ClaimMapperUserInfo:
			var userInfoMapper UserInfoClaimMapperDetails
			err := json.Unmarshal(detailsRaw, &userInfoMapper)
			if err != nil {
				return nil, 0, err
			}
			row.Details = userInfoMapper
			break
		default:
			logging.Logger.Fatalf("Unsupported mapper type %v in claims mapper %v", row.Type, row.Id.String())
		}

		result = append(result, &row)
	}

	return result, totalCount, nil
}

func (c *ClaimMapperRepositoryImpl) CreateClaimMapper(ctx context.Context, claimMapper *ClaimMapper) (uuid.UUID, error) {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	var resultingId uuid.UUID

	tx, err := rcs.GetTx()
	if err != nil {
		return resultingId, nil
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

	return resultingId, err
}

func (c *ClaimMapperRepositoryImpl) AssociateClaimMapper(ctx context.Context, request AssociateScopeClaimRequest) (uuid.UUID, error) {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	var resultingId uuid.UUID

	tx, err := rcs.GetTx()
	if err != nil {
		return resultingId, nil
	}

	sqlString := `insert into "scope_claims"
				("scope_id", "claim_mapper_id")
				values ($1, $2)
				returning "id"`
	logging.Logger.Debugf("executing sql: %s", sqlString)

	err = tx.QueryRow(sqlString,
		request.ScopeId,
		request.ClaimMapperId).Scan(&resultingId)

	return resultingId, err
}
