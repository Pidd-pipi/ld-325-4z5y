package router_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/blueship581/cybuildprice/backend/internal/constants"
	"github.com/blueship581/cybuildprice/backend/internal/model"
	"github.com/blueship581/cybuildprice/backend/internal/router"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type envelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func setupServer(t *testing.T) (*httptest.Server, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := model.Migrate(db); err != nil {
		t.Fatal(err)
	}
	approved := model.Supplier{Name: "审核通过店铺", Status: constants.SupplierApproved}
	pending := model.Supplier{Name: "待审核店铺", Status: constants.SupplierPending}
	db.Create(&approved)
	db.Create(&pending)
	product := model.Product{Name: "E2E 岩板", Unit: "片"}
	db.Create(&product)
	db.Create(&model.Offer{ProductID: product.ID, SupplierID: approved.ID, UnitPrice: 400, MOQ: 10, Freight: "包邮", DeliveryDays: 3, StockStatus: constants.StatusInStock})
	engine := router.New(db, slog.New(slog.NewTextHandler(io.Discard, nil)), "test-secret")
	return httptest.NewServer(engine), db
}

func doJSON(t *testing.T, server *httptest.Server, method, path, token string, body any) envelope {
	t.Helper()
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(method, path, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	server.Config.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("%s %s -> %d %s", method, path, rec.Code, rec.Body.String())
	}
	var payload envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Code != 0 {
		t.Fatalf("%s %s business error: %s", method, path, payload.Message)
	}
	return payload
}

func TestAlertLifecycleOverHTTP(t *testing.T) {
	server, _ := setupServer(t)
	defer server.Close()

	// 1. Subscribe as demo user; baseline is the only valid offer (400).
	alertPayload := doJSON(t, server, http.MethodPost, "/api/v1/alerts", "", map[string]any{
		"product_id": 1, "target_price": 360, "drop_percent": 10,
	})
	var alert struct {
		ID            uint    `json:"ID"`
		BaselinePrice float64 `json:"BaselinePrice"`
		Status        string  `json:"Status"`
	}
	if err := json.Unmarshal(alertPayload.Data, &alert); err != nil {
		t.Fatal(err)
	}
	if alert.BaselinePrice != 400 || alert.Status != constants.AlertStatusActive {
		t.Fatalf("unexpected alert: %+v", alert)
	}

	// 2. Obtain a supplier JWT and submit a qualifying quote (350 = -12.5%).
	tokenPayload := doJSON(t, server, http.MethodGet, "/api/v1/demo/supplier-token", "", nil)
	var tokenData struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(tokenPayload.Data, &tokenData); err != nil {
		t.Fatal(err)
	}

	submit := func(price float64, supplierID uint, status string) struct {
		NewVersion     bool `json:"new_version"`
		TriggeredCount int  `json:"triggered_count"`
	} {
		payload := doJSON(t, server, http.MethodPost, "/api/v1/supplier/offers", tokenData.Token, map[string]any{
			"product_id": 1, "supplier_id": supplierID, "unit_price": price,
			"moq": 5, "freight": "包邮", "delivery_days": 2, "stock_status": status,
		})
		var result struct {
			NewVersion     bool `json:"new_version"`
			TriggeredCount int  `json:"triggered_count"`
		}
		if err := json.Unmarshal(payload.Data, &result); err != nil {
			t.Fatal(err)
		}
		return result
	}

	hit := submit(350, 1, constants.StatusInStock)
	if !hit.NewVersion || hit.TriggeredCount != 1 {
		t.Fatalf("expected one trigger, got %+v", hit)
	}
	repeat := submit(350, 1, constants.StatusInStock)
	if repeat.NewVersion || repeat.TriggeredCount != 0 {
		t.Fatalf("duplicate version must be idempotent, got %+v", repeat)
	}

	// 3. Personal center lists the subscription under triggered with fields.
	listPayload := doJSON(t, server, http.MethodGet, "/api/v1/alerts", "", nil)
	var groups map[string][]map[string]any
	if err := json.Unmarshal(listPayload.Data, &groups); err != nil {
		t.Fatal(err)
	}
	triggered := groups[constants.AlertStatusTriggered]
	if len(triggered) != 1 {
		t.Fatalf("expected 1 triggered alert, got %d", len(triggered))
	}
	row := triggered[0]
	if row["baseline_price"].(float64) != 400 || row["triggered_price"].(float64) != 350 {
		t.Fatalf("unexpected triggered row: %+v", row)
	}
	if row["triggered_at"].(string) == "" || row["current_supplier"].(string) != "审核通过店铺" {
		t.Fatalf("missing trigger time or current supplier: %+v", row)
	}
}

func TestPendingSupplierOfferRejected(t *testing.T) {
	server, _ := setupServer(t)
	defer server.Close()
	tokenPayload := doJSON(t, server, http.MethodGet, "/api/v1/demo/supplier-token", "", nil)
	var tokenData struct {
		Token string `json:"token"`
	}
	json.Unmarshal(tokenPayload.Data, &tokenData)

	raw, _ := json.Marshal(map[string]any{
		"product_id": 1, "supplier_id": 2, "unit_price": 300,
		"moq": 5, "freight": "包邮", "delivery_days": 2, "stock_status": "in_stock",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/supplier/offers", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenData.Token)
	rec := httptest.NewRecorder()
	server.Config.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for pending supplier, got %d: %s", rec.Code, rec.Body.String())
	}
}
