package allocator

import (
	"testing"

	"github.com/zpol/vscale/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.IpAllocation{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestAllocateVscaleOverlay(t *testing.T) {
	db := setupTestDB(t)
	a, err := New(db, DefaultGlobalPool, 24)
	if err != nil {
		t.Fatal(err)
	}
	r1, err := a.AllocateVscaleOverlay(1, "")
	if err != nil {
		t.Fatal(err)
	}
	if r1.CIDR != "10.66.0.0/24" {
		t.Fatalf("got %s", r1.CIDR)
	}
	r2, err := a.AllocateVscaleOverlay(2, "")
	if err != nil {
		t.Fatal(err)
	}
	if r2.CIDR != "10.66.1.0/24" {
		t.Fatalf("got %s", r2.CIDR)
	}
}

func TestAllocateHostIP(t *testing.T) {
	db := setupTestDB(t)
	a, err := New(db, DefaultGlobalPool, 24)
	if err != nil {
		t.Fatal(err)
	}
	_, err = a.AllocateVscaleOverlay(1, "10.66.0.0/24")
	if err != nil {
		t.Fatal(err)
	}
	h, err := a.AllocateHostIP(1, "10.66.0.0/24", models.AllocKindHub, 1)
	if err != nil {
		t.Fatal(err)
	}
	if h.CIDR != "10.66.0.1/32" {
		t.Fatalf("hub got %s", h.CIDR)
	}
	d, err := a.AllocateHostIP(1, "10.66.0.0/24", models.AllocKindDevice, 2)
	if err != nil {
		t.Fatal(err)
	}
	if d.CIDR != "10.66.0.2/32" {
		t.Fatalf("device got %s", d.CIDR)
	}
}

func TestRouteConflictWithLocal(t *testing.T) {
	db := setupTestDB(t)
	a, err := New(db, DefaultGlobalPool, 24)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = a.AllocateVscaleOverlay(1, "10.66.0.0/24")
	_, err = a.ReserveRoute(1, "192.168.1.0/24", 1, []string{"192.168.1.0/24"}, false)
	if err == nil {
		t.Fatal("expected conflict with local CIDR")
	}
	_, err = a.ReserveRoute(1, "10.50.0.0/24", 2, []string{"192.168.1.0/24"}, false)
	if err != nil {
		t.Fatal(err)
	}
}

func TestPreferredOverlayConflict(t *testing.T) {
	db := setupTestDB(t)
	a, err := New(db, DefaultGlobalPool, 24)
	if err != nil {
		t.Fatal(err)
	}
	_, err = a.AllocateVscaleOverlay(1, "10.66.0.0/24")
	if err != nil {
		t.Fatal(err)
	}
	_, err = a.AllocateVscaleOverlay(2, "10.66.0.0/24")
	if err == nil {
		t.Fatal("expected preferred conflict")
	}
}
