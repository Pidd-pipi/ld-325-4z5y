package repository

import (
	"fmt"

	"github.com/blueship581/cybuildprice/backend/internal/constants"
	"github.com/blueship581/cybuildprice/backend/internal/model"
	"gorm.io/gorm"
)

type AlertRepository interface {
	Create(model.PriceAlert) (model.PriceAlert, error)
	ListByUser(string) ([]model.PriceAlert, error)
	// ListActiveByProduct returns active subscriptions for a product with
	// their baseline prices.
	ListActiveByProduct(productID uint) ([]model.PriceAlert, error)
}

type alertRepository struct{ db *gorm.DB }

func NewAlertRepository(db *gorm.DB) AlertRepository { return &alertRepository{db} }

func (r *alertRepository) Create(value model.PriceAlert) (model.PriceAlert, error) {
	if err := r.db.Create(&value).Error; err != nil {
		return value, fmt.Errorf("create alert: %w", err)
	}
	return value, nil
}

func (r *alertRepository) ListByUser(user string) ([]model.PriceAlert, error) {
	var rows []model.PriceAlert
	err := r.db.
		Preload("Product").
		Preload("TriggeredOffer.Supplier").
		Where("user_id = ?", user).
		Order("created_at DESC").
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list alerts: %w", err)
	}
	return rows, nil
}

func (r *alertRepository) ListActiveByProduct(productID uint) ([]model.PriceAlert, error) {
	var rows []model.PriceAlert
	if err := r.db.Where("product_id = ? AND status = ?", productID, constants.AlertStatusActive).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list active alerts: %w", err)
	}
	return rows, nil
}
