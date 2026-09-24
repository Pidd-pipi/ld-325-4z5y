package model

import "gorm.io/gorm"

type Favorite struct {
	gorm.Model
	UserID    string
	ProductID uint
	Folder    string
	Product   Product
}

type Budget struct {
	gorm.Model
	UserID   string
	RoomType string
	Area     float64
	Estimate float64
	Payload  string
}
