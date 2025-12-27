package models

import "time"

type BaseModel struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"-" gorm:"autoUpdateTime"`
	CreateBy  string
	UpdateBy  string
	IsDeleted int8 `gorm:"index"`
}
