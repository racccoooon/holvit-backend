package services

import (
	"context"
	"github.com/google/uuid"
	"holvit/h"
	"holvit/ioc"
	"holvit/middlewares"
	"holvit/repos"
)

type CreateRoleRequest struct {
	RealmId     uuid.UUID
	ClientId    h.Opt[uuid.UUID]
	Name        string
	DisplayName h.Opt[string]
	Description h.Opt[string]
}

type RoleService interface {
	CreateRole(ctx context.Context, request CreateRoleRequest)
}

func NewRoleService() RoleService {
	return &RoleServiceImpl{}
}

type RoleServiceImpl struct{}

func (s *RoleServiceImpl) CreateRole(ctx context.Context, request CreateRoleRequest) {
	scope := middlewares.GetScope(ctx)

	rolesRepository := ioc.Get[repos.RolesRepository](scope)
	rolesRepository.CreateRole(ctx, repos.Role{
		RealmId:     request.RealmId,
		ClientId:    request.ClientId,
		DisplayName: request.DisplayName.OrDefault(request.Name),
		Name:        request.Name,
		Description: request.Description.OrDefault(""),
	}).Unwrap() // TODO: handle duplicate error
}
