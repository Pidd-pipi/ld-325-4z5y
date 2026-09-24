package dto

type SubmitOfferRequest struct {
	ProductID    uint    `json:"product_id" validate:"required,gt=0"`
	SupplierID   uint    `json:"supplier_id" validate:"required,gt=0"`
	UnitPrice    float64 `json:"unit_price" validate:"required,gt=0"`
	MOQ          int     `json:"moq" validate:"required,gte=1"`
	Freight      string  `json:"freight" validate:"required,max=60"`
	DeliveryDays int     `json:"delivery_days" validate:"required,gte=1,lte=90"`
	StockStatus  string  `json:"stock_status" validate:"required,oneof=in_stock out_of_stock discontinued"`
}
