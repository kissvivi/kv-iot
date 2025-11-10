package service

import (
	"errors"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"kv-iot/auth/data"
	"kv-iot/config"
	"log"
	"strconv"
	"time"
)

var (
	ErrorTokenMalformed   = errors.New("token不可用")
	ErrorTokenExpired     = errors.New("token过期")
	ErrorTokenNotValidYet = errors.New("token无效")
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
	Username string   `json:"username"`
	Roles    []string `json:"roles"`    // 授权功能点
	IsAdmin  bool     `json:"is_admin"` // 是否为管理员，为true时表示管理，不做权限校验，默认拥有所有操作权限
	jwt.StandardClaims
}

// jwtKey JWT签名密钥
var jwtKey []byte

// 初始化JWT密钥
func init() {
	// 从配置中加载JWT密钥
	cfg, err := config.InitConfig()
	if err != nil {
		log.Printf("初始化JWT配置失败，使用默认密钥: %v", err)
		jwtKey = []byte("default_jwt_key_change_in_production")
	} else {
		jwtKey = []byte(cfg.Application.AuthServer.JwtKey)
	}
}

// GenerateToken 生成授权后的token
func GenerateToken(user data.User) (string, error) {
	nowTime := time.Now()
	// 从配置获取token过期时间
	cfg, err := config.InitConfig()
	if err != nil {
		log.Printf("获取JWT配置失败，使用默认过期时间: %v", err)
	}

	var expireTime time.Time
	tokenTimeout := 30 // 默认30分钟
	if err == nil {
		tokenTimeout = int(cfg.Application.AuthServer.TokenTimeout)
	}
	expireTime = nowTime.Add(time.Duration(tokenTimeout) * time.Minute)

	// 获取用户角色信息
	authCodes := []string{}
	// TODO: 从数据库获取用户的实际角色和权限

	claims := Claims{
		Username: user.UserName,
		Roles:    authCodes,
		IsAdmin:  user.IsAdmin(),
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expireTime.Unix(),
			NotBefore: time.Now().Add(-time.Second * 10).Unix(),
			IssuedAt:  nowTime.Unix(),
			Issuer:    "kv-iot",
			Subject:   strconv.Itoa(int(user.ID)),
			Id:        strconv.FormatInt(time.Now().UnixNano(), 10), // 添加唯一ID
		},
	}

	tokenClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err := tokenClaims.SignedString(jwtKey)

	if err == nil {
		log.Printf("为用户 %s 生成JWT令牌成功，过期时间: %v", user.UserName, expireTime)
	} else {
		log.Printf("生成JWT令牌失败: %v", err)
	}

	return token, err
}

func ParseToken(token string) (*Claims, error) {
	tokenClaims, err := jwt.ParseWithClaims(token, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// 验证签名算法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			log.Printf("无效的签名方法: %v", token.Header["alg"])
			return nil, ErrorTokenMalformed
		}
		return jwtKey, nil
	})

	// 处理验证错误
	if ve, ok := err.(*jwt.ValidationError); ok {
		if ve.Errors&jwt.ValidationErrorMalformed != 0 {
			log.Printf("JWT令牌格式错误")
			return nil, ErrorTokenMalformed
		} else if ve.Errors&jwt.ValidationErrorExpired != 0 {
			log.Printf("JWT令牌已过期")
			return nil, ErrorTokenExpired
		} else if ve.Errors&jwt.ValidationErrorNotValidYet != 0 {
			log.Printf("JWT令牌尚未生效")
			return nil, ErrorTokenNotValidYet
		} else {
			log.Printf("JWT令牌验证失败: %v", ve.Errors)
			return nil, ErrorTokenMalformed
		}
	} else if err != nil {
		log.Printf("JWT令牌解析失败: %v", err)
		return nil, ErrorTokenMalformed
	}

	if tokenClaims != nil {
		if claims, ok := tokenClaims.Claims.(*Claims); ok && tokenClaims.Valid {
			log.Printf("JWT令牌验证成功，用户: %s", claims.Username)
			return claims, nil
		}
	}

	return nil, ErrorTokenMalformed
}
