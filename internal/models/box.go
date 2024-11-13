package models

import (
	"encoding/json"
)

type Box struct {
	BaseModel
	Name       string          `gorm:"type:varchar(255);not null;unique" json:"name"`
	Properties json.RawMessage `gorm:"type:jsonb" json:"properties,omitempty"`
	Items      []Item          `gorm:"foreignKey:BoxID" json:"items,omitempty"`
}
