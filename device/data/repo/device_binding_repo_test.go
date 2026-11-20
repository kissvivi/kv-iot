package repo

import (
	"strings"
	"testing"

	"kv-iot/device/data"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newDryRunDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(
		mysql.New(mysql.Config{
			DSN:                       "user:pass@tcp(127.0.0.1:1)/kv_iot?charset=utf8mb4&parseTime=True&loc=Local",
			SkipInitializeWithVersion: true,
		}),
		&gorm.Config{
			DisableAutomaticPing:   true,
			DryRun:                 true,
			Logger:                 logger.Discard,
			SkipDefaultTransaction: true,
		},
	)
	if err != nil {
		t.Fatalf("open dry-run database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql database: %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	return db
}

func captureQuerySQL(t *testing.T, db *gorm.DB) *string {
	t.Helper()

	captured := new(string)
	if err := db.Callback().Query().After("gorm:query").Register("test:capture_query", func(tx *gorm.DB) {
		*captured = tx.Statement.SQL.String()
	}); err != nil {
		t.Fatalf("register query callback: %v", err)
	}
	return captured
}

func captureCreateSQL(t *testing.T, db *gorm.DB) *string {
	t.Helper()

	captured := new(string)
	if err := db.Callback().Create().After("gorm:create").Register("test:capture_create", func(tx *gorm.DB) {
		*captured = tx.Statement.SQL.String()
	}); err != nil {
		t.Fatalf("register create callback: %v", err)
	}
	return captured
}

func TestProductProtocolProfileRepoAddAndFind(t *testing.T) {
	db := newDryRunDB(t)
	repo := NewProductProtocolProfileRepo(db)

	createSQL := captureCreateSQL(t, db)
	if err := repo.Add(data.ProductProtocolProfile{
		ProductID:      1,
		ProtocolType:   "modbus_tcp",
		DriverName:     "modbus_tcp",
		ProfileVersion: "v1",
		Status:         data.ProtocolProfileStatusDraft,
	}); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if sql := *createSQL; !strings.Contains(sql, "INSERT INTO `product_protocol_profiles`") {
		t.Fatalf("create SQL = %q", sql)
	}

	querySQL := captureQuerySQL(t, db)
	err, profiles := repo.FindByProductID(1)
	if err != nil {
		t.Fatalf("FindByProductID() error = %v", err)
	}
	if profiles != nil {
		t.Fatalf("DryRun profiles = %#v, want nil", profiles)
	}
	if sql := *querySQL; !strings.Contains(sql, "WHERE product_id = ?") {
		t.Fatalf("query SQL = %q", sql)
	}
}

func TestProductPointBindingRepoFindByPoint(t *testing.T) {
	db := newDryRunDB(t)
	repo := NewProductPointBindingRepo(db)
	querySQL := captureQuerySQL(t, db)

	err, _ := repo.FindByProfileIDAndPoint(10, data.PointKindProperty, 20)
	if err != nil {
		t.Fatalf("FindByProfileIDAndPoint() error = %v", err)
	}
	if sql := *querySQL; !strings.Contains(sql, "profile_id = ? AND point_kind = ? AND point_id = ?") {
		t.Fatalf("query SQL = %q", sql)
	}
}

func TestDevicePointBindingRepoFindByInstance(t *testing.T) {
	db := newDryRunDB(t)
	repo := NewDevicePointBindingRepo(db)
	querySQL := captureQuerySQL(t, db)

	err, _ := repo.FindByInstance(1, 2, 3)
	if err != nil {
		t.Fatalf("FindByInstance() error = %v", err)
	}
	if sql := *querySQL; !strings.Contains(sql, "device_id = ? AND channel_instance_id = ? AND product_point_binding_id = ?") {
		t.Fatalf("query SQL = %q", sql)
	}
}
