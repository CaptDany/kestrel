package engine

import (
	"testing"
	"github.com/CaptDany/kestrel/internal/db"
)

func TestSortItems(t *testing.T) {
	items := []db.Item{
		{Price: pf(100), CreatedAt: "2026-01-03"},
		{Price: pf(50), CreatedAt: "2026-01-01"},
		{Price: pf(200), CreatedAt: "2026-01-02"},
	}

	sortItems(items, "price_asc")
	if *items[0].Price != 50 || *items[1].Price != 100 || *items[2].Price != 200 {
		t.Errorf("price_asc sort failed: %v", items)
	}

	sortItems(items, "price_desc")
	if *items[0].Price != 200 || *items[1].Price != 100 || *items[2].Price != 50 {
		t.Errorf("price_desc sort failed: %v", items)
	}
}

func TestSortByDateAdded(t *testing.T) {
	items := []db.Item{
		{Price: pf(100), CreatedAt: "2026-01-03"},
		{Price: pf(50), CreatedAt: "2026-01-01"},
		{Price: pf(200), CreatedAt: "2026-01-02"},
	}

	sortItems(items, "date_added")
	if items[0].CreatedAt != "2026-01-01" || items[1].CreatedAt != "2026-01-02" || items[2].CreatedAt != "2026-01-03" {
		t.Errorf("date_added sort failed")
	}
}

func TestSortByPriority(t *testing.T) {
	items := []db.Item{
		{Price: pf(100), Priority: 0, CreatedAt: "2026-01-03"},
		{Price: pf(50), Priority: 1, CreatedAt: "2026-01-01"},
		{Price: pf(200), Priority: -1, CreatedAt: "2026-01-02"},
	}

	sortItems(items, "priority")
	if items[0].Priority != 1 || items[1].Priority != 0 || items[2].Priority != -1 {
		t.Errorf("priority sort failed: %+v", items)
	}
}

func pf(f float64) *float64 { return &f }
