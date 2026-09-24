package service

import (
	"fmt"

	"github.com/blueship581/cybuildprice/backend/internal/constants"
	"github.com/blueship581/cybuildprice/backend/internal/dto"
	apperrors "github.com/blueship581/cybuildprice/backend/internal/errors"
	"github.com/blueship581/cybuildprice/backend/internal/model"
	"github.com/blueship581/cybuildprice/backend/internal/repository"
	"gorm.io/gorm"
)

// AlertService owns subscription creation, listing and trigger bookkeeping.
type AlertService struct {
	db     *gorm.DB
	alerts repository.AlertRepository
	offers repository.OfferRepository
}

func NewAlertService(db *gorm.DB, alerts repository.AlertRepository, offers repository.OfferRepository) *AlertService {
	return &AlertService{db: db, alerts: alerts, offers: offers}
}

// Subscribe stores the subscription and snapshots the current lowest valid
// offer as the baseline. A product without any valid offer cannot be compared
// yet, so the request is rejected with a client-facing validation error.
func (s *AlertService) Subscribe(user string, input dto.CreateAlertRequest) (model.PriceAlert, error) {
	cheapest, err := s.offers.CheapestValidOffer(input.ProductID)
	if err != nil {
		return model.PriceAlert{}, fmt.Errorf("load baseline offer: %w", err)
	}
	if cheapest == nil {
		return model.PriceAlert{}, apperrors.NewValidation("当前没有审核通过且有货的有效报价，暂无法建立预警基线")
	}
	alert := model.PriceAlert{
		UserID:        user,
		ProductID:     input.ProductID,
		TargetPrice:   input.TargetPrice,
		DropPercent:   input.DropPercent,
		BaselinePrice: cheapest.UnitPrice,
		Status:        constants.AlertStatusActive,
	}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		return s.alerts.Create(tx, &alert)
	}); err != nil {
		return model.PriceAlert{}, fmt.Errorf("create subscription: %w", err)
	}
	return alert, nil
}

// List returns the subscriber's subscriptions split into ongoing and
// triggered groups, enriched with the live cheapest valid offer.
func (s *AlertService) List(user string) (map[string][]dto.AlertView, error) {
	rows, err := s.alerts.ListByUser(user)
	if err != nil {
		return nil, fmt.Errorf("list subscriptions: %w", err)
	}
	productIDs := make([]uint, 0, len(rows))
	for _, row := range rows {
		productIDs = append(productIDs, row.ProductID)
	}
	cheapest, err := s.offers.CheapestValidByProducts(productIDs)
	if err != nil {
		return nil, fmt.Errorf("load current offers: %w", err)
	}
	result := map[string][]dto.AlertView{
		constants.AlertStatusActive:    {},
		constants.AlertStatusTriggered: {},
	}
	for _, row := range rows {
		view := buildAlertView(row, cheapest[row.ProductID])
		result[row.Status] = append(result[row.Status], view)
	}
	return result, nil
}

func buildAlertView(alert model.PriceAlert, current model.Offer) dto.AlertView {
	layout := "2006-01-02 15:04"
	view := dto.AlertView{
		ID:                alert.ID,
		Status:            alert.Status,
		ProductID:         alert.ProductID,
		ProductName:       alert.Product.Name,
		Brand:             alert.Product.Brand,
		Model:             alert.Product.ModelNumber,
		Unit:              alert.Product.Unit,
		Thumbnail:         alert.Product.Thumbnail,
		TargetPrice:       alert.TargetPrice,
		DropPercent:       alert.DropPercent,
		BaselinePrice:     alert.BaselinePrice,
		Comparable:        false,
		TriggeredPrice:    alert.TriggeredPrice,
		TriggeredSupplier: alert.TriggerSupplier,
		CreatedAt:         alert.CreatedAt.Format(layout),
	}
	if current.ID != 0 {
		view.Comparable = true
		view.CurrentPrice = current.UnitPrice
		view.CurrentSupplier = current.Supplier.Name
		view.CurrentDrop = DropPercent(alert.BaselinePrice, current.UnitPrice)
	}
	if alert.TriggeredAt != nil {
		view.TriggeredAt = alert.TriggeredAt.Format(layout)
	}
	return view
}
