package services

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgtype"
	"github.com/mssola/user_agent"
	"holvit/ioc"
	"holvit/middlewares"
	"holvit/repositories"
	"holvit/utils"
)

type IsKnownDeviceRequest struct {
	UserId   uuid.UUID
	DeviceId string
}

type IsKnownDeviceResponse struct {
	IsKnown              bool
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
	IsKnownUserDevice(ctx context.Context, request IsKnownDeviceRequest) (*IsKnownDeviceResponse, error)
	SendVerificationEmail(ctx context.Context, request SendVerificationRequest) (*SendVerificationResponse, error)
	AddKnownDevice(ctx context.Context, request AddDeviceRequest) error
}

func NewDeviceService() DeviceService {
	return &DeviceServiceImpl{}
}

type DeviceServiceImpl struct{}

func (d *DeviceServiceImpl) AddKnownDevice(ctx context.Context, request AddDeviceRequest) error {
	scope := middlewares.GetScope(ctx)

	userDeviceRepository := ioc.Get[repositories.UserDeviceRepository](scope)
	devices, _, err := userDeviceRepository.FindUserDevices(ctx, repositories.UserDeviceFilter{
		UserId:   &request.UserId,
		DeviceId: &request.DeviceId,
	})
	if err != nil {
		return err
	}
	if len(devices) > 0 {
		return nil
	}

	ua := user_agent.New(request.UserAgent)
	browser, browserVersion := ua.Browser()
	displayName := fmt.Sprintf("%s %s", browser, browserVersion)

	clockService := ioc.Get[ClockService](scope)
	now := clockService.Now()

	_, err = userDeviceRepository.CreateUserDevice(ctx, &repositories.UserDevice{
		UserId:      request.UserId,
		DisplayName: displayName,
		DeviceId:    request.DeviceId,
		UserAgent:   request.UserAgent,
		LastIp:      pgtype.Inet{},
		LastLoginAt: now,
	})
	if err != nil {
		return err
	}

	return nil
}

func (d *DeviceServiceImpl) SendVerificationEmail(ctx context.Context, request SendVerificationRequest) (*SendVerificationResponse, error) {
	scope := middlewares.GetScope(ctx)
	jobService := ioc.Get[JobService](scope)

	num, err := utils.GenerateRandomNumber(999_999)
	if err != nil {
		return nil, err
	}
	code := fmt.Sprintf("%d", num)

	err = jobService.QueueJob(ctx, repositories.SendMailJobDetails{
		To:      nil,
		Subject: "Device Verification Code",
		Body:    fmt.Sprintf(`<html><body>Enter the following code:<br/>%v</body></html>`, code),
	})
	if err != nil {
		return nil, err
	}

	return &SendVerificationResponse{
		Code: code,
	}, nil
}

func (d *DeviceServiceImpl) IsKnownUserDevice(ctx context.Context, request IsKnownDeviceRequest) (*IsKnownDeviceResponse, error) {
	scope := middlewares.GetScope(ctx)

	userDeviceRepository := ioc.Get[repositories.UserDeviceRepository](scope)
	devices, _, err := userDeviceRepository.FindUserDevices(ctx, repositories.UserDeviceFilter{
		UserId:   &request.UserId,
		DeviceId: &request.DeviceId,
	})
	if err != nil {
		return nil, err
	}
	if len(devices) > 0 {
		return &IsKnownDeviceResponse{
			IsKnown:              true,
			RequiresVerification: false,
		}, nil
	}

	userRepository := ioc.Get[repositories.UserRepository](scope)
	user, err := userRepository.FindUserById(ctx, request.UserId)
	if err != nil {
		return nil, err
	}

	realmRepository := ioc.Get[repositories.RealmRepository](scope)
	realm, err := realmRepository.FindRealmById(ctx, user.RealmId)
	if err != nil {
		return nil, err
	}

	return &IsKnownDeviceResponse{
		IsKnown:              false,
		RequiresVerification: realm.RequireDeviceVerification,
	}, nil
}
