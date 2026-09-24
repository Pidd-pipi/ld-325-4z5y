package dto

type CreateAlertRequest struct {
	ProductID   uint    `json:"product_id" validate:"required,gt=0"`
	TargetPrice float64 `json:"target_price" validate:"required,gt=0"`
	DropPercent float64 `json:"drop_percent" validate:"gte=0,lte=100"`
}

type AlertProductView struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
	Unit string `json:"unit"`
}

type AlertView struct {
	ID          uint             `json:"id"`
	ProductID   uint             `json:"product_id"`
	Product     AlertProductView `json:"product"`
	Status      string           `json:"status"`
	Baseline    float64          `json:"baseline_price"`
	Current     float64          `json:"current_price"`
	DropPercent float64          `json:"drop_percent"`
	TargetPrice float64          `json:"target_price"`
	Comparable  bool             `json:"comparable"`
	Purchasable bool             `json:"purchasable"`
	TriggeredAt string           `json:"triggered_at,omitempty"`
	Supplier    string           `json:"supplier,omitempty"`
}
