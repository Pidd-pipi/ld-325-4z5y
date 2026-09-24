package service

import (
	"fmt"
	"time"

	"github.com/blueship581/cybuildprice/backend/internal/constants"
	"github.com/blueship581/cybuildprice/backend/internal/dto"
	apperrors "github.com/blueship581/cybuildprice/backend/internal/errors"
	"github.com/blueship581/cybuildprice/backend/internal/model"
	"github.com/blueship581/cybuildprice/backend/internal/repository"
	"gorm.io/gorm"
)

// SubmitOfferResult reports whether a submission created a new offer version
// and how many subscriptions fired because of it.
type SubmitOfferResult struct {
	Offer          model.Offer `json:"offer"`
	NewVersion     bool        `json:"new_version"`
	TriggeredCount int         `json:"triggered_count"`
}

// OfferSubmissionService handles supplier repricing and alert evaluation.
type OfferSubmissionService struct {
	db        *gorm.DB
	offers    repository.OfferRepository
	suppliers repository.SupplierRepository
	products  repository.ProductRepository
	alerts    repository.AlertRepository
	history   repository.PriceHistoryRepository
}

func NewOfferSubmissionService(
	db *gorm.DB,
	offers repository.OfferRepository,
	suppliers repository.SupplierRepository,
	products repository.ProductRepository,
	alerts repository.AlertRepository,
	history repository.PriceHistoryRepository,
) *OfferSubmissionService {
	return &OfferSubmissionService{db: db, offers: offers, suppliers: suppliers, products: products, alerts: alerts, history: history}
}

// Submit persists a supplier quote. Identical commercial terms from the same
// shop reuse the previous row, so re-submitting one offer version can never
// generate a second notification. A genuinely new in-stock price from an
// approved shop is evaluated against all active subscriptions of the product.
func (s *OfferSubmissionService) Submit(input dto.SubmitOfferRequest) (SubmitOfferResult, error) {
	supplier, err := s.suppliers.Get(input.SupplierID)
	if err != nil {
		return SubmitOfferResult{}, fmt.Errorf("load supplier: %w", err)
	}
	if supplier.Status != constants.SupplierApproved {
		return SubmitOfferResult{}, apperrors.NewValidation("店铺尚未通过审核，报价暂不生效")
	}
	exists, err := s.products.Exists(input.ProductID)
	if err != nil {
		return SubmitOfferResult{}, fmt.Errorf("check product: %w", err)
	}
	if !exists {
		return SubmitOfferResult{}, apperrors.NewValidation("建材不存在")
	}

	offer := model.Offer{
		ProductID:    input.ProductID,
		SupplierID:   input.SupplierID,
		UnitPrice:    input.UnitPrice,
		MOQ:          input.MOQ,
		Freight:      input.Freight,
		DeliveryDays: input.DeliveryDays,
		StockStatus:  input.StockStatus,
	}
	triggered := 0
	newVersion := false
	err = s.db.Transaction(func(tx *gorm.DB) error {
		same, err := s.offers.FindSameVersion(tx, input.ProductID, input.SupplierID, input.UnitPrice, input.MOQ, input.DeliveryDays, input.Freight)
		if err != nil {
			return err
		}
		if same != nil {
			offer = *same
			return nil
		}
		newVersion = true
		if err := s.offers.Create(tx, &offer); err != nil {
			return err
		}
		if err := s.history.Create(tx, &model.PriceHistory{
			ProductID:  offer.ProductID,
			OfferID:    offer.ID,
			Price:      offer.UnitPrice,
			RecordedAt: time.Now(),
		}); err != nil {
			return err
		}
		// Only approved shops with in-stock quotes can fire an alert.
		if offer.StockStatus != constants.StatusInStock {
			return nil
		}
		subscriptions, err := s.alerts.ListActiveByProduct(tx, offer.ProductID)
		if err != nil {
			return err
		}
		for _, alert := range subscriptions {
			if !AlertMatch(alert.BaselinePrice, alert.TargetPrice, offer.UnitPrice, alert.DropPercent) {
				continue
			}
			duplicated, err := s.alerts.EventExists(tx, alert.ID, offer.ID)
			if err != nil {
				return err
			}
			if duplicated {
				continue
			}
			now := time.Now()
			if err := s.alerts.MarkTriggered(tx, alert.ID, offer.ID, offer.UnitPrice, supplier.Name, now); err != nil {
				return err
			}
			if err := s.alerts.CreateEvent(tx, &model.AlertEvent{
				AlertID:      alert.ID,
				OfferID:      offer.ID,
				ProductID:    offer.ProductID,
				UserID:       alert.UserID,
				Price:        offer.UnitPrice,
				SupplierName: supplier.Name,
			}); err != nil {
				return err
			}
			triggered++
		}
		return nil
	})
	if err != nil {
		return SubmitOfferResult{}, fmt.Errorf("submit offer: %w", err)
	}
	return SubmitOfferResult{Offer: offer, NewVersion: newVersion, TriggeredCount: triggered}, nil
}
