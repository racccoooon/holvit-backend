package services

import (
	"context"
	"github.com/google/uuid"
	"holvit/httpErrors"
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

type SessionService interface {
	ValidateSession(ctx context.Context, token string) (*repositories.Session, error)
	IsKnownUserDevice(ctx context.Context, request IsKnownDeviceRequest) (*IsKnownDeviceResponse, error)
}

func NewSessionService() SessionService {
	return &SessionServiceImpl{}
}

type SessionServiceImpl struct{}

func (s *SessionServiceImpl) ValidateSession(ctx context.Context, token string) (*repositories.Session, error) {
	scope := middlewares.GetScope(ctx)

	clockService := ioc.Get[ClockService](scope)
	now := clockService.Now()

	hashedToken := utils.CheapHash(token)

	sessionRepository := ioc.Get[repositories.SessionRepository](scope)
	sessions, count, err := sessionRepository.FindSessions(ctx, repositories.SessionFilter{
		HashedToken: &hashedToken,
	})
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, httpErrors.Unauthorized()
	}
	session := sessions[0]

	if session.ValidUntil.Compare(now) < 0 {
		return nil, httpErrors.Unauthorized()
	}

	return session, nil
}

func (s *SessionServiceImpl) IsKnownUserDevice(ctx context.Context, request IsKnownDeviceRequest) (*IsKnownDeviceResponse, error) {
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
