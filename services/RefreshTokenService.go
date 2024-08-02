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
	ValidateAndRefresh(ctx context.Context, token string, clientId uuid.UUID) (string, *repositories.RefreshToken, error)
	CreateRefreshToken(ctx context.Context, request CreateRefreshTokenRequest) (string, *repositories.RefreshToken, error)
}

func NewRefreshTokenService() RefreshTokenService {
	return &RefreshTokenServiceImpl{}
}

type RefreshTokenServiceImpl struct{}

func (r *RefreshTokenServiceImpl) ValidateAndRefresh(ctx context.Context, token string, clientId uuid.UUID) (string, *repositories.RefreshToken, error) {
	scope := middlewares.GetScope(ctx)

	hashedToken := utils.CheapHash(token)

	clockService := ioc.Get[ClockService](scope)
	now := clockService.Now()

	refreshTokenRepository := ioc.Get[repositories.RefreshTokenRepository](scope)
	tokens, count, err := refreshTokenRepository.FindRefreshTokens(ctx, repositories.RefreshTokenFilter{
		HashedToken: &hashedToken,
	})
	if err != nil {
		return "", nil, err
	}
	if count == 0 {
		return "", nil, httpErrors.Unauthorized()
	}

	refreshToken := tokens[0]
	if refreshToken.ValidUntil.Compare(now) < 0 {
		return "", nil, httpErrors.Unauthorized()
	}

	err = refreshTokenRepository.DeleteRefreshToken(ctx, refreshToken.Id)
	if err != nil {
		return "", nil, err
	}

	return r.CreateRefreshToken(ctx, CreateRefreshTokenRequest{
		ClientId: clientId,
		UserId:   refreshToken.UserId,
		RealmId:  refreshToken.RealmId,
		Issuer:   refreshToken.Issuer,
		Subject:  refreshToken.Subject,
		Audience: refreshToken.Audience,
		Scopes:   refreshToken.Scopes,
	})
}

func (r *RefreshTokenServiceImpl) CreateRefreshToken(ctx context.Context, request CreateRefreshTokenRequest) (string, *repositories.RefreshToken, error) {
	scope := middlewares.GetScope(ctx)

	clockService := ioc.Get[ClockService](scope)
	now := clockService.Now()

	token, err := utils.GenerateRandomString(32)
	if err != nil {
		return "", nil, err
	}

	hashedToken := utils.CheapHash(token)

	refreshTokenRepository := ioc.Get[repositories.RefreshTokenRepository](scope)
	refreshToken := repositories.RefreshToken{
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
	tokenId, err := refreshTokenRepository.CreateRefreshToken(ctx, &refreshToken)
	if err != nil {
		return "", nil, err
	}

	refreshToken.Id = tokenId

	return token, &refreshToken, nil
}
