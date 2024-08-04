package services

import (
	"context"
	"github.com/google/uuid"
	"holvit/httpErrors"
	"holvit/ioc"
	"holvit/middlewares"
	"holvit/repositories"
	"holvit/utils"
	"time"
)

type CreateSessionRequest struct {
	UserId   uuid.UUID
	RealmId  uuid.UUID
	DeviceId uuid.UUID
}

type SessionService interface {
	CreateSession(ctx context.Context, request CreateSessionRequest) (string, error)
	ValidateSession(ctx context.Context, token string) (*repositories.Session, error)
}

func NewSessionService() SessionService {
	return &SessionServiceImpl{}
}

type SessionServiceImpl struct{}

func (s *SessionServiceImpl) CreateSession(ctx context.Context, request CreateSessionRequest) (string, error) {
	scope := middlewares.GetScope(ctx)

	clockService := ioc.Get[utils.ClockService](scope)
	now := clockService.Now()

	token, err := utils.GenerateRandomStringBase64(32)
	if err != nil {
		return "", err
	}

	hashedToken := utils.CheapHash(token)

	sessionRepository := ioc.Get[repositories.SessionRepository](scope)
	_, err = sessionRepository.CreateSession(ctx, &repositories.Session{
		UserId:       request.UserId,
		UserDeviceId: request.DeviceId,
		RealmId:      request.RealmId,
		ValidUntil:   now.Add(time.Hour * 24 * 30), //TODO: read from realm config
		HashedToken:  hashedToken,
	})
	if err != nil {
		return "", err
	}

	return token, nil
}

func (s *SessionServiceImpl) ValidateSession(ctx context.Context, token string) (*repositories.Session, error) {
	scope := middlewares.GetScope(ctx)

	clockService := ioc.Get[utils.ClockService](scope)
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
		return nil, httpErrors.Unauthorized().WithMessage("session not found")
	}
	session := sessions[0]

	if session.ValidUntil.Compare(now) < 0 {
		return nil, httpErrors.Unauthorized().WithMessage("session not valid")
	}

	return session, nil
}
