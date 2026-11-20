package service

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
)

const DeviceShareScope = "device:mobile"

// DeviceShareClaims authorizes a mobile page to read one device and execute
// its bound actions. It deliberately carries no management permissions.
type DeviceShareClaims struct {
	DeviceID int64  `json:"device_id"`
	Scope    string `json:"scope"`
	jwt.StandardClaims
}

func GenerateDeviceShareToken(deviceID int64, ttl time.Duration) (string, time.Time, error) {
	if deviceID <= 0 {
		return "", time.Time{}, errors.New("device id must be positive")
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}

	now := time.Now()
	expiresAt := now.Add(ttl)
	claims := DeviceShareClaims{
		DeviceID: deviceID,
		Scope:    DeviceShareScope,
		StandardClaims: jwt.StandardClaims{
			Audience:  DeviceShareScope,
			ExpiresAt: expiresAt.Unix(),
			IssuedAt:  now.Unix(),
			Issuer:    "kv-iot",
			NotBefore: now.Add(-10 * time.Second).Unix(),
			Subject:   fmt.Sprintf("device:%d", deviceID),
			Id:        strconv.FormatInt(now.UnixNano(), 10),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(currentJWTKey())
	if err != nil {
		return "", time.Time{}, err
	}
	return token, expiresAt, nil
}

func ParseDeviceShareToken(raw string) (*DeviceShareClaims, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, ErrorTokenMalformed
	}
	parsed, err := jwt.ParseWithClaims(raw, &DeviceShareClaims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, ErrorTokenMalformed
		}
		return currentJWTKey(), nil
	})
	if err != nil {
		if validationError, ok := err.(*jwt.ValidationError); ok {
			switch {
			case validationError.Errors&jwt.ValidationErrorExpired != 0:
				return nil, ErrorTokenExpired
			case validationError.Errors&jwt.ValidationErrorNotValidYet != 0:
				return nil, ErrorTokenNotValidYet
			}
		}
		return nil, ErrorTokenMalformed
	}
	claims, ok := parsed.Claims.(*DeviceShareClaims)
	if !ok || !parsed.Valid || claims.DeviceID <= 0 || claims.Scope != DeviceShareScope {
		return nil, ErrorTokenMalformed
	}
	return claims, nil
}
