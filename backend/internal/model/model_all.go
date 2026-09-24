package model

import "gorm.io/gorm"

func AllModels() []any {
	return []any{&Category{}, &Product{}, &Supplier{}, &Offer{}, &PriceHistory{}, &Favorite{}, &PriceAlert{}, &AlertEvent{}, &Budget{}}
}

func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(AllModels()...); err != nil {
		return err
	}
	// Subscriptions created before the lifecycle fields existed are treated
	// as ongoing; they simply lack a snapshot baseline.
	return db.Model(&PriceAlert{}).Where("status = ''").Update("status", "active").Error
}
