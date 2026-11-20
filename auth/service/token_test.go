package service

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"kv-iot/auth/data"
)

func TestGenerateAndParseTokenClaims(t *testing.T) {
	setJWTKeyForTesting("test-jwt-key")
	user := data.User{
		Model:       gorm.Model{ID: 7},
		UserName:    "operator",
		IsAdminFlag: false,
	}

	token, err := GenerateToken(
		user,
		[]string{"operator", "operator", ""},
		[]string{"product:read", "command:execute"},
	)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	claims, err := ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken() error = %v", err)
	}
	if claims.Subject != "7" || claims.Username != "operator" {
		t.Fatalf("unexpected claims identity: subject=%q username=%q", claims.Subject, claims.Username)
	}
	if len(claims.Roles) != 1 || claims.Roles[0] != "operator" {
		t.Fatalf("unexpected roles: %#v", claims.Roles)
	}
	if len(claims.Permissions) != 2 {
		t.Fatalf("unexpected permissions: %#v", claims.Permissions)
	}
	if !HasPermission(claims, "product:read") {
		t.Fatal("HasPermission() rejected an assigned permission")
	}
	if HasPermission(claims, "system:manage") {
		t.Fatal("HasPermission() accepted an unassigned permission")
	}
}

func TestAdminTokenReceivesWildcardPermission(t *testing.T) {
	setJWTKeyForTesting("test-jwt-key")
	user := data.User{
		Model:       gorm.Model{ID: 1},
		UserName:    "admin",
		IsAdminFlag: true,
	}
	token, err := GenerateToken(user, nil, nil)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	claims, err := ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken() error = %v", err)
	}
	if !claims.Admin() {
		t.Fatal("admin claim was not preserved")
	}
	if !HasPermission(claims, "anything:read") {
		t.Fatal("admin should bypass permission checks")
	}
}

func TestTokenFromRequestPrefersBearerHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest("GET", "/", nil)
	context.Request.Header.Set("Authorization", "Bearer standard-token")
	context.Request.Header.Set("token", "legacy-token")

	if got := TokenFromRequest(context); got != "standard-token" {
		t.Fatalf("TokenFromRequest() = %q, want standard-token", got)
	}
}

func TestAuthAndPermissionMiddleware(t *testing.T) {
	setJWTKeyForTesting("test-jwt-key")
	gin.SetMode(gin.TestMode)

	user := data.User{
		Model:       gorm.Model{ID: 9},
		UserName:    "operator",
		IsAdminFlag: false,
	}
	token, err := GenerateToken(user, []string{"operator"}, []string{"system:manage"})
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	router := gin.New()
	router.GET("/private", AuthRequired(), PermissionRequired("system:manage"), func(c *gin.Context) {
		c.Status(204)
	})

	noToken := httptest.NewRecorder()
	router.ServeHTTP(noToken, httptest.NewRequest("GET", "/private", nil))
	if noToken.Code != 401 {
		t.Fatalf("no-token status = %d, want 401", noToken.Code)
	}

	authorized := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/private", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(authorized, request)
	if authorized.Code != 204 {
		t.Fatalf("authorized status = %d, want 204", authorized.Code)
	}

	limitedToken, err := GenerateToken(user, nil, []string{"product:read"})
	if err != nil {
		t.Fatalf("GenerateToken() limited error = %v", err)
	}
	forbidden := httptest.NewRecorder()
	request = httptest.NewRequest("GET", "/private", nil)
	request.Header.Set("Authorization", "Bearer "+limitedToken)
	router.ServeHTTP(forbidden, request)
	if forbidden.Code != 403 {
		t.Fatalf("forbidden status = %d, want 403", forbidden.Code)
	}
}
