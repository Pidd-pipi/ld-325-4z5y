package dto

type SubmitOfferRequest struct {
	ProductID    uint    `json:"product_id" validate:"required,gt=0"`
	SupplierID   uint    `json:"supplier_id" validate:"required,gt=0"`
	UnitPrice    float64 `json:"unit_price" validate:"required,gt=0"`
	MOQ          int     `json:"moq" validate:"required,gt=0"`
	Freight      string  `json:"freight" validate:"max=100"`
	DeliveryDays int     `json:"delivery_days" validate:"required,gt=0,lte=365"`
}
