package services

import (
	"context"
	"holvit/httpErrors"
	"holvit/ioc"
	"holvit/middlewares"
	"holvit/repositories"
	"holvit/utils"
)

type SessionService interface {
	ValidateSession(ctx context.Context, token string) (*repositories.Session, error)
}

func NewSessionService() SessionService {
	return &SessionServiceImpl{}
}

type SessionServiceImpl struct{}

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
		return nil, httpErrors.Unauthorized()
	}
	session := sessions[0]

	if session.ValidUntil.Compare(now) < 0 {
		return nil, httpErrors.Unauthorized()
	}

	return session, nil
}
