package repository

import (
	"fmt"

	"github.com/blueship581/cybuildprice/backend/internal/constants"
	"github.com/blueship581/cybuildprice/backend/internal/model"
	"gorm.io/gorm"
)

type OfferRepository interface {
	ListByProduct(uint) ([]model.Offer, error)
	UpdateStatus(uint, string) (model.Offer, error)
	Create(*gorm.DB, *model.Offer) error
	FindSameVersion(tx *gorm.DB, productID, supplierID uint, unitPrice float64, moq, deliveryDays int, freight string) (*model.Offer, error)
	CheapestValidByProducts(productIDs []uint) (map[uint]model.Offer, error)
	CheapestValidOffer(productID uint) (*model.Offer, error)
}

type offerRepository struct{ db *gorm.DB }

func NewOfferRepository(db *gorm.DB) OfferRepository { return &offerRepository{db} }

// validOfferFilter matches offers that count toward the market price:
// the shop has been approved and the quote is currently in stock.
func validOfferFilter(query *gorm.DB) *gorm.DB {
	return query.Joins("JOIN suppliers ON suppliers.id = offers.supplier_id AND suppliers.deleted_at IS NULL").
		Where("suppliers.status = ?", constants.SupplierApproved).
		Where("offers.stock_status = ?", constants.StatusInStock)
}

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

func (r *offerRepository) Create(tx *gorm.DB, value *model.Offer) error {
	if err := tx.Create(value).Error; err != nil {
		return fmt.Errorf("create offer: %w", err)
	}
	return nil
}

// FindSameVersion returns the most recent non-deleted offer from the same shop
// with identical commercial terms. Repeated submissions return that row so no
// new offer version is produced.
func (r *offerRepository) FindSameVersion(tx *gorm.DB, productID, supplierID uint, unitPrice float64, moq, deliveryDays int, freight string) (*model.Offer, error) {
	var offer model.Offer
	query := tx.Where("product_id = ? AND supplier_id = ? AND unit_price = ? AND moq = ? AND delivery_days = ? AND freight = ?",
		productID, supplierID, unitPrice, moq, deliveryDays, freight).
		Order("id DESC")
	err := query.First(&offer).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find same offer version: %w", err)
	}
	return &offer, nil
}

// CheapestValidByProducts returns the cheapest currently-valid offer for each
// requested product. Products without a valid offer are simply absent from
// the map, which the service renders as "not comparable".
func (r *offerRepository) CheapestValidByProducts(productIDs []uint) (map[uint]model.Offer, error) {
	result := make(map[uint]model.Offer)
	if len(productIDs) == 0 {
		return result, nil
	}
	var offers []model.Offer
	if err := validOfferFilter(r.db.Model(&model.Offer{}).Preload("Supplier")).
		Where("offers.product_id IN ?", productIDs).
		Order("offers.unit_price ASC").
		Find(&offers).Error; err != nil {
		return nil, fmt.Errorf("list cheapest valid offers: %w", err)
	}
	for _, offer := range offers {
		if _, exists := result[offer.ProductID]; !exists {
			result[offer.ProductID] = offer
		}
	}
	return result, nil
}

func (r *offerRepository) CheapestValidOffer(productID uint) (*model.Offer, error) {
	var offer model.Offer
	err := validOfferFilter(r.db.Model(&model.Offer{}).Preload("Supplier")).
		Where("offers.product_id = ?", productID).
		Order("offers.unit_price ASC").
		First(&offer).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find cheapest valid offer: %w", err)
	}
	return &offer, nil
}
