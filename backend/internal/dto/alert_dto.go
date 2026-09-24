package dto

type CreateAlertRequest struct {
	ProductID   uint    `json:"product_id" validate:"required,gt=0"`
	TargetPrice float64 `json:"target_price" validate:"required,gt=0"`
	DropPercent float64 `json:"drop_percent" validate:"gte=0,lte=100"`
}

// AlertView is the personal-center representation of a subscription.
type AlertView struct {
	ID                uint    `json:"id"`
	Status            string  `json:"status"`
	ProductID         uint    `json:"product_id"`
	ProductName       string  `json:"product_name"`
	Brand             string  `json:"brand"`
	Model             string  `json:"model"`
	Unit              string  `json:"unit"`
	Thumbnail         string  `json:"thumbnail"`
	TargetPrice       float64 `json:"target_price"`
	DropPercent       float64 `json:"drop_percent"`
	BaselinePrice     float64 `json:"baseline_price"`
	CurrentPrice      float64 `json:"current_price"`
	CurrentSupplier   string  `json:"current_supplier"`
	CurrentDrop       float64 `json:"current_drop"`
	Comparable        bool    `json:"comparable"`
	TriggeredPrice    float64 `json:"triggered_price"`
	TriggeredSupplier string  `json:"triggered_supplier"`
	TriggeredAt       string  `json:"triggered_at"`
	CreatedAt         string  `json:"created_at"`
}
