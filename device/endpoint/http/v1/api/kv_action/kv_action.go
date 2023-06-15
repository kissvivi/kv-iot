package kv_action

import (
	"github.com/gin-gonic/gin"
	"kv-iot/device/data"
	"kv-iot/device/service"
	"kv-iot/pkg/result"
)

type ApiKvAction struct {
	baseService *service.BaseService
}

func NewApiKvAction(baseService *service.BaseService) *ApiKvAction {
	return &ApiKvAction{baseService: baseService}
}

func (ak *ApiKvAction) AddKvAction(c *gin.Context) {
	action := data.KvAction{}
	if err := c.BindJSON(&action); err != nil {
		result.BaseResult{}.ErrResult(c, nil, "参数错误")
		return
	}

	err := ak.baseService.KvActionService.AddKvAction(action)
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "添加失败")
	} else {
		result.BaseResult{}.SuccessResult(c, action, "添加成功")
	}
}

func (ak *ApiKvAction) DelKvAction(c *gin.Context) {
	action := data.KvAction{}
	if err := c.BindJSON(&action); err != nil {
		result.BaseResult{}.ErrResult(c, nil, "参数错误")
		return
	}

	err := ak.baseService.KvActionService.DelKvAction(action)
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "删除失败")
	} else {
		result.BaseResult{}.SuccessResult(c, nil, "删除成功")
	}
}

func (ak *ApiKvAction) GetKvAction(c *gin.Context) {
	action := data.KvAction{}
	if err := c.BindJSON(&action); err != nil {
		result.BaseResult{}.ErrResult(c, nil, "参数错误")
		return
	}

	err, actionList := ak.baseService.KvActionService.GetKvAction(action)
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "查询失败")
	} else {
		result.BaseResult{}.SuccessResult(c, actionList, "查询成功")
	}
}

func (ak *ApiKvAction) GetAllKvAction(c *gin.Context) {
	err, actionList := ak.baseService.KvActionService.GetAllKvAction()
	if err != nil {
		result.BaseResult{}.ErrResult(c, nil, "查询失败")
	} else {
		result.BaseResult{}.SuccessResult(c, actionList, "查询成功")
	}
}
