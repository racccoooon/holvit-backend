package services

import (
	"context"
	"github.com/google/uuid"
	"holvit/h"
	"holvit/ioc"
	"holvit/middlewares"
	"holvit/repos"
	"slices"
)

type CreateRoleRequest struct {
	RealmId     uuid.UUID
	ClientId    h.Opt[uuid.UUID]
	Name        string
	DisplayName h.Opt[string]
	Description h.Opt[string]
}

type DeleteRoleRequest struct {
	RoleIds []uuid.UUID
	RealmId uuid.UUID
}

type SetImplicationRequest struct {
	RealmId uuid.UUID
	RoleId  uuid.UUID
	RoleIds []uuid.UUID
}

type WrongRealmRoleError struct{}

func (e WrongRealmRoleError) Error() string {
	return "Wrong realm"
}

type CircularRoleImplicationError struct{}

func (e CircularRoleImplicationError) Error() string {
	return "Circular role implication error"
}

type GetRolesForUserRequest struct {
	UserId  uuid.UUID
	RealmId uuid.UUID
}

type RoleService interface {
	CreateRole(ctx context.Context, request CreateRoleRequest)
	DeleteRoles(ctx context.Context, request DeleteRoleRequest)
	SetImplications(ctx context.Context, request SetImplicationRequest) h.Result[h.Unit]
}

func NewRoleService() RoleService {
	return &roleServiceImpl{}
}

type roleServiceImpl struct{}

func (s *roleServiceImpl) CreateRole(ctx context.Context, request CreateRoleRequest) {
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

func (s *roleServiceImpl) DeleteRoles(ctx context.Context, request DeleteRoleRequest) {
	scope := middlewares.GetScope(ctx)

	rolesRepository := ioc.Get[repos.RolesRepository](scope)
	rolesRepository.DeleteRoles(ctx, request.RealmId, request.RoleIds)
	s.recalculateCache(ctx, request.RealmId)
}

func (s *roleServiceImpl) recalculateCache(ctx context.Context, realmId uuid.UUID) {
	scope := middlewares.GetScope(ctx)

	rolesRepository := ioc.Get[repos.RolesRepository](scope)
	roleImplicationRepository := ioc.Get[repos.RoleImplicationRepository](scope)

	roles := rolesRepository.FindRoles(ctx, repos.RoleFilter{
		RealmId: h.Some(realmId),
	})
	implications := roleImplicationRepository.FindRoleImplications(ctx, repos.RoleImplicationFilter{
		RealmId: realmId,
	})

	implicationCache := make(map[uuid.UUID]h.T2[bool, []uuid.UUID], len(roles.Values()))

	for _, role := range roles.Values() {
		implicationCache[role.Id] = h.NewT2(false, make([]uuid.UUID, 0))
	}

	for _, implication := range implications {
		impliedRoles := implicationCache[implication.RoleId].Second
		impliedRoles = append(impliedRoles, implication.ImpliedRoleId)
		implicationCache[implication.RoleId] = h.NewT2(false, impliedRoles)
	}

	var findImplications func(roleId uuid.UUID) []uuid.UUID
	findImplications = func(roleId uuid.UUID) []uuid.UUID {
		if implicationCache[roleId].First {
			return implicationCache[roleId].Second
		}
		res := make([]uuid.UUID, 0)
		for _, edge := range implicationCache[roleId].Second {
			res = append(res, edge)
			res = append(res, findImplications(edge)...)
		}
		implicationCache[roleId] = h.NewT2(true, res)
		return res
	}

	for _, role := range roles.Values() {
		impliedRoles := findImplications(role.Id)
		rolesRepository.UpdateRole(ctx, role.Id, repos.RoleUpdate{
			ImpliesCache: h.Some(impliedRoles),
		})
	}
}

func (s *roleServiceImpl) SetImplications(ctx context.Context, request SetImplicationRequest) h.Result[h.Unit] {
	scope := middlewares.GetScope(ctx)

	rolesRepository := ioc.Get[repos.RolesRepository](scope)
	roleImplicationRepository := ioc.Get[repos.RoleImplicationRepository](scope)

	impliedRoles := rolesRepository.FindRoles(ctx, repos.RoleFilter{
		RoleIds: h.Some(request.RoleIds),
	})

	implications := make([]repos.RoleImplication, 0, len(request.RoleIds))
	for _, impliedRole := range impliedRoles.Values() {
		if slices.Contains(impliedRole.ImpliesCache, request.RoleId) {
			return h.UErr(CircularRoleImplicationError{})
		}
		if impliedRole.RealmId != request.RealmId {
			return h.UErr(WrongRealmRoleError{})
		}

		implications = append(implications, repos.RoleImplication{
			RoleId:        request.RoleId,
			ImpliedRoleId: impliedRole.Id,
		})
	}

	roleImplicationRepository.DeleteImplicationsForRole(ctx, request.RoleId)
	roleImplicationRepository.CreateImplications(ctx, implications)

	rolesRepository.UpdateRole(ctx, request.RoleId, repos.RoleUpdate{
		ImpliesCache: h.Some(request.RoleIds),
	})

	s.recalculateCache(ctx, request.RealmId)

	return h.UOk()
}
