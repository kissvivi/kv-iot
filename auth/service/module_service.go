package service

import (
	"errors"
	"kv-iot/auth/data"
	"kv-iot/auth/data/repo"
	"strings"

	"gorm.io/gorm"
)

var (
	ErrModuleExists       = errors.New("模块编码已存在")
	ErrModuleCodeRequired = errors.New("模块编码不能为空")
)

type ModuleService interface {
	ListModules() ([]data.Module, error)
	GetModule(id int64) (data.Module, error)
	CreateModule(module data.Module) (data.Module, error)
	UpdateModule(id int64, fields map[string]interface{}) (data.Module, error)
	DeleteModule(id int64) error
}

type ModuleServiceImpl struct {
	moduleRepo repo.ModuleRepo
}

func NewModuleServiceImpl(moduleRepo repo.ModuleRepo) *ModuleServiceImpl {
	return &ModuleServiceImpl{moduleRepo: moduleRepo}
}

func (s ModuleServiceImpl) ListModules() ([]data.Module, error) {
	return s.moduleRepo.List()
}

func (s ModuleServiceImpl) GetModule(id int64) (data.Module, error) {
	module, err := s.moduleRepo.Get(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return data.Module{}, ErrModuleNotFound
	}
	return module, err
}

func (s ModuleServiceImpl) CreateModule(module data.Module) (data.Module, error) {
	module.Name = strings.TrimSpace(module.Name)
	module.Code = strings.TrimSpace(module.Code)
	if module.Code == "" {
		return data.Module{}, ErrModuleCodeRequired
	}
	if _, err := s.moduleRepo.FindByCode(module.Code); err == nil {
		return data.Module{}, ErrModuleExists
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return data.Module{}, err
	}
	if err := s.moduleRepo.Create(&module); err != nil {
		return data.Module{}, err
	}
	return module, nil
}

func (s ModuleServiceImpl) UpdateModule(id int64, fields map[string]interface{}) (data.Module, error) {
	if codeValue, ok := fields["code"]; ok {
		code, ok := codeValue.(string)
		if !ok {
			return data.Module{}, ErrModuleCodeRequired
		}
		code = strings.TrimSpace(code)
		if code == "" {
			return data.Module{}, ErrModuleCodeRequired
		}
		if existing, err := s.moduleRepo.FindByCode(code); err == nil && int64(existing.ID) != id {
			return data.Module{}, ErrModuleExists
		} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return data.Module{}, err
		}
		fields["code"] = code
	}
	if len(fields) > 0 {
		if err := s.moduleRepo.Update(id, fields); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return data.Module{}, ErrModuleNotFound
			}
			return data.Module{}, err
		}
	}
	return s.GetModule(id)
}

func (s ModuleServiceImpl) DeleteModule(id int64) error {
	err := s.moduleRepo.DeleteWithRelations(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrModuleNotFound
	}
	return err
}
