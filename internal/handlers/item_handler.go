package handlers

import (
	"Boxed/internal/models"
	"Boxed/internal/services"
	"encoding/json"
	"github.com/gofiber/fiber/v2"
	"net/http"
	"strconv"
)

type ItemHandler struct {
	service services.ItemService
}

func NewItemHandler(service services.ItemService) *ItemHandler {
	return &ItemHandler{service: service}
}

func (h *ItemHandler) CreateItem(c *fiber.Ctx) error {
	var req struct {
		Name       string                 `json:"name"`
		Path       string                 `json:"path"`
		Type       string                 `json:"type"`
		Size       int64                  `json:"size"`
		BoxID      uint                   `json:"box_id"`
		Properties map[string]interface{} `json:"properties"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(map[string]interface{}{"error": err.Error()})
	}
	propertiesJSON, _ := json.Marshal(req.Properties)
	item := &models.Item{Name: req.Name, Path: req.Path, Type: req.Type, Size: req.Size, BoxID: req.BoxID, Properties: propertiesJSON}
	err := h.service.Create(item)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(map[string]interface{}{"error": err.Error()})
	}

	return c.Status(http.StatusCreated).JSON(item)
}

func (h *ItemHandler) GetItemByID(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(map[string]interface{}{"error": "invalid item ID"})
	}

	item, err := h.service.GetItemByID(uint(id))
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(map[string]interface{}{"error": "item not found"})
	}

	return c.JSON(item)
}

func (h *ItemHandler) UpdateItem(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(map[string]interface{}{"error": "invalid item ID"})
	}

	var req struct {
		Name       string                 `json:"name"`
		Path       string                 `json:"path"`
		Properties map[string]interface{} `json:"properties"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(map[string]interface{}{"error": "invalid input"})
	}

	item, err := h.service.UpdateItemPartial(uint(id), req.Name, req.Path, req.Properties)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(map[string]interface{}{"error": "could not update item"})
	}

	return c.JSON(item)
}

func (h *ItemHandler) DeleteItem(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(map[string]interface{}{"error": "invalid item ID"})
	}
	var force bool
	if c.Params("force") != "true" {
		force = true
	}
	if err = h.service.DeleteItem(uint(id), force); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(map[string]interface{}{"error": err.Error()})
	}

	return c.SendStatus(http.StatusNoContent)
}

func (h *ItemHandler) ListItems(c *fiber.Ctx) error {
	items, err := h.service.GetItems()
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(map[string]interface{}{"error": "could not list items"})
	}
	return c.JSON(items)
}

func (h *ItemHandler) ListDeletedItems(c *fiber.Ctx) error {
	items, err := h.service.FindDeleted()
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(map[string]interface{}{"error": "could not list items"})
	}
	return c.JSON(items)
}

func (h *ItemHandler) GetItemTree(c *fiber.Ctx) error {
	idParam := c.Params("id")
	parentID, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(map[string]interface{}{"error": "Invalid ID"})
	}

	levelParam := c.Query("level", "1")
	maxLevel, err := strconv.Atoi(levelParam)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(map[string]interface{}{"error": "Invalid level"})
	}

	itemTree, err := h.service.GetAllDescendants(uint(parentID), maxLevel+1)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(map[string]interface{}{"error": err.Error()})
	}

	return c.JSON(itemTree)
}

func (h *ItemHandler) ItemsSearch(c *fiber.Ctx) error {
	filter := c.Query("$filter", "")
	order := c.Query("$orderby", "id")
	limit, err := strconv.Atoi(c.Query("$limit", "10"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(map[string]interface{}{"error": "Invalid limit"})
	}
	offset, err := strconv.Atoi(c.Query("$skip", "0"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(map[string]interface{}{"error": "Invalid skip"})
	}

	searchResult, err := h.service.ItemsSearch(filter, order, limit, offset)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(map[string]interface{}{"error": err.Error()})
	}
	return c.JSON(searchResult)

}

func (h *ItemHandler) ItemMove(c *fiber.Ctx) error {
	var req struct {
		From  string `json:"from"`
		To    string `json:"to"`
		Force bool   `json:"force"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(map[string]interface{}{"error": err.Error()})
	}

	return nil
}

func (h *ItemHandler) ItemCopy(c *fiber.Ctx) error {
	var req struct {
		From       string `json:"from"`
		To         string `json:"to"`
		Properties string `json:"properties"`
		Force      bool   `json:"force"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(map[string]interface{}{"error": err.Error()})
	}

	return nil
}
