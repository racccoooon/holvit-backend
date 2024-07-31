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
}

type RefreshTokenService interface {
	ValidateAndRefresh(ctx context.Context, token string, clientId uuid.UUID) (string, error)
	CreateRefreshToken(ctx context.Context, request CreateRefreshTokenRequest) (string, error)
}

func NewRefreshTokenService() RefreshTokenService {
	return &RefreshTokenServiceImpl{}
}

type RefreshTokenServiceImpl struct{}

func (r *RefreshTokenServiceImpl) ValidateAndRefresh(ctx context.Context, token string, clientId uuid.UUID) (string, error) {
	scope := middlewares.GetScope(ctx)

	hashedToken := utils.CheapHash(token)

	refreshTokenRepository := ioc.Get[repositories.RefreshTokenRepository](scope)
	tokens, count, err := refreshTokenRepository.FindRefreshTokens(ctx, repositories.RefreshTokenFilter{
		HashedToken: &hashedToken,
	})
	if err != nil {
		return "", err
	}
	if count == 0 {
		return "", httpErrors.Unauthorized()
	}

	refreshToken := tokens[0]
	if refreshToken.ValidUntil.Compare(time.Now()) < 0 {
		return "", httpErrors.Unauthorized()
	}

	refreshTokenRepository.DeleteRefreshToken(ctx, refreshToken.Id)

	return r.CreateRefreshToken(ctx, CreateRefreshTokenRequest{
		ClientId: clientId,
		UserId:   refreshToken.UserId,
		RealmId:  refreshToken.RealmId,
	})
}

func (r *RefreshTokenServiceImpl) CreateRefreshToken(ctx context.Context, request CreateRefreshTokenRequest) (refreshTokenId string, err error) {
	scope := middlewares.GetScope(ctx)

	token, err := utils.GenerateRandomString(32)
	if err != nil {
		return "", err
	}

	hashedToken := utils.CheapHash(token)

	refreshTokenRepository := ioc.Get[repositories.RefreshTokenRepository](scope)
	_, err = refreshTokenRepository.CreateRefreshToken(ctx, &repositories.RefreshToken{
		UserId:      request.UserId,
		ClientId:    request.ClientId,
		RealmId:     request.RealmId,
		HashedToken: hashedToken,
		ValidUntil:  time.Now().Add(time.Hour), //TODO: make configurable
	})
	if err != nil {
		return "", err
	}

	return token, nil
}
