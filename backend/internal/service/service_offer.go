package service

import (
	"errors"
	"fmt"

	"github.com/blueship581/cybuildprice/backend/internal/constants"
	"github.com/blueship581/cybuildprice/backend/internal/dto"
	apperrors "github.com/blueship581/cybuildprice/backend/internal/errors"
	"github.com/blueship581/cybuildprice/backend/internal/model"
	"github.com/blueship581/cybuildprice/backend/internal/repository"
	"gorm.io/gorm"
)

type OfferService struct {
	repo      repository.OfferRepository
	suppliers repository.SupplierRepository
	products  repository.ProductRepository
	alerts    repository.AlertRepository
}

func NewOfferService(repo repository.OfferRepository, suppliers repository.SupplierRepository, products repository.ProductRepository, alerts repository.AlertRepository) *OfferService {
	return &OfferService{repo: repo, suppliers: suppliers, products: products, alerts: alerts}
}

func (s *OfferService) List(productID uint) ([]model.Offer, error) {
	rows, err := s.repo.ListByProduct(productID)
	if err != nil {
		return nil, fmt.Errorf("list offer service: %w", err)
	}
	return rows, nil
}

func (s *OfferService) UpdateStatus(id uint, status string) (model.Offer, error) {
	return s.repo.UpdateStatus(id, status)
}

// Submit handles a supplier price submission. A repeated submission of the
// same offer version (same product, supplier, price and terms) is idempotent:
// it returns the stored version and never re-fires alerts. Otherwise a new
// offer version is persisted and, when the shop is approved and the offer is
// in stock, every active subscription meeting its drop/target terms fires
// exactly once.
func (s *OfferService) Submit(input dto.SubmitOfferRequest) (model.Offer, []model.AlertEvent, error) {
	supplier, err := s.suppliers.Get(input.SupplierID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.Offer{}, nil, fmt.Errorf("%w: 供应商不存在", apperrors.ErrNotFound)
		}
		return model.Offer{}, nil, err
	}
	if _, err := s.products.Get(input.ProductID); err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			return model.Offer{}, nil, fmt.Errorf("%w: 商品不存在", apperrors.ErrNotFound)
		}
		return model.Offer{}, nil, err
	}
	if existing, err := s.repo.FindSubmitted(input.ProductID, input.SupplierID, input.UnitPrice, input.MOQ, input.DeliveryDays, input.Freight); err != nil {
		return model.Offer{}, nil, err
	} else if existing.ID != 0 {
		// Same offer version submitted again: single record, no new alerts.
		return existing, nil, nil
	}

	offer := model.Offer{
		ProductID:    input.ProductID,
		SupplierID:   input.SupplierID,
		UnitPrice:    input.UnitPrice,
		MOQ:          input.MOQ,
		Freight:      input.Freight,
		DeliveryDays: input.DeliveryDays,
		StockStatus:  constants.StatusInStock,
	}

	// Only approved shops with in-stock quotes can trigger subscribers.
	fires := []repository.AlertFire{}
	if supplier.Status == constants.SupplierApproved {
		subscriptions, err := s.alerts.ListActiveByProduct(input.ProductID)
		if err != nil {
			return model.Offer{}, nil, err
		}
		for _, subscription := range subscriptions {
			if offerMatches(subscription, offer.UnitPrice) {
				fires = append(fires, repository.AlertFire{
					AlertID:   subscription.ID,
					ProductID: subscription.ProductID,
					UserID:    subscription.UserID,
					UnitPrice: offer.UnitPrice,
				})
			}
		}
	}
	saved, events, err := s.repo.SubmitOfferAndEvaluate(offer, fires)
	if err != nil {
		return model.Offer{}, nil, err
	}
	return saved, events, nil
}

// offerMatches applies the subscription terms: the drop from the baseline
// reaches the subscribed percentage and the new price is at or below target.
func offerMatches(alert model.PriceAlert, price float64) bool {
	if alert.BaselinePrice <= 0 || price >= alert.BaselinePrice {
		return false
	}
	drop := (alert.BaselinePrice - price) / alert.BaselinePrice * 100
	return drop >= alert.DropPercent && price <= alert.TargetPrice
}
