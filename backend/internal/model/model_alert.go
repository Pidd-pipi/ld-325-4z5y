package model

import (
	"gorm.io/gorm"
	"time"
)

// PriceAlert is a user subscription. BaselinePrice captures the lowest valid
// offer at subscription time and is the reference used for drop-percent checks.
type PriceAlert struct {
	gorm.Model
	UserID          string `gorm:"index"`
	ProductID       uint
	Product         Product
	TargetPrice     float64
	DropPercent     float64
	BaselinePrice   float64
	Status          string
	TriggeredAt     *time.Time
	TriggeredPrice  float64
	TriggerOfferID  *uint
	TriggerSupplier string
}

// AlertEvent is the in-site notification generated when an alert fires.
// The unique index guarantees at most one event per subscription and offer version.
type AlertEvent struct {
	gorm.Model
	AlertID      uint   `gorm:"uniqueIndex:idx_alert_offer"`
	OfferID      uint   `gorm:"uniqueIndex:idx_alert_offer"`
	ProductID    uint   `gorm:"index"`
	UserID       string `gorm:"index"`
	Price        float64
	SupplierName string
}
