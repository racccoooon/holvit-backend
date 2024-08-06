package services

import (
	"context"
	"github.com/google/uuid"
	"holvit/h"
	"holvit/httpErrors"
	"holvit/ioc"
	"holvit/middlewares"
	"holvit/repos"
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
	ValidateSession(ctx context.Context, token string) (*repos.Session, error)
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

	sessionRepository := ioc.Get[repos.SessionRepository](scope)
	_ = sessionRepository.CreateSession(ctx, &repos.Session{
		UserId:       request.UserId,
		UserDeviceId: request.DeviceId,
		RealmId:      request.RealmId,
		ValidUntil:   now.Add(time.Hour * 24 * 30), //TODO: read from realm config
		HashedToken:  hashedToken,
	}).Unwrap()

	return token, nil
}

func (s *SessionServiceImpl) ValidateSession(ctx context.Context, token string) (*repos.Session, error) {
	scope := middlewares.GetScope(ctx)

	clockService := ioc.Get[utils.ClockService](scope)
	now := clockService.Now()

	hashedToken := utils.CheapHash(token)

	sessionRepository := ioc.Get[repos.SessionRepository](scope)
	session := sessionRepository.FindSessions(ctx, repos.SessionFilter{
		HashedToken: h.Some(hashedToken),
	}).Unwrap().First()

	if session.ValidUntil.Compare(now) < 0 {
		return nil, httpErrors.Unauthorized().WithMessage("session not valid")
	}

	return &session, nil
}
