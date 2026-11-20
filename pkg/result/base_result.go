package result

import "github.com/gin-gonic/gin"

const (
	OK       = 0
	ERR      = 5000
	NO_TOKEN = 4003
)

type BaseResult struct {
	Msg       string `json:"msg"`
	Code      int    `json:"code"`
	Data      any    `json:"data"`
	RequestID string `json:"requestId,omitempty"`
}

func (b BaseResult) ErrResult(c *gin.Context, res any, msg string) {
	b.Msg = msg
	b.Code = ERR
	b.Data = res
	b.RequestID = c.GetHeader("X-Request-ID")
	c.JSON(200, b)
}

func (b BaseResult) SuccessResult(c *gin.Context, res any, msg string) {
	b.Msg = msg
	b.Code = OK
	b.Data = res
	b.RequestID = c.GetHeader("X-Request-ID")
	c.JSON(200, b)
}

func (b BaseResult) NoTokenResult(c *gin.Context, res any, msg string) {
	b.Msg = msg
	b.Code = NO_TOKEN
	b.Data = res
	b.RequestID = c.GetHeader("X-Request-ID")
	c.JSON(200, b)
}
