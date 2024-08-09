package services

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"holvit/constants"
	"holvit/h"
	"holvit/ioc"
	"holvit/logging"
	"holvit/middlewares"
	"holvit/repos"
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
	GetClaims(ctx context.Context, request GetClaimsRequest) []ClaimResponse
}

func NewClaimsService() ClaimsService {
	return &ClaimsServiceImpl{}
}

type ClaimsServiceImpl struct{}

func (c *ClaimsServiceImpl) GetClaims(ctx context.Context, request GetClaimsRequest) []ClaimResponse {
	scope := middlewares.GetScope(ctx)

	claimMapperRepository := ioc.Get[repos.ClaimMapperRepository](scope)
	mappers := claimMapperRepository.FindClaimMappers(ctx, repos.ClaimMapperFilter{
		ScopeIds: h.Some(request.ScopeIds),
	})

	claims := make([]ClaimResponse, 0, len(mappers.Values()))

	userInfoMappers := make([]interface{}, 0)
	for _, mapper := range mappers.Values() {
		switch mapper.Type {
		case constants.ClaimMapperUserInfo:
			userInfoMappers = append(userInfoMappers, mapper.Details)
		}
	}

	userRepository := ioc.Get[repos.UserRepository](scope)
	if len(userInfoMappers) > 0 {
		user := userRepository.FindUserById(ctx, request.UserId).Unwrap()

		for _, m := range userInfoMappers {
			mapper := m.(repos.UserInfoClaimMapperDetails)

			switch mapper.Property {
			case constants.UserInfoPropertyId:
				claims = append(claims, ClaimResponse{
					Name:  mapper.ClaimName,
					Claim: user.Id.String(),
				})
			case constants.UserInfoPropertyUsername:
				claims = append(claims, ClaimResponse{
					Name:  mapper.ClaimName,
					Claim: user.Username,
				})
			case constants.UserInfoPropertyEmail:
				user.Email.IfSome(func(x string) {
					claims = append(claims, ClaimResponse{
						Name:  mapper.ClaimName,
						Claim: x,
					})
				})
			case constants.UserInfoPropertyEmailVerified:
				if user.Email.IsSome() {
					claims = append(claims, ClaimResponse{
						Name:  mapper.ClaimName,
						Claim: fmt.Sprintf("%t", user.EmailVerified),
					})
				}
			default:
				logging.Logger.Fatalf("Unknown user property %s", mapper.Property)
			}
		}
	}

	return claims
}
