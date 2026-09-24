package repository

import (
	"fmt"
	"time"

	"github.com/blueship581/cybuildprice/backend/internal/model"
	"gorm.io/gorm"
)

type PriceHistoryRepository interface {
	List(uint, time.Time) ([]model.PriceHistory, error)
	Create(tx *gorm.DB, value *model.PriceHistory) error
}
type priceHistoryRepository struct{ db *gorm.DB }

func NewPriceHistoryRepository(db *gorm.DB) PriceHistoryRepository {
	return &priceHistoryRepository{db}
}
func (r *priceHistoryRepository) List(productID uint, since time.Time) ([]model.PriceHistory, error) {
	var rows []model.PriceHistory
	if err := r.db.Where("product_id = ? AND recorded_at >= ?", productID, since).Order("recorded_at ASC").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list history: %w", err)
	}
	return rows, nil
}
func (r *priceHistoryRepository) Create(tx *gorm.DB, value *model.PriceHistory) error {
	if err := tx.Create(value).Error; err != nil {
		return fmt.Errorf("create history: %w", err)
	}
	return nil
}
