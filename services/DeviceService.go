package services

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgtype"
	"github.com/mssola/user_agent"
	"holvit/h"
	"holvit/ioc"
	"holvit/middlewares"
	"holvit/repos"
	"holvit/utils"
	"strings"
)

type IsKnownDeviceRequest struct {
	UserId   uuid.UUID
	DeviceId string
}

type IsKnownDeviceResponse struct {
	IsKnown              bool
	RequiresVerification bool
	Id                   *uuid.UUID
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
	AddKnownDevice(ctx context.Context, request AddDeviceRequest) (*uuid.UUID, error)
}

func NewDeviceService() DeviceService {
	return &DeviceServiceImpl{}
}

type DeviceServiceImpl struct{}

func (d *DeviceServiceImpl) AddKnownDevice(ctx context.Context, request AddDeviceRequest) (*uuid.UUID, error) {
	scope := middlewares.GetScope(ctx)

	userDeviceRepository := ioc.Get[repos.UserDeviceRepository](scope)
	devices := userDeviceRepository.FindUserDevices(ctx, repos.UserDeviceFilter{
		UserId:   h.Some(request.UserId),
		DeviceId: h.Some(request.DeviceId),
	}).Unwrap()
	if devices.Count() > 0 {
		return utils.Ptr(devices.First().Id), nil
	}

	ua := user_agent.New(request.UserAgent)
	browser, browserVersion := ua.Browser()
	displayName := fmt.Sprintf("%s %s", browser, browserVersion)

	clockService := ioc.Get[utils.ClockService](scope)
	now := clockService.Now()

	lastIp := pgtype.Inet{}
	ipStr := request.Ip
	// Remove the port if it exists
	if colonIndex := strings.LastIndex(ipStr, ":"); colonIndex != -1 {
		ipStr = ipStr[:colonIndex] // Remove port part
	}
	if strings.HasPrefix(ipStr, "[") && strings.HasSuffix(ipStr, "]") {
		ipStr = ipStr[1 : len(ipStr)-1] // Remove square brackets for IPv6
	}

	err := lastIp.Set(ipStr)
	if err != nil {
		return nil, err
	}

	id := userDeviceRepository.CreateUserDevice(ctx, &repos.UserDevice{
		UserId:      request.UserId,
		DisplayName: displayName,
		DeviceId:    request.DeviceId,
		UserAgent:   request.UserAgent,
		LastIp:      lastIp,
		LastLoginAt: now,
	}).Unwrap()

	return &id, nil
}

func (d *DeviceServiceImpl) SendVerificationEmail(ctx context.Context, request SendVerificationRequest) (*SendVerificationResponse, error) {
	scope := middlewares.GetScope(ctx)
	jobService := ioc.Get[JobService](scope)

	num, err := utils.GenerateRandomNumber(999_999)
	if err != nil {
		return nil, err
	}
	code := fmt.Sprintf("%d", num)

	err = jobService.QueueJob(ctx, repos.SendMailJobDetails{
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

	userDeviceRepository := ioc.Get[repos.UserDeviceRepository](scope)
	devices := userDeviceRepository.FindUserDevices(ctx, repos.UserDeviceFilter{
		UserId:   h.Some(request.UserId),
		DeviceId: h.Some(request.DeviceId),
	}).Unwrap()
	if devices.Count() > 0 {
		return &IsKnownDeviceResponse{
			IsKnown:              true,
			RequiresVerification: false,
			Id:                   utils.Ptr(devices.Values()[0].Id),
		}, nil
	}

	userRepository := ioc.Get[repos.UserRepository](scope)
	user := userRepository.FindUserById(ctx, request.UserId).Unwrap()

	realmRepository := ioc.Get[repos.RealmRepository](scope)
	realm := realmRepository.FindRealmById(ctx, user.RealmId).Unwrap()

	return &IsKnownDeviceResponse{
		IsKnown:              false,
		RequiresVerification: realm.RequireDeviceVerification,
		Id:                   nil,
	}, nil
}
