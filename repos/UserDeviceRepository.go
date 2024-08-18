package repos

import (
	"context"
	"github.com/google/uuid"
	"github.com/jackc/pgtype"
	"holvit/h"
	"holvit/ioc"
	"holvit/logging"
	"holvit/middlewares"
	"holvit/requestContext"
	"holvit/sqlb"
	"holvit/utils"
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

	DeviceId h.Opt[string]
	UserId   h.Opt[uuid.UUID]
}

type UserDeviceRepository interface {
	FindUserDeviceById(ctx context.Context, id uuid.UUID) h.Opt[UserDevice]
	FindUserDevices(ctx context.Context, filter UserDeviceFilter) FilterResult[UserDevice]
	CreateUserDevice(ctx context.Context, userDevice UserDevice) uuid.UUID
}

func NewUserDeviceRepository() UserDeviceRepository {
	return &userDeviceRepositoryImpl{}
}

type userDeviceRepositoryImpl struct{}

func (r *userDeviceRepositoryImpl) FindUserDeviceById(ctx context.Context, id uuid.UUID) h.Opt[UserDevice] {
	return r.FindUserDevices(ctx, UserDeviceFilter{
		BaseFilter: BaseFilter{
			Id: h.Some(id),
		},
	}).FirstOrNone()
}

func (r *userDeviceRepositoryImpl) FindUserDevices(ctx context.Context, filter UserDeviceFilter) FilterResult[UserDevice] {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	tx, err := rcs.GetTx()
	if err != nil {
		panic(err)
	}

	q := sqlb.Select(filter.CountCol(),
		"id", "user_id", "device_id", "display_name", "user_agent", "last_ip", "last_login_at").
		From("user_devices")

	filter.Id.IfSome(func(x uuid.UUID) {
		q.Where("id = ?", x)
	})

	filter.DeviceId.IfSome(func(x string) {
		q.Where("device_id = ?", x)
	})

	filter.UserId.IfSome(func(x uuid.UUID) {
		q.Where("user_id = ?", x)
	})

	filter.PagingInfo.IfSome(func(x PagingInfo) {
		x.Apply(q)
	})

	filter.SortInfo.IfSome(func(x SortInfo) {
		x.Apply(q)
	})

	query := q.Build()
	logging.Logger.Debugf("executing sql: %s", query.Query)
	rows, err := tx.Query(query.Query, query.Parameters...)
	if err != nil {
		panic(mapCustomErrorCodes(err))
	}
	defer utils.PanicOnErr(rows.Close)

	var totalCount int
	var result []UserDevice
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
			panic(mapCustomErrorCodes(err))
		}
		result = append(result, row)
	}

	return NewPagedResult(result, totalCount)
}

func (r *userDeviceRepositoryImpl) CreateUserDevice(ctx context.Context, userDevice UserDevice) uuid.UUID {
	scope := middlewares.GetScope(ctx)
	rcs := ioc.Get[requestContext.RequestContextService](scope)

	var resultingId uuid.UUID

	tx, err := rcs.GetTx()
	if err != nil {
		panic(err)
	}

	q := sqlb.InsertInto("user_devices", "user_id", "device_id", "display_name", "user_agent", "last_ip", "last_login_at").
		Values(userDevice.UserId,
			userDevice.DeviceId,
			userDevice.DisplayName,
			userDevice.UserAgent,
			userDevice.LastIp,
			userDevice.LastLoginAt).
		Returning("id")

	query := q.Build()
	logging.Logger.Debugf("executing sql: %s", query.Query)
	err = tx.QueryRow(query.Query, query.Parameters...).Scan(&resultingId)
	if err != nil {
		panic(mapCustomErrorCodes(err))
	}

	return resultingId
}
