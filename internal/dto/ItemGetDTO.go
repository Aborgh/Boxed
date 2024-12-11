package dto

type ItemGetDTO struct {
	ID         uint                   `json:"id"`
	ParentID   *uint                  `json:"parent_id,omitempty"`
	BoxID      uint                   `json:"box_id"`
	Name       string                 `json:"name"`
	Path       string                 `json:"path"`
	Type       string                 `json:"type"`
	Size       int64                  `json:"size"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	Children   []*ItemGetDTO          `json:"children,omitempty"`
	Extension  string                 `json:"extension,omitempty"`
}
