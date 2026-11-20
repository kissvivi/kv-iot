package service

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"kv-iot/pkg/result"
)

const (
	ClaimsContextKey = "auth.claims"
	forbiddenCode    = 4004
)

func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := TokenFromRequest(c)
		if token == "" {
			abortAuth(c, http.StatusUnauthorized, result.NO_TOKEN, "请求未携带token，无权限访问")
			return
		}

		claims, err := ParseToken(token)
		if err != nil {
			switch err {
			case ErrorTokenExpired:
				abortAuth(c, http.StatusUnauthorized, result.NO_TOKEN, "token授权已过期，请重新申请授权")
			case ErrorTokenNotValidYet:
				abortAuth(c, http.StatusUnauthorized, result.NO_TOKEN, "token尚未生效")
			default:
				abortAuth(c, http.StatusUnauthorized, result.NO_TOKEN, "token不可用")
			}
			return
		}

		c.Set(ClaimsContextKey, claims)
		c.Next()
	}
}

func PermissionRequired(code string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := ClaimsFromContext(c)
		if !ok {
			abortAuth(c, http.StatusUnauthorized, result.NO_TOKEN, "认证信息缺失，请重新登录")
			return
		}
		if strings.TrimSpace(code) == "" || HasPermission(claims, code) {
			c.Next()
			return
		}
		abortAuth(c, http.StatusForbidden, forbiddenCode, "权限不足，无法执行该操作")
	}
}

func ClaimsFromContext(c *gin.Context) (*Claims, bool) {
	value, ok := c.Get(ClaimsContextKey)
	if !ok {
		return nil, false
	}
	claims, ok := value.(*Claims)
	return claims, ok && claims != nil
}

func HasPermission(claims *Claims, required string) bool {
	if claims == nil {
		return false
	}
	if claims.Admin() {
		return true
	}
	required = strings.TrimSpace(required)
	for _, permission := range claims.Permissions {
		switch {
		case permission == "*":
			return true
		case permission == required:
			return true
		case strings.HasSuffix(permission, ":*"):
			prefix := strings.TrimSuffix(permission, "*")
			if strings.HasPrefix(required, prefix) {
				return true
			}
		}
	}
	return false
}

func abortAuth(c *gin.Context, status, code int, message string) {
	c.AbortWithStatusJSON(status, result.BaseResult{
		Code:      code,
		Msg:       message,
		Data:      nil,
		RequestID: c.GetHeader("X-Request-ID"),
	})
}
