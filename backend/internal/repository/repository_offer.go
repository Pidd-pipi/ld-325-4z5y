package repository

import (
	"fmt"
	"time"

	"github.com/blueship581/cybuildprice/backend/internal/constants"
	"github.com/blueship581/cybuildprice/backend/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type OfferRepository interface {
	ListByProduct(uint) ([]model.Offer, error)
	UpdateStatus(uint, string) (model.Offer, error)
	// LowestValidByProducts returns the cheapest valid offer (approved shop,
	// in stock) per product for the given product ids.
	LowestValidByProducts([]uint) (map[uint]model.Offer, error)
	// FindSubmitted finds an existing offer carrying the same business
	// version as the submitted tuple.
	FindSubmitted(productID, supplierID uint, unitPrice float64, moq, deliveryDays int, freight string) (model.Offer, error)
	// SubmitOfferAndEvaluate persists a new offer and, inside the same
	// transaction, fires every eligible active alert. fires lists the
	// alerts that matched; duplicate (alert, offer) rows are ignored.
	SubmitOfferAndEvaluate(model.Offer, []AlertFire) (model.Offer, []model.AlertEvent, error)
}

// AlertFire is one alert candidate resolved by the service layer: the
// repository only performs the state transition and event insert.
type AlertFire struct {
	AlertID   uint
	ProductID uint
	UserID    string
	UnitPrice float64
}

type offerRepository struct{ db *gorm.DB }

func NewOfferRepository(db *gorm.DB) OfferRepository { return &offerRepository{db} }

func (r *offerRepository) ListByProduct(id uint) ([]model.Offer, error) {
	var data []model.Offer
	if err := r.db.Preload("Supplier").Where("product_id = ?", id).Order("unit_price ASC").Find(&data).Error; err != nil {
		return nil, fmt.Errorf("list offers: %w", err)
	}
	return data, nil
}

func (r *offerRepository) UpdateStatus(id uint, status string) (model.Offer, error) {
	var item model.Offer
	if err := r.db.First(&item, id).Error; err != nil {
		return item, fmt.Errorf("find offer: %w", err)
	}
	item.StockStatus = status
	if err := r.db.Save(&item).Error; err != nil {
		return item, fmt.Errorf("update offer: %w", err)
	}
	return item, nil
}

// validOffersFor loads offers for the given products that belong to an
// approved supplier and are currently in stock; the per-product minimum
// is selected in Go so the query stays portable across dialects.
func (r *offerRepository) LowestValidByProducts(productIDs []uint) (map[uint]model.Offer, error) {
	result := map[uint]model.Offer{}
	if len(productIDs) == 0 {
		return result, nil
	}
	var rows []model.Offer
	err := r.db.Preload("Supplier").
		Joins("JOIN suppliers ON suppliers.id = offers.supplier_id").
		Where("offers.product_id IN ? AND offers.stock_status = ? AND suppliers.status = ?",
			productIDs, constants.StatusInStock, constants.SupplierApproved).
		Order("offers.unit_price ASC").
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("load valid offers: %w", err)
	}
	for _, row := range rows {
		if _, exists := result[row.ProductID]; !exists {
			result[row.ProductID] = row
		}
	}
	return result, nil
}

func (r *offerRepository) FindSubmitted(productID, supplierID uint, unitPrice float64, moq, deliveryDays int, freight string) (model.Offer, error) {
	var item model.Offer
	err := r.db.Where("product_id = ? AND supplier_id = ? AND unit_price = ? AND moq = ? AND delivery_days = ? AND freight = ?",
		productID, supplierID, unitPrice, moq, deliveryDays, freight).First(&item).Error
	if err == gorm.ErrRecordNotFound {
		return item, nil
	}
	if err != nil {
		return item, fmt.Errorf("find submitted offer: %w", err)
	}
	return item, nil
}

func (r *offerRepository) SubmitOfferAndEvaluate(offer model.Offer, fires []AlertFire) (model.Offer, []model.AlertEvent, error) {
	events := []model.AlertEvent{}
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&offer).Error; err != nil {
			return fmt.Errorf("create offer: %w", err)
		}
		now := time.Now()
		for _, fire := range fires {
			// Atomically claim the alert: only an active row flips, so
			// concurrent qualifying offers cannot double-trigger it.
			result := tx.Model(&model.PriceAlert{}).
				Where("id = ? AND status = ?", fire.AlertID, constants.AlertStatusActive).
				Updates(map[string]any{
					"status":             constants.AlertStatusTriggered,
					"triggered_offer_id": offer.ID,
					"triggered_at":       now,
				})
			if result.Error != nil {
				return fmt.Errorf("trigger alert: %w", result.Error)
			}
			if result.RowsAffected == 0 {
				continue
			}
			event := model.AlertEvent{AlertID: fire.AlertID, OfferID: offer.ID, ProductID: fire.ProductID, UserID: fire.UserID, Price: fire.UnitPrice}
			// Unique (alert_id, offer_id): a repeated submission of the
			// same offer version is ignored at the storage boundary too.
			created := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&event)
			if created.Error != nil {
				return fmt.Errorf("create alert event: %w", created.Error)
			}
			if created.RowsAffected > 0 {
				events = append(events, event)
			}
		}
		return nil
	})
	if err != nil {
		return model.Offer{}, nil, err
	}
	return offer, events, nil
}
