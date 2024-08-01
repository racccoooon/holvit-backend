package services

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"holvit/constants"
	"holvit/ioc"
	"holvit/logging"
	"holvit/middlewares"
	"holvit/repositories"
)

type ClaimResponse struct {
	Name  string
	Claim interface{}
}

type GetClaimsRequest struct {
	UserId   uuid.UUID
	ScopeIds []uuid.UUID
}

type ClaimsService interface {
	GetClaims(ctx context.Context, request GetClaimsRequest) ([]*ClaimResponse, error)
}

func NewClaimsService() ClaimsService {
	return &ClaimsServiceImpl{}
}

type ClaimsServiceImpl struct{}

func (c *ClaimsServiceImpl) GetClaims(ctx context.Context, request GetClaimsRequest) ([]*ClaimResponse, error) {
	scope := middlewares.GetScope(ctx)

	claimMapperRepository := ioc.Get[repositories.ClaimMapperRepository](scope)
	mappers, _, err := claimMapperRepository.FindClaimMappers(ctx, repositories.ClaimMapperFilter{
		ScopeIds: request.ScopeIds,
	})
	if err != nil {
		return nil, err
	}

	claims := make([]*ClaimResponse, 0, len(mappers))

	userInfoMappers := make([]interface{}, 0)
	for _, mapper := range mappers {
		switch mapper.Type {
		case constants.ClaimMapperUserInfo:
			userInfoMappers = append(userInfoMappers, mapper.Details)
			break
		}
	}

	userRepository := ioc.Get[repositories.UserRepository](scope)
	if len(userInfoMappers) > 0 {
		user, err := userRepository.FindUserById(ctx, request.UserId)
		if err != nil {
			return nil, err
		}

		for _, m := range userInfoMappers {
			mapper := m.(repositories.UserInfoClaimMapperDetails)

			switch mapper.Property {
			case constants.UserInfoPropertyId:
				claims = append(claims, &ClaimResponse{
					Name:  mapper.ClaimName,
					Claim: user.Id.String(),
				})
				break
			case constants.UserInfoPropertyUsername:
				if user.Username != nil {
					claims = append(claims, &ClaimResponse{
						Name:  mapper.ClaimName,
						Claim: *user.Username,
					})
				}
				break
			case constants.UserInfoPropertyEmail:
				if user.Email != nil {
					claims = append(claims, &ClaimResponse{
						Name:  mapper.ClaimName,
						Claim: *user.Email,
					})
				}
				break
			case constants.UserInfoPropertyEmailVerified:
				if user.Email != nil {
					claims = append(claims, &ClaimResponse{
						Name:  mapper.ClaimName,
						Claim: fmt.Sprintf("%t", user.EmailVerified),
					})
				}
				break
			default:
				logging.Logger.Fatalf("Unknown user property %s", mapper.Property)
			}
		}
	}

	return claims, nil
}
