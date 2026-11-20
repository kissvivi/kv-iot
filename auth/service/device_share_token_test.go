package service

import (
	"testing"
	"time"
)

func TestDeviceShareTokenRoundTrip(t *testing.T) {
	setJWTKeyForTesting("device-share-test-key")

	token, expiresAt, err := GenerateDeviceShareToken(42, time.Hour)
	if err != nil {
		t.Fatalf("GenerateDeviceShareToken() error = %v", err)
	}
	if !expiresAt.After(time.Now()) {
		t.Fatalf("expiresAt = %v, want a future time", expiresAt)
	}
	claims, err := ParseDeviceShareToken(token)
	if err != nil {
		t.Fatalf("ParseDeviceShareToken() error = %v", err)
	}
	if claims.DeviceID != 42 {
		t.Fatalf("DeviceID = %d, want 42", claims.DeviceID)
	}
	if claims.Scope != DeviceShareScope {
		t.Fatalf("Scope = %q, want %q", claims.Scope, DeviceShareScope)
	}
}

func TestDeviceShareTokenRejectsManagementToken(t *testing.T) {
	setJWTKeyForTesting("device-share-test-key")

	token, _, err := GenerateDeviceShareToken(42, time.Hour)
	if err != nil {
		t.Fatalf("GenerateDeviceShareToken() error = %v", err)
	}
	claims, err := ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken() error = %v", err)
	}
	if claims.Admin() || len(claims.Permissions) != 0 {
		t.Fatalf("device share token unexpectedly carries management permissions: %+v", claims)
	}
}
