package services

import (
	"context"
	"github.com/google/uuid"
	"holvit/h"
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
	CreateSession(ctx context.Context, request CreateSessionRequest) string
	LookupSession(ctx context.Context, token string) h.Optional[repos.Session]
}

func NewSessionService() SessionService {
	return &SessionServiceImpl{}
}

type SessionServiceImpl struct{}

func (s *SessionServiceImpl) CreateSession(ctx context.Context, request CreateSessionRequest) string {
	scope := middlewares.GetScope(ctx)

	clockService := ioc.Get[utils.ClockService](scope)
	now := clockService.Now()

	token := utils.GenerateRandomStringBase64(32) //TODO: constant
	hashedToken := utils.CheapHash(token)

	sessionRepository := ioc.Get[repos.SessionRepository](scope)
	_ = sessionRepository.CreateSession(ctx, repos.Session{
		UserId:       request.UserId,
		UserDeviceId: request.DeviceId,
		RealmId:      request.RealmId,
		ValidUntil:   now.Add(time.Hour * 24 * 30), //TODO: read from realm config
		HashedToken:  hashedToken,
	})

	return token
}

func (s *SessionServiceImpl) LookupSession(ctx context.Context, token string) h.Optional[repos.Session] {
	scope := middlewares.GetScope(ctx)

	clockService := ioc.Get[utils.ClockService](scope)
	now := clockService.Now()

	hashedToken := utils.CheapHash(token)

	sessionRepository := ioc.Get[repos.SessionRepository](scope)
	session := sessionRepository.FindSessions(ctx, repos.SessionFilter{
		HashedToken: h.Some(hashedToken),
	}).FirstOrNone()

	if session, ok := session.Get(); ok && session.ValidUntil.Compare(now) < 0 {
		return h.None[repos.Session]()
	}

	return session
}
