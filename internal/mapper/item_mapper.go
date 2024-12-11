package mapper

import (
	"Boxed/internal/cmd"
	"Boxed/internal/dto"
	"Boxed/internal/models"
	"encoding/json"
)

func ToItemGetDTO(item *models.Item) (*dto.ItemGetDTO, error) {
	var props map[string]interface{}
	if item.Properties != nil {
		err := json.Unmarshal(item.Properties, &props)
		if err != nil {
			return nil, err
		}
	}

	childrenDTOs := make([]*dto.ItemGetDTO, 0, len(item.Children))
	for _, child := range item.Children {
		childDto, err := ToItemGetDTO(&child)
		if err != nil {
			return nil, err
		}
		childrenDTOs = append(childrenDTOs, childDto)
	}
	itemDTO := &dto.ItemGetDTO{
		ID:         item.ID,
		ParentID:   item.ParentID,
		BoxID:      item.BoxID,
		Name:       item.Name,
		Path:       cmd.LtreeToPath(item.Path),
		Type:       item.Type,
		Size:       item.Size,
		Properties: props,
		Children:   childrenDTOs,
		Extension:  item.Extension,
	}
	return itemDTO, nil
}

func ToItemModel(d dto.ItemGetDTO) (*models.Item, error) {
	props, err := json.Marshal(d.Properties)
	if err != nil {
		return nil, err
	}

	childrenItems := make([]models.Item, 0, len(d.Children))
	for _, childDTO := range d.Children {
		childItem, err := ToItemModel(*childDTO)
		if err != nil {
			return nil, err
		}
		childrenItems = append(childrenItems, *childItem)
	}

	return &models.Item{

		BaseModel: models.BaseModel{
			ID: d.ID,
		},
		ParentID:   d.ParentID,
		BoxID:      d.BoxID,
		Name:       d.Name,
		Path:       cmd.PathToLtree(d.Path),
		Type:       d.Type,
		Size:       d.Size,
		Properties: props,
		Children:   childrenItems,
		Extension:  d.Extension,
	}, nil
}

func ToItemsGetDTOs(items []models.Item) ([]dto.ItemGetDTO, error) {
	itemsGetDTOS := make([]dto.ItemGetDTO, 0, len(items))
	for _, item := range items {
		itemGetDTO, err := ToItemGetDTO(&item)
		if err != nil {
			return nil, err
		}
		itemsGetDTOS = append(itemsGetDTOS, *itemGetDTO)
	}
	return itemsGetDTOS, nil
}
