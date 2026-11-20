package api

import (
	"github.com/gin-gonic/gin"
	"kv-iot/pkg/result"
)

func success(c *gin.Context, data any, message string) {
	result.BaseResult{}.SuccessResult(c, data, message)
}

func fail(c *gin.Context, status int, message string) {
	c.JSON(status, result.BaseResult{
		Code:      result.ERR,
		Msg:       message,
		Data:      nil,
		RequestID: c.GetHeader("X-Request-ID"),
	})
}
