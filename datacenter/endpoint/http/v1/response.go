package v1

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"kv-iot/datacenter/runtime"
	"kv-iot/datacenter/state"
)

const (
	codeOK           = 0
	codeBadRequest   = 4000
	codeUnauthorized = 4001
	codeNotFound     = 4004
	codeConflict     = 4009
	codeGatewayError = 5020
	codeInternal     = 5000
)

type Response struct {
	Code      int    `json:"code"`
	Data      any    `json:"data"`
	Msg       string `json:"msg"`
	RequestID string `json:"requestId,omitempty"`
}

func writeSuccess(c *gin.Context, status int, data any, message string) {
	c.JSON(status, Response{
		Code:      codeOK,
		Data:      data,
		Msg:       message,
		RequestID: requestID(c),
	})
}

func writeError(c *gin.Context, status, code int, message string) {
	c.JSON(status, Response{
		Code:      code,
		Data:      nil,
		Msg:       message,
		RequestID: requestID(c),
	})
}

func writeRuntimeError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	code := codeInternal
	switch {
	case errors.Is(err, context.Canceled):
		status = 499
		code = codeBadRequest
	case errors.Is(err, context.DeadlineExceeded):
		status = http.StatusGatewayTimeout
		code = codeGatewayError
	case errors.Is(err, state.ErrStateNotFound),
		errors.Is(err, runtime.ErrChannelNotFound),
		errors.Is(err, runtime.ErrCommandNotFound):
		status = http.StatusNotFound
		code = codeNotFound
	case errors.Is(err, runtime.ErrChannelAlreadyRunning),
		errors.Is(err, runtime.ErrEndpointAlreadyRunning),
		errors.Is(err, runtime.ErrCommandInProgress):
		status = http.StatusConflict
		code = codeConflict
	case errors.Is(err, runtime.ErrInvalidChannelSpec),
		errors.Is(err, runtime.ErrCommandInvalid):
		status = http.StatusBadRequest
		code = codeBadRequest
	}
	writeError(c, status, code, err.Error())
}

func requestID(c *gin.Context) string {
	if value := c.GetString("runtime_request_id"); value != "" {
		return value
	}
	return runtime.GenerateRequestID()
}
