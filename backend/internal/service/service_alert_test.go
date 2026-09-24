package service

import (
	"testing"
	"time"

	"github.com/blueship581/cybuildprice/backend/internal/constants"
	"github.com/blueship581/cybuildprice/backend/internal/model"
	"gorm.io/gorm"
)

func TestAlertMatch(t *testing.T) {
	cases := []struct {
		name        string
		baseline    float64
		target      float64
		newPrice    float64
		dropPercent float64
		want        bool
	}{
		{"exact drop and below target", 100, 95, 90, 10, true},
		{"drop reached but above target", 100, 80, 89, 10, false},
		{"below target but drop too small", 100, 95, 96, 10, false},
		{"no drop required percent zero within target", 100, 100, 100, 0, true},
		{"new price higher than baseline", 100, 200, 110, 10, false},
		{"zero baseline cannot compare", 0, 10, 1, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := AlertMatch(tc.baseline, tc.target, tc.newPrice, tc.dropPercent); got != tc.want {
				t.Fatalf("AlertMatch(%v,%v,%v,%v) = %v, want %v", tc.baseline, tc.target, tc.newPrice, tc.dropPercent, got, tc.want)
			}
		})
	}
}

func TestDropPercent(t *testing.T) {
	cases := []struct {
		baseline float64
		price    float64
		want     float64
	}{
		{100, 90, 10},
		{200, 160, 20},
		{100, 100, 0},
		{0, 10, 0},
	}
	for _, tc := range cases {
		if got := DropPercent(tc.baseline, tc.price); got != tc.want {
			t.Fatalf("DropPercent(%v,%v) = %v, want %v", tc.baseline, tc.price, got, tc.want)
		}
	}
}

func TestBuildAlertView(t *testing.T) {
	triggeredAt := time.Date(2026, 9, 24, 10, 0, 0, 0, time.Local)
	alert := model.PriceAlert{
		Status:          constants.AlertStatusTriggered,
		TargetPrice:     360,
		DropPercent:     10,
		BaselinePrice:   400,
		TriggeredPrice:  350,
		TriggerSupplier: "筑家优选旗舰店",
		TriggeredAt:     &triggeredAt,
		Product:         model.Product{Name: "岩板", Brand: "石界", ModelNumber: "YB-1", Unit: "片"},
	}
	t.Run("no valid offer is not comparable", func(t *testing.T) {
		view := buildAlertView(alert, model.Offer{})
		if view.Comparable {
			t.Fatal("expected not comparable when no valid offer exists")
		}
		if view.TriggeredPrice != 350 || view.TriggeredSupplier != "筑家优选旗舰店" {
			t.Fatalf("trigger snapshot lost: %+v", view)
		}
	})
	t.Run("valid offer shows current price and drop", func(t *testing.T) {
		current := model.Offer{Model: gorm.Model{ID: 9}, UnitPrice: 320, Supplier: model.Supplier{Name: "森木地板仓"}}
		view := buildAlertView(alert, current)
		if !view.Comparable || view.CurrentPrice != 320 || view.CurrentSupplier != "森木地板仓" {
			t.Fatalf("unexpected current view: %+v", view)
		}
		if view.CurrentDrop != 20 {
			t.Fatalf("CurrentDrop = %v, want 20", view.CurrentDrop)
		}
	})
}
