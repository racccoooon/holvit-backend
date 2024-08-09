package services

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/mssola/user_agent"
	"holvit/h"
	"holvit/ioc"
	"holvit/middlewares"
	"holvit/repos"
	"holvit/utils"
)

type IsKnownDeviceRequest struct {
	UserId   uuid.UUID
	DeviceId string
}

type IsKnownDeviceResponse struct {
	Id                   h.Optional[uuid.UUID]
	RequiresVerification bool
}

type SendVerificationRequest struct {
	UserId    uuid.UUID
	DeviceId  string
	UserAgent string
}

type SendVerificationResponse struct {
	Code string
}

type AddDeviceRequest struct {
	UserId    uuid.UUID
	DeviceId  string
	UserAgent string
	Ip        string
}

type DeviceService interface {
	IsKnownUserDevice(ctx context.Context, request IsKnownDeviceRequest) IsKnownDeviceResponse
	SendVerificationEmail(ctx context.Context, request SendVerificationRequest) SendVerificationResponse
	AddKnownDevice(ctx context.Context, request AddDeviceRequest) uuid.UUID
}

func NewDeviceService() DeviceService {
	return &DeviceServiceImpl{}
}

type DeviceServiceImpl struct{}

func (d *DeviceServiceImpl) AddKnownDevice(ctx context.Context, request AddDeviceRequest) uuid.UUID {
	scope := middlewares.GetScope(ctx)

	userDeviceRepository := ioc.Get[repos.UserDeviceRepository](scope)
	devices := userDeviceRepository.FindUserDevices(ctx, repos.UserDeviceFilter{
		UserId:   h.Some(request.UserId),
		DeviceId: h.Some(request.DeviceId),
	})
	if devices.Count() > 0 {
		return devices.First().Id
	}

	ua := user_agent.New(request.UserAgent)
	browser, browserVersion := ua.Browser()
	displayName := fmt.Sprintf("%s %s", browser, browserVersion)

	clockService := ioc.Get[utils.ClockService](scope)
	now := clockService.Now()

	id := userDeviceRepository.CreateUserDevice(ctx, repos.UserDevice{
		UserId:      request.UserId,
		DisplayName: displayName,
		DeviceId:    request.DeviceId,
		UserAgent:   request.UserAgent,
		LastIp:      utils.InetFromString(request.Ip),
		LastLoginAt: now,
	})

	return id
}

func (d *DeviceServiceImpl) SendVerificationEmail(ctx context.Context, request SendVerificationRequest) SendVerificationResponse {
	scope := middlewares.GetScope(ctx)
	jobService := ioc.Get[JobService](scope)

	num := utils.GenerateRandomNumber(999_999)
	code := fmt.Sprintf("%d", num)

	//TODO: save the code in redis

	jobService.QueueJob(ctx, repos.SendMailJobDetails{
		To:      nil,
		Subject: "Device Verification Code",
		Body:    fmt.Sprintf(`<html><body>Enter the following code:<br/>%v</body></html>`, code),
	})

	return SendVerificationResponse{
		Code: code,
	}
}

func (d *DeviceServiceImpl) IsKnownUserDevice(ctx context.Context, request IsKnownDeviceRequest) IsKnownDeviceResponse {
	scope := middlewares.GetScope(ctx)

	userDeviceRepository := ioc.Get[repos.UserDeviceRepository](scope)
	devices := userDeviceRepository.FindUserDevices(ctx, repos.UserDeviceFilter{
		UserId:   h.Some(request.UserId),
		DeviceId: h.Some(request.DeviceId),
	})
	if devices.Count() > 0 {
		return IsKnownDeviceResponse{
			RequiresVerification: false,
			Id:                   h.Some(devices.Values()[0].Id),
		}
	}

	userRepository := ioc.Get[repos.UserRepository](scope)
	user := userRepository.FindUserById(ctx, request.UserId).Unwrap()

	realmRepository := ioc.Get[repos.RealmRepository](scope)
	realm := realmRepository.FindRealmById(ctx, user.RealmId).Unwrap()

	return IsKnownDeviceResponse{
		RequiresVerification: realm.RequireDeviceVerification,
		Id:                   h.None[uuid.UUID](),
	}
}
