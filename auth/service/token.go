package service

import (
	"errors"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"kv-iot/auth/data"
	"kv-iot/config"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
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
	Username    string   `json:"username"`
	Roles       []string `json:"roles,omitempty"`       // 用户角色标识
	Permissions []string `json:"permissions,omitempty"` // 用户权限标识
	IsAdmin     bool     `json:"isAdmin"`               // 是否为管理员，为true时不校验权限
	LegacyAdmin bool     `json:"is_admin,omitempty"`    // 兼容旧版Token
	jwt.StandardClaims
}

func (c Claims) Admin() bool {
	return c.IsAdmin || c.LegacyAdmin
}

var (
	jwtKeyOnce sync.Once
	jwtKey     []byte
)

func currentJWTKey() []byte {
	jwtKeyOnce.Do(func() {
		jwtKey = []byte(strings.TrimSpace(os.Getenv("KV_IOT_JWT_KEY")))
		if len(jwtKey) > 0 {
			return
		}

		cfg, err := config.InitConfig()
		if err == nil && strings.TrimSpace(cfg.Application.AuthServer.JwtKey) != "" {
			jwtKey = []byte(strings.TrimSpace(cfg.Application.AuthServer.JwtKey))
			return
		}

		jwtKey = []byte("default_jwt_key_change_in_production")
		log.Printf("JWT密钥未配置，正在使用开发默认密钥；生产环境请配置 KV_IOT_JWT_KEY")
	})
	return jwtKey
}

// GenerateToken 生成授权后的token
func GenerateToken(user data.User, roles, permissions []string) (string, error) {
	nowTime := time.Now()
	tokenTimeout := 30 // 默认30分钟
	cfg, err := config.InitConfig()
	if err == nil && cfg.Application.AuthServer.TokenTimeout > 0 {
		tokenTimeout = int(cfg.Application.AuthServer.TokenTimeout)
	} else if err != nil {
		log.Printf("获取JWT配置失败，使用默认过期时间: %v", err)
	}
	expireTime := nowTime.Add(time.Duration(tokenTimeout) * time.Minute)

	roles = uniqueStrings(roles)
	permissions = uniqueStrings(permissions)
	if user.IsAdmin() {
		permissions = appendUnique(permissions, "*")
	}

	claims := Claims{
		Username:    user.UserName,
		Roles:       roles,
		Permissions: permissions,
		IsAdmin:     user.IsAdmin(),
		LegacyAdmin: user.IsAdmin(),
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
	token, err := tokenClaims.SignedString(currentJWTKey())

	if err == nil {
		log.Printf("为用户 %s 生成JWT令牌成功，过期时间: %v", user.UserName, expireTime)
	} else {
		log.Printf("生成JWT令牌失败: %v", err)
	}

	return token, err
}

func ParseToken(token string) (*Claims, error) {
	if strings.TrimSpace(token) == "" {
		return nil, ErrorTokenMalformed
	}
	tokenClaims, err := jwt.ParseWithClaims(token, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// 验证签名算法
		if token.Method != jwt.SigningMethodHS256 {
			log.Printf("无效的签名方法: %v", token.Header["alg"])
			return nil, ErrorTokenMalformed
		}
		return currentJWTKey(), nil
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

// TokenFromRequest prefers the standard Authorization: Bearer header and
// keeps the old raw Authorization/token headers working during migration.
func TokenFromRequest(c *gin.Context) string {
	authorization := strings.TrimSpace(c.GetHeader("Authorization"))
	if authorization != "" {
		parts := strings.Fields(authorization)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return strings.TrimSpace(parts[1])
		}
		if len(parts) == 1 {
			return parts[0]
		}
	}
	return strings.TrimSpace(c.GetHeader("token"))
}

func uniqueStrings(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func appendUnique(values []string, value string) []string {
	for _, item := range values {
		if item == value {
			return values
		}
	}
	return append(values, value)
}

func setJWTKeyForTesting(value string) {
	jwtKeyOnce = sync.Once{}
	jwtKey = []byte(value)
	jwtKeyOnce.Do(func() {})
}
