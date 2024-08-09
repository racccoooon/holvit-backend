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

type CreateRefreshTokenRequest struct {
	ClientId uuid.UUID
	UserId   uuid.UUID
	RealmId  uuid.UUID

	Issuer   string
	Subject  string
	Audience string
	Scopes   []string
}

type RefreshTokenService interface {
	ValidateAndRefresh(ctx context.Context, token string, clientId uuid.UUID) h.Result[h.T2[string, repos.RefreshToken]]
	CreateRefreshToken(ctx context.Context, request CreateRefreshTokenRequest) (string, repos.RefreshToken)
}

func NewRefreshTokenService() RefreshTokenService {
	return &RefreshTokenServiceImpl{}
}

type RefreshTokenServiceImpl struct{}

func (r *RefreshTokenServiceImpl) ValidateAndRefresh(ctx context.Context, token string, clientId uuid.UUID) h.Result[h.T2[string, repos.RefreshToken]] {
	scope := middlewares.GetScope(ctx)

	hashedToken := utils.CheapHash(token)

	clockService := ioc.Get[utils.ClockService](scope)
	now := clockService.Now()

	refreshTokenRepository := ioc.Get[repos.RefreshTokenRepository](scope)
	refreshToken := refreshTokenRepository.FindRefreshTokens(ctx, repos.RefreshTokenFilter{
		HashedToken: h.Some(hashedToken),
	}).First()

	if refreshToken.ValidUntil.Compare(now) < 0 {
		return h.Err[h.T2[string, repos.RefreshToken]](httpErrors.Unauthorized().WithMessage("token not valid"))
	}

	refreshTokenRepository.DeleteRefreshToken(ctx, refreshToken.Id).Unwrap()

	return h.Ok(h.NewT2(r.CreateRefreshToken(ctx, CreateRefreshTokenRequest{
		ClientId: clientId,
		UserId:   refreshToken.UserId,
		RealmId:  refreshToken.RealmId,
		Issuer:   refreshToken.Issuer,
		Subject:  refreshToken.Subject,
		Audience: refreshToken.Audience,
		Scopes:   refreshToken.Scopes,
	})))
}

func (r *RefreshTokenServiceImpl) CreateRefreshToken(ctx context.Context, request CreateRefreshTokenRequest) (string, repos.RefreshToken) {
	scope := middlewares.GetScope(ctx)

	clockService := ioc.Get[utils.ClockService](scope)
	now := clockService.Now()

	token := utils.GenerateRandomStringBase64(32) // TODO: constant
	hashedToken := utils.CheapHash(token)

	refreshTokenRepository := ioc.Get[repos.RefreshTokenRepository](scope)
	refreshToken := repos.RefreshToken{
		UserId:      request.UserId,
		ClientId:    request.ClientId,
		RealmId:     request.RealmId,
		HashedToken: hashedToken,
		ValidUntil:  now.Add(time.Hour), //TODO: make configurable
		Issuer:      request.Issuer,
		Subject:     request.Subject,
		Audience:    request.Audience,
		Scopes:      request.Scopes,
	}
	tokenId := refreshTokenRepository.CreateRefreshToken(ctx, refreshToken)

	refreshToken.Id = tokenId

	return token, refreshToken
}
