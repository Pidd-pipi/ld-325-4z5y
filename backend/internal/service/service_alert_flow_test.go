package service

import (
	"testing"

	"github.com/blueship581/cybuildprice/backend/internal/constants"
	"github.com/blueship581/cybuildprice/backend/internal/dto"
	"github.com/blueship581/cybuildprice/backend/internal/model"
	"github.com/blueship581/cybuildprice/backend/internal/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := model.Migrate(db); err != nil {
		t.Fatal(err)
	}
	return db
}

func seedOfferScenario(t *testing.T, db *gorm.DB) (model.Supplier, model.Supplier, model.Product) {
	t.Helper()
	approved := model.Supplier{Name: "已审核店铺", Status: constants.SupplierApproved, Rating: 4.8}
	pending := model.Supplier{Name: "待审核店铺", Status: constants.SupplierPending, Rating: 4.1}
	if err := db.Create(&approved).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&pending).Error; err != nil {
		t.Fatal(err)
	}
	product := model.Product{Name: "测试岩板", Unit: "片"}
	if err := db.Create(&product).Error; err != nil {
		t.Fatal(err)
	}
	baseline := model.Offer{ProductID: product.ID, SupplierID: approved.ID, UnitPrice: 400, MOQ: 10, Freight: "包邮", DeliveryDays: 3, StockStatus: constants.StatusInStock}
	if err := db.Create(&baseline).Error; err != nil {
		t.Fatal(err)
	}
	return approved, pending, product
}

func newOfferSubmissionService(db *gorm.DB) *OfferSubmissionService {
	return NewOfferSubmissionService(
		db,
		repository.NewOfferRepository(db),
		repository.NewSupplierRepository(db),
		repository.NewProductRepository(db),
		repository.NewAlertRepository(db),
		repository.NewPriceHistoryRepository(db),
	)
}

func TestSubscribeSnapshotsBaseline(t *testing.T) {
	db := newTestDB(t)
	_, _, product := seedOfferScenario(t, db)
	svc := NewAlertService(db, repository.NewAlertRepository(db), repository.NewOfferRepository(db))

	alert, err := svc.Subscribe("user-1", dto.CreateAlertRequest{ProductID: product.ID, TargetPrice: 360, DropPercent: 10})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	if alert.BaselinePrice != 400 || alert.Status != constants.AlertStatusActive {
		t.Fatalf("unexpected baseline snapshot: %+v", alert)
	}
}

func TestSubscribeRejectedWithoutValidOffer(t *testing.T) {
	db := newTestDB(t)
	product := model.Product{Name: "无报价产品"}
	if err := db.Create(&product).Error; err != nil {
		t.Fatal(err)
	}
	svc := NewAlertService(db, repository.NewAlertRepository(db), repository.NewOfferRepository(db))
	if _, err := svc.Subscribe("user-1", dto.CreateAlertRequest{ProductID: product.ID, TargetPrice: 100, DropPercent: 10}); err == nil {
		t.Fatal("expected error when no valid offer exists")
	}
}

func TestSubmitOfferTriggersAlertOncePerVersion(t *testing.T) {
	db := newTestDB(t)
	approved, pending, product := seedOfferScenario(t, db)
	offerSvc := newOfferSubmissionService(db)
	alertSvc := NewAlertService(db, repository.NewAlertRepository(db), repository.NewOfferRepository(db))

	if _, err := alertSvc.Subscribe("user-1", dto.CreateAlertRequest{ProductID: product.ID, TargetPrice: 360, DropPercent: 10}); err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	submit := func(supplierID uint, price float64, status string) SubmitOfferResult {
		t.Helper()
		result, err := offerSvc.Submit(dto.SubmitOfferRequest{
			ProductID: product.ID, SupplierID: supplierID, UnitPrice: price,
			MOQ: 5, Freight: "包邮", DeliveryDays: 2, StockStatus: status,
		})
		if err != nil {
			t.Fatalf("submit offer: %v", err)
		}
		return result
	}

	// Pending shop: rejected outright, no trigger.
	if _, err := offerSvc.Submit(dto.SubmitOfferRequest{
		ProductID: product.ID, SupplierID: pending.ID, UnitPrice: 300,
		MOQ: 5, Freight: "包邮", DeliveryDays: 2, StockStatus: constants.StatusInStock,
	}); err == nil {
		t.Fatal("pending supplier submission must be rejected")
	}

	// Approved shop but out of stock: new version, no trigger.
	outOfStock := submit(approved.ID, 299, constants.StatusOutOfStock)
	if !outOfStock.NewVersion || outOfStock.TriggeredCount != 0 {
		t.Fatalf("out-of-stock offer must not trigger: %+v", outOfStock)
	}

	// Small drop does not meet the 10% threshold.
	smallDrop := submit(approved.ID, 380, constants.StatusInStock)
	if smallDrop.TriggeredCount != 0 {
		t.Fatalf("small drop must not trigger: %+v", smallDrop)
	}

	// Qualifying new in-stock version: exactly one trigger.
	hit := submit(approved.ID, 350, constants.StatusInStock)
	if !hit.NewVersion || hit.TriggeredCount != 1 {
		t.Fatalf("qualifying offer should trigger exactly once: %+v", hit)
	}

	// Re-submitting the same version: no new version, no second event.
	repeat := submit(approved.ID, 350, constants.StatusInStock)
	if repeat.NewVersion || repeat.TriggeredCount != 0 {
		t.Fatalf("duplicate offer version must be idempotent: %+v", repeat)
	}

	var events []model.AlertEvent
	if err := db.Find(&events).Error; err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("expected exactly one alert event, got %d", len(events))
	}

	groups, err := alertSvc.List("user-1")
	if err != nil {
		t.Fatalf("list alerts: %v", err)
	}
	triggered := groups[constants.AlertStatusTriggered]
	if len(triggered) != 1 {
		t.Fatalf("expected one triggered subscription, got %d", len(triggered))
	}
	view := triggered[0]
	if view.BaselinePrice != 400 || view.TriggeredPrice != 350 || view.CurrentPrice != 350 {
		t.Fatalf("unexpected triggered view: %+v", view)
	}
	if !view.Comparable || view.CurrentDrop != 12.5 {
		t.Fatalf("expected comparable 12.5%% drop, got %+v", view)
	}
}
