package models

import (
	"encoding/json"
)

type Item struct {
	BaseModel
	ParentID   *uint           `gorm:"index" json:"parent_id,omitempty"`
	BoxID      uint            `gorm:"index" json:"box_id"`
	Name       string          `gorm:"type:varchar(255);not null" json:"-"`
	Path       string          `gorm:"type:text;not null" json:"path"`
	Type       string          `gorm:"type:varchar(50);not null" json:"type"`
	Size       int64           `gorm:"default:0" json:"size"`
	SHA256     string          `gorm:"type:varchar(64)" json:"sha256,omitempty"`
	SHA512     string          `gorm:"type:varchar(128)" json:"sha512,omitempty"`
	Properties json.RawMessage `gorm:"type:jsonb" json:"properties,omitempty"`
	Children   []Item          `gorm:"-" json:"children,omitempty"`
	Extension  string          `gorm:"type:varchar(20)" json:"extension,omitempty"`
	// For future reference
	//Version    int             `gorm:"default:1" json:"version"`
	//Versions   []Item          `gorm:"foreignKey:ParentID" json:"versions,omitempty"`
}
