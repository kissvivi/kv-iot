package result

import "github.com/gin-gonic/gin"

const (
	OK       = 2000
	ERR      = 5000
	NO_TOKEN = 4003
)

type BaseResult struct {
	Msg    string `json:"msg"`
	Code   int    `json:"code"`
	Result any    `json:"result"`
}

func (b BaseResult) ErrResult(c *gin.Context, res any, msg string) {
	b.Msg = msg
	b.Code = ERR
	b.Result = res
	c.JSON(200, b)
}

func (b BaseResult) SuccessResult(c *gin.Context, res any, msg string) {
	b.Msg = msg
	b.Code = OK
	b.Result = res
	c.JSON(200, b)
}

func (b BaseResult) NoTokenResult(c *gin.Context, res any, msg string) {
	b.Msg = msg
	b.Code = NO_TOKEN
	b.Result = res
	c.JSON(200, b)
}
