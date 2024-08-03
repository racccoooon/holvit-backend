package repositories

import (
	"context"
	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
	"github.com/jackc/pgtype"
	"holvit/httpErrors"
	"holvit/ioc"
	"holvit/logging"
	"holvit/middlewares"
	"holvit/requestContext"
	"time"
)

type UserDevice struct {
	BaseModel

	UserId uuid.UUID

	DisplayName string
	DeviceId    string
	UserAgent   string
	LastIp      pgtype.Inet
	LastLoginAt time.Time
}

type UserDeviceFilter struct {
	BaseFilter

	DeviceId *string
	UserId   *uuid.UUID
	LastIp   *pgtype.Inet
}

type UserDeviceRepository interface {
	FindUserDeviceById(ctx context.Context, id uuid.UUID) (*UserDevice, error)
	FindUserDevices(ctx context.Context, filter UserDeviceFilter) ([]*UserDevice, int, error)
	CreateUserDevice(ctx context.Context, userDevice *UserDevice) (uuid.UUID, error)
}

func NewUserDeviceRepository() UserDeviceRepository {
	return &UserDeviceRepositoryImpl{}
}

type UserDeviceRepositoryImpl struct{}

func (r *UserDeviceRepositoryImpl) FindUserDeviceById(ctx context.Context, id uuid.UUID) (*UserDevice, error) {
	result, resultCount, err := r.FindUserDevices(ctx, UserDeviceFilter{
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

func (r *UserDeviceRepositoryImpl) FindUserDevices(ctx context.Context, filter UserDeviceFilter) ([]*UserDevice, int, error) {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		return nil, 0, err
	}

	sb := sqlbuilder.Select("count(*) over()",
		"id", "user_id", "device_id", "display_name", "user_agent", "last_ip", "last_login_at").
		From("user_devices")

	if filter.DeviceId != nil {
		sb.Where(sb.Equal("device_id", *filter.DeviceId))
	}

	if filter.UserId != nil {
		sb.Where(sb.Equal("user_id", *filter.UserId))
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
	var result []*UserDevice
	for rows.Next() {
		var row UserDevice
		err := rows.Scan(&totalCount,
			&row.Id,
			&row.UserId,
			&row.DeviceId,
			&row.DisplayName,
			&row.UserAgent,
			&row.LastIp,
			&row.LastLoginAt)
		if err != nil {
			return nil, 0, err
		}
		result = append(result, &row)
	}

	return result, totalCount, nil
}

func (r *UserDeviceRepositoryImpl) CreateUserDevice(ctx context.Context, userDevice *UserDevice) (uuid.UUID, error) {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	var resultingId uuid.UUID

	tx, err := rcs.GetTx()
	if err != nil {
		return resultingId, err
	}

	sqlString := `insert into "user_devices"
    			("user_id", "device_id", "display_name", "user_agent", "last_ip", "last_login_at")
    			values ($1, $2, $3, $4, $5, $6)
    			returning "id"`
	logging.Logger.Debugf("executing sql: %s", sqlString)

	err = tx.QueryRow(sqlString,
		userDevice.UserId,
		userDevice.DeviceId,
		userDevice.DisplayName,
		userDevice.UserAgent,
		userDevice.LastIp,
		userDevice.LastLoginAt).Scan(&resultingId)

	return resultingId, err
}
