package services

type RoleService interface {
}

func NewRoleService() RoleService {
	return &RoleServiceImpl{}
}

type RoleServiceImpl struct{}
