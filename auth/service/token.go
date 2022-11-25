package service

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"kv-iot/auth/data"
	"kv-iot/config"
	"strconv"
	"time"
)

// JWTAuth 权限校验接口
type JWTAuth interface {
	GetToken(username, password string) (token string, errcode string)
	CheckToken() gin.HandlerFunc
	Verify() gin.HandlerFunc
}

// Claims 增加授权功能点
// 增加是否为系统超管权限
type Claims struct {
	Username string `json:"username"`
	// Password string `json:"password"`
	Roles   []string `json:"roles"` // 授权功能点
	IsAdmin bool     // 是否为管理员，为true时表示管理，不做权限校验，默认拥有所有操作权限
	jwt.StandardClaims
}

var jwtKey = []byte(config.Application{}.AuthServer.JwtKey)

// GenerateToken 生成授权后的token
func GenerateToken(user data.User) (string, error) {
	nowTime := time.Now()
	expireTime := nowTime.Add(time.Duration(config.Application{}.AuthServer.TokenTimeout) * time.Minute)
	authCodes := []string{""}
	claims := Claims{
		user.UserName,
		authCodes,
		user.IsAdmin(),
		jwt.StandardClaims{
			ExpiresAt: expireTime.Unix(),
			Issuer:    "",
			Id:        strconv.Itoa(int(user.ID)),
		},
	}

	tokenClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err := tokenClaims.SignedString(jwtKey)

	return token, err
}

func ParseToken(token string) (*Claims, error) {
	tokenClaims, err := jwt.ParseWithClaims(token, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})

	if tokenClaims != nil {
		if claims, ok := tokenClaims.Claims.(*Claims); ok && tokenClaims.Valid {
			return claims, nil
		}
	}

	return nil, err
}
