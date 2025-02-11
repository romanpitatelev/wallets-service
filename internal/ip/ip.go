package ip

import "gorm.io/gorm"

type IP struct {
	gorm.Model
	Address string `json:"address" gorm:"unique"`
	Count   int    `json:"count"`
}
