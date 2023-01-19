package service

import (
	"kv-iot/auth/data"
	"kv-iot/auth/data/repo"
	"log"
)

var _ roleService = (*RoleServiceImpl)(nil)

type roleService interface {
	AddRole(role data.Role) (err error)
	DelRole(role data.Role) (err error)
}

type RoleServiceImpl struct {
	roleRepo repo.RoleRepo
}

func (r RoleServiceImpl) AddRole(role data.Role) (err error) {
	if err = r.roleRepo.Add(role); err != nil {
		log.Println(err)
		return
	}
	return
}

func (r RoleServiceImpl) DelRole(role data.Role) (err error) {
	if err = r.roleRepo.Delete(role); err != nil {
		log.Println(err)
		return
	}
	return
}

func NewRoleServiceImpl(roleRepo repo.RoleRepo) *RoleServiceImpl {
	return &RoleServiceImpl{roleRepo: roleRepo}
}
