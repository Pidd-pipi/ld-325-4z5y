package model

import "gorm.io/gorm"

func AllModels() []any {
	return []any{&Category{}, &Product{}, &Supplier{}, &Offer{}, &PriceHistory{}, &Favorite{}, &PriceAlert{}, &AlertEvent{}, &Budget{}}
}
func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(AllModels()...); err != nil {
		return err
	}
	// Subscriptions created before the lifecycle shipped have no status;
	// treat them as in-progress so they remain visible and evaluable.
	return db.Model(&PriceAlert{}).Where("status = ''").Update("status", "active").Error
}
