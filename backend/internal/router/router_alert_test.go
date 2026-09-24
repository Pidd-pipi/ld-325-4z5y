package router

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/blueship581/cybuildprice/backend/internal/constants"
	"github.com/blueship581/cybuildprice/backend/internal/middleware"
	"github.com/blueship581/cybuildprice/backend/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func testLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

type envelope struct {
	Code int             `json:"code"`
	Data json.RawMessage `json:"data"`
}

func setupRouter(t *testing.T) (*httptest.Server, *gorm.DB, string) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := model.Migrate(db); err != nil {
		t.Fatal(err)
	}
	category := model.Category{Name: "瓷砖"}
	db.Create(&category)
	product := model.Product{Name: "岩板", CategoryID: category.ID}
	db.Create(&product)
	supplier := model.Supplier{Name: "旗舰店", Status: constants.SupplierApproved}
	db.Create(&supplier)
	db.Create(&model.Offer{ProductID: product.ID, SupplierID: supplier.ID, UnitPrice: 400, MOQ: 1, DeliveryDays: 3, StockStatus: constants.StatusInStock})

	// Discard logger noise by using a no-op slog logger.
	engine := New(db, testLogger(), "test-secret")
	server := httptest.NewServer(engine)
	supplierToken, err := middleware.NewDemoToken("test-secret", "supplier-1", constants.RoleSupplier)
	if err != nil {
		t.Fatal(err)
	}
	return server, db, supplierToken
}

func TestAlertLifecycleOverHTTP(t *testing.T) {
	server, db, supplierToken := setupRouter(t)
	defer server.Close()

	// 1. Demo user subscribes; baseline snapshots the 400 in-stock offer.
	body := `{"product_id":1,"target_price":360,"drop_percent":10}`
	resp, err := http.Post(server.URL+"/api/v1/alerts", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	var created envelope
	decode(t, resp, &created)
	var alert map[string]any
	json.Unmarshal(created.Data, &alert)
	if alert["baseline_price"].(float64) != 400 {
		t.Fatalf("baseline = %v, want 400", alert["baseline_price"])
	}

	// 2. Supplier submits 350 (12.5% drop, <= 360 target): one trigger.
	submitBody := `{"product_id":1,"supplier_id":1,"unit_price":350,"moq":1,"delivery_days":2,"freight":"包邮"}`
	req, _ := http.NewRequest(http.MethodPost, server.URL+"/api/v1/supplier/offers", bytes.NewBufferString(submitBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+supplierToken)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	var submitted envelope
	decode(t, resp, &submitted)
	var submitData struct {
		Triggered int `json:"triggered"`
	}
	json.Unmarshal(submitted.Data, &submitData)
	if submitData.Triggered != 1 {
		t.Fatalf("triggered = %d, want 1", submitData.Triggered)
	}

	// 3. Identical re-submission: idempotent, zero triggers.
	req, _ = http.NewRequest(http.MethodPost, server.URL+"/api/v1/supplier/offers", bytes.NewBufferString(submitBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+supplierToken)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	decode(t, resp, &submitted)
	json.Unmarshal(submitted.Data, &submitData)
	if submitData.Triggered != 0 {
		t.Fatalf("duplicate submission triggered = %d, want 0", submitData.Triggered)
	}

	// 4. Personal center shows the alert as triggered with original/current.
	resp, err = http.Get(server.URL + "/api/v1/alerts")
	if err != nil {
		t.Fatal(err)
	}
	var listed envelope
	decode(t, resp, &listed)
	var alerts []map[string]any
	json.Unmarshal(listed.Data, &alerts)
	if len(alerts) != 1 || alerts[0]["status"] != "triggered" {
		t.Fatalf("alerts = %s", listed.Data)
	}
	if alerts[0]["baseline_price"].(float64) != 400 || alerts[0]["current_price"].(float64) != 350 {
		t.Fatalf("prices wrong: %v %v", alerts[0]["baseline_price"], alerts[0]["current_price"])
	}
	if alerts[0]["purchasable"] != true || alerts[0]["triggered_at"] == "" {
		t.Fatalf("triggered view incomplete: %v", alerts[0])
	}

	// 5. Exactly one notification event exists.
	var events int64
	db.Model(&model.AlertEvent{}).Count(&events)
	if events != 1 {
		t.Fatalf("events = %d, want 1", events)
	}
}

func TestSubmitRejectsNonSupplier(t *testing.T) {
	server, _, _ := setupRouter(t)
	defer server.Close()
	resp, err := http.Post(server.URL+"/api/v1/supplier/offers", "application/json",
		bytes.NewBufferString(`{"product_id":1,"supplier_id":1,"unit_price":350,"moq":1,"delivery_days":2}`))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func decode(t *testing.T, resp *http.Response, target any) {
	t.Helper()
	raw, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d: %s", resp.StatusCode, raw)
	}
	if err := json.Unmarshal(raw, target); err != nil {
		t.Fatalf("decode %s: %v", raw, err)
	}
}
