package repository

import (
	"fmt"

	apperrors "github.com/blueship581/cybuildprice/backend/internal/errors"
	"github.com/blueship581/cybuildprice/backend/internal/model"
	"gorm.io/gorm"
)

type SupplierRepository interface {
	List() ([]model.Supplier, error)
	UpdateStatus(uint, string) (model.Supplier, error)
	Get(id uint) (model.Supplier, error)
}
type supplierRepository struct{ db *gorm.DB }

func NewSupplierRepository(db *gorm.DB) SupplierRepository { return &supplierRepository{db} }
func (r *supplierRepository) List() ([]model.Supplier, error) {
	var rows []model.Supplier
	if err := r.db.Order("rating DESC").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list suppliers: %w", err)
	}
	return rows, nil
}
func (r *supplierRepository) UpdateStatus(id uint, status string) (model.Supplier, error) {
	var row model.Supplier
	if err := r.db.First(&row, id).Error; err != nil {
		return row, fmt.Errorf("find supplier: %w", err)
	}
	row.Status = status
	if err := r.db.Save(&row).Error; err != nil {
		return row, fmt.Errorf("update supplier: %w", err)
	}
	return row, nil
}
func (r *supplierRepository) Get(id uint) (model.Supplier, error) {
	var row model.Supplier
	if err := r.db.First(&row, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return row, apperrors.ErrNotFound
		}
		return row, fmt.Errorf("get supplier: %w", err)
	}
	return row, nil
}
