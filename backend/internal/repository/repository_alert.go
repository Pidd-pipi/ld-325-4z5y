package repository

import (
	"errors"
	"fmt"
	"time"

	"github.com/blueship581/cybuildprice/backend/internal/constants"
	"github.com/blueship581/cybuildprice/backend/internal/model"
	"gorm.io/gorm"
)

type AlertRepository interface {
	Create(*gorm.DB, *model.PriceAlert) error
	ListByUser(userID string) ([]model.PriceAlert, error)
	ListActiveByProduct(tx *gorm.DB, productID uint) ([]model.PriceAlert, error)
	MarkTriggered(tx *gorm.DB, alertID, offerID uint, price float64, supplier string, at time.Time) error
	CreateEvent(tx *gorm.DB, event *model.AlertEvent) error
	EventExists(tx *gorm.DB, alertID, offerID uint) (bool, error)
}

type alertRepository struct{ db *gorm.DB }

func NewAlertRepository(db *gorm.DB) AlertRepository { return &alertRepository{db} }

func (r *alertRepository) Create(tx *gorm.DB, value *model.PriceAlert) error {
	if err := tx.Create(value).Error; err != nil {
		return fmt.Errorf("create alert: %w", err)
	}
	return nil
}

func (r *alertRepository) ListByUser(userID string) ([]model.PriceAlert, error) {
	var rows []model.PriceAlert
	if err := r.db.Preload("Product").Where("user_id = ?", userID).
		Order("status ASC, created_at DESC").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list alerts: %w", err)
	}
	return rows, nil
}

func (r *alertRepository) ListActiveByProduct(tx *gorm.DB, productID uint) ([]model.PriceAlert, error) {
	var rows []model.PriceAlert
	if err := tx.Where("product_id = ? AND status = ?", productID, constants.AlertStatusActive).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list active alerts: %w", err)
	}
	return rows, nil
}

func (r *alertRepository) MarkTriggered(tx *gorm.DB, alertID, offerID uint, price float64, supplier string, at time.Time) error {
	values := map[string]any{
		"status":           constants.AlertStatusTriggered,
		"triggered_at":     at,
		"triggered_price":  price,
		"trigger_offer_id": offerID,
		"trigger_supplier": supplier,
	}
	if err := tx.Model(&model.PriceAlert{}).Where("id = ?", alertID).Updates(values).Error; err != nil {
		return fmt.Errorf("mark alert triggered: %w", err)
	}
	return nil
}

// CreateEvent stores the in-site notification. A duplicate (alert, offer) pair
// is swallowed because it means the same offer version was submitted twice.
func (r *alertRepository) CreateEvent(tx *gorm.DB, event *model.AlertEvent) error {
	if err := tx.Create(event).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil
		}
		return fmt.Errorf("create alert event: %w", err)
	}
	return nil
}

func (r *alertRepository) EventExists(tx *gorm.DB, alertID, offerID uint) (bool, error) {
	var count int64
	if err := tx.Model(&model.AlertEvent{}).Where("alert_id = ? AND offer_id = ?", alertID, offerID).Count(&count).Error; err != nil {
		return false, fmt.Errorf("count alert events: %w", err)
	}
	return count > 0, nil
}
