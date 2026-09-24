package model

import (
	"time"

	"gorm.io/gorm"
)

type Favorite struct {
	gorm.Model
	UserID    string
	ProductID uint
	Folder    string
	Product   Product
}

// PriceAlert records one user subscription. BaselinePrice is the lowest
// valid (shop-approved, in-stock) offer at subscription time. An alert
// transitions from active to triggered exactly once, when a newly
// submitted offer matches the subscription terms.
type PriceAlert struct {
	gorm.Model
	UserID           string
	ProductID        uint
	Product          Product
	TargetPrice      float64
	DropPercent      float64
	Status           string
	BaselinePrice    float64
	TriggeredAt      *time.Time
	TriggeredOfferID *uint
	TriggeredOffer   *Offer
}

// AlertEvent is the notification record produced for a triggered alert.
// The (alert_id, offer_id) unique index guarantees that one offer version
// submitting repeatedly never yields a second notification.
type AlertEvent struct {
	gorm.Model
	AlertID   uint `gorm:"uniqueIndex:idx_alert_offer"`
	OfferID   uint `gorm:"uniqueIndex:idx_alert_offer"`
	ProductID uint
	UserID    string
	Price     float64
}

type Budget struct {
	gorm.Model
	UserID   string
	RoomType string
	Area     float64
	Estimate float64
	Payload  string
}
