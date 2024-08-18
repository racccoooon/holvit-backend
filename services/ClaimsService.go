package services

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/sourcegraph/conc/iter"
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
	return &claimsServiceImpl{}
}

type claimsServiceImpl struct{}

func (c *claimsServiceImpl) GetClaims(ctx context.Context, request GetClaimsRequest) []ClaimResponse {
	scope := middlewares.GetScope(ctx)

	claimMapperRepository := ioc.Get[repos.ClaimMapperRepository](scope)
	mappers := claimMapperRepository.FindClaimMappers(ctx, repos.ClaimMapperFilter{
		ScopeIds: h.Some(request.ScopeIds),
	})

	claims := make([]ClaimResponse, 0, len(mappers.Values()))

	userInfoMappers := make([]interface{}, 0)
	rolesMappers := make([]interface{}, 0)

	for _, mapper := range mappers.Values() {
		switch mapper.Type {
		case constants.ClaimMapperUserInfo:
			userInfoMappers = append(userInfoMappers, mapper.Details)
		case constants.ClaimMapperRoles:
			rolesMappers = append(rolesMappers, mapper.Details)
		}
	}

	userRepository := ioc.Get[repos.UserRepository](scope)
	user := userRepository.FindUserById(ctx, request.UserId).Unwrap()

	if len(rolesMappers) > 0 {
		userRoleRepository := ioc.Get[repos.UserRoleRepository](scope)
		roleRepository := ioc.Get[repos.RoleRepository](scope)

		userRoles := userRoleRepository.FindUserRoles(ctx, repos.UserRoleFilter{
			UserId: request.UserId,
		})

		roleIds := iter.Map(userRoles, func(userRole *repos.UserRole) uuid.UUID {
			return userRole.RoleId
		})

		roles := roleRepository.FindRoles(ctx, repos.RoleFilter{
			RealmId: user.RealmId,
			RoleIds: h.Some(roleIds),
		})

		roleNames := iter.Map(roles.Values(), func(role *repos.Role) string {
			return role.Name
		})

		claims = append(claims, ClaimResponse{
			Name:  "role",
			Claim: roleNames,
		})
	}

	if len(userInfoMappers) > 0 {

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
