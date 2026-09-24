package service

import (
	"testing"

	"github.com/blueship581/cybuildprice/backend/internal/constants"
	"github.com/blueship581/cybuildprice/backend/internal/dto"
	"github.com/blueship581/cybuildprice/backend/internal/model"
	"github.com/blueship581/cybuildprice/backend/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func alertFixture(t *testing.T) (*gorm.DB, *AlertService, *OfferService, model.Product, model.Supplier, model.Supplier) {
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
	approved := model.Supplier{Name: "已审核店铺", Status: constants.SupplierApproved, Rating: 4.8}
	pending := model.Supplier{Name: "待审核店铺", Status: constants.SupplierPending, Rating: 4.2}
	db.Create(&approved)
	db.Create(&pending)
	db.Create(&model.Offer{ProductID: product.ID, SupplierID: approved.ID, UnitPrice: 400, MOQ: 1, DeliveryDays: 3, StockStatus: constants.StatusInStock})
	db.Create(&model.Offer{ProductID: product.ID, SupplierID: approved.ID, UnitPrice: 380, MOQ: 1, DeliveryDays: 3, StockStatus: constants.StatusOutOfStock})

	productRepo := repository.NewProductRepository(db)
	offerRepo := repository.NewOfferRepository(db)
	supplierRepo := repository.NewSupplierRepository(db)
	alertRepo := repository.NewAlertRepository(db)
	alertSvc := NewAlertService(alertRepo, offerRepo, productRepo)
	offerSvc := NewOfferService(offerRepo, supplierRepo, productRepo, alertRepo)
	return db, alertSvc, offerSvc, product, approved, pending
}

func TestSubscribeSnapshotsLowestValidBaseline(t *testing.T) {
	_, alerts, _, product, _, _ := alertFixture(t)
	view, err := alerts.Subscribe("u1", dto.CreateAlertRequest{ProductID: product.ID, TargetPrice: 360, DropPercent: 10})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	if view.Baseline != 400 {
		t.Fatalf("baseline = %v, want 400 (out-of-stock 380 must be ignored)", view.Baseline)
	}
	if view.Status != constants.AlertStatusActive || !view.Comparable {
		t.Fatalf("unexpected view: %+v", view)
	}
}

func TestSubscribeRejectedWithoutValidOffer(t *testing.T) {
	db, alerts, _, _, _, _ := alertFixture(t)
	category := model.Category{Name: "门窗"}
	db.Create(&category)
	empty := model.Product{Name: "断桥窗", CategoryID: category.ID}
	db.Create(&empty)
	if _, err := alerts.Subscribe("u1", dto.CreateAlertRequest{ProductID: empty.ID, TargetPrice: 100, DropPercent: 10}); err == nil {
		t.Fatal("expected error when product has no valid offer")
	}
}

func TestOfferSubmitFiresAlertOnce(t *testing.T) {
	db, alerts, offers, product, approved, _ := alertFixture(t)
	if _, err := alerts.Subscribe("u1", dto.CreateAlertRequest{ProductID: product.ID, TargetPrice: 360, DropPercent: 10}); err != nil {
		t.Fatal(err)
	}
	submit := dto.SubmitOfferRequest{ProductID: product.ID, SupplierID: approved.ID, UnitPrice: 350, MOQ: 1, DeliveryDays: 2, Freight: "包邮"}
	_, events, err := offers.Submit(submit)
	if err != nil || len(events) != 1 {
		t.Fatalf("first submit events=%d err=%v", len(events), err)
	}
	// Repeated submission of the same offer version: one row, no new event.
	if _, events2, err := offers.Submit(submit); err != nil || len(events2) != 0 {
		t.Fatalf("duplicate submit events=%d err=%v", len(events2), err)
	}
	var offerCount int64
	if err := db.Model(&model.Offer{}).Where("product_id = ?", product.ID).Count(&offerCount).Error; err != nil {
		t.Fatal(err)
	}
	// 2 seed offers + 1 submitted version.
	if offerCount != 3 {
		t.Fatalf("offer count = %d, want 3", offerCount)
	}
	list, err := alerts.List("u1")
	if err != nil || len(list) != 1 || list[0].Status != constants.AlertStatusTriggered {
		t.Fatalf("list = %+v err=%v", list, err)
	}
	if list[0].Baseline != 400 || list[0].Current != 350 || list[0].TriggeredAt == "" {
		t.Fatalf("triggered view missing fields: %+v", list[0])
	}
	if !list[0].Purchasable {
		t.Fatal("triggering quote should still be purchasable")
	}
}

func TestOfferTermsGateAlerts(t *testing.T) {
	_, alerts, offers, product, approved, pending := alertFixture(t)
	// Drop 10% from 400 needs <= 360; target 340 is stricter.
	if _, err := alerts.Subscribe("u1", dto.CreateAlertRequest{ProductID: product.ID, TargetPrice: 340, DropPercent: 10}); err != nil {
		t.Fatal(err)
	}
	// 350 meets drop but not target: no trigger.
	if _, events, err := offers.Submit(dto.SubmitOfferRequest{ProductID: product.ID, SupplierID: approved.ID, UnitPrice: 350, MOQ: 1, DeliveryDays: 2}); err != nil || len(events) != 0 {
		t.Fatalf("target gate events=%d err=%v", len(events), err)
	}
	// Pending shop quote: no trigger even at the right price.
	if _, events, err := offers.Submit(dto.SubmitOfferRequest{ProductID: product.ID, SupplierID: pending.ID, UnitPrice: 330, MOQ: 1, DeliveryDays: 2}); err != nil || len(events) != 0 {
		t.Fatalf("shop approval gate events=%d err=%v", len(events), err)
	}
	// Approved shop, 5% drop does not reach 10%.
	if _, events, err := offers.Submit(dto.SubmitOfferRequest{ProductID: product.ID, SupplierID: approved.ID, UnitPrice: 390, MOQ: 1, DeliveryDays: 2}); err != nil || len(events) != 0 {
		t.Fatalf("drop gate events=%d err=%v", len(events), err)
	}
	// Approved shop at 330: drop and target both met.
	if _, events, err := offers.Submit(dto.SubmitOfferRequest{ProductID: product.ID, SupplierID: approved.ID, UnitPrice: 330, MOQ: 1, DeliveryDays: 2, Freight: "新价"}); err != nil || len(events) != 1 {
		t.Fatalf("matching submit events=%d err=%v", len(events), err)
	}
}

func TestTriggeredAlertBecomesNotPurchasableWhenStockGone(t *testing.T) {
	db, alerts, offers, product, approved, _ := alertFixture(t)
	if _, err := alerts.Subscribe("u1", dto.CreateAlertRequest{ProductID: product.ID, TargetPrice: 360, DropPercent: 10}); err != nil {
		t.Fatal(err)
	}
	saved, events, err := offers.Submit(dto.SubmitOfferRequest{ProductID: product.ID, SupplierID: approved.ID, UnitPrice: 350, MOQ: 1, DeliveryDays: 2})
	if err != nil || len(events) != 1 {
		t.Fatalf("events=%d err=%v", len(events), err)
	}
	if err := db.Model(&model.Offer{}).Where("id = ?", saved.ID).Update("stock_status", constants.StatusOutOfStock).Error; err != nil {
		t.Fatal(err)
	}
	list, err := alerts.List("u1")
	if err != nil || len(list) != 1 {
		t.Fatalf("list err=%v", err)
	}
	if list[0].Purchasable {
		t.Fatal("alert must not be purchasable once triggering quote goes out of stock")
	}
}
