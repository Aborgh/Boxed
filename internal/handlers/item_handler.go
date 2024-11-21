package handlers

import (
	"Boxed/internal/services"
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

	item, err := h.service.CreateItem(req.Name, req.Path, req.Type, req.Size, req.BoxID, req.Properties)
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

	item, err := h.service.UpdateItem(uint(id), req.Name, req.Path, req.Properties)
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

	if err := h.service.DeleteItem(uint(id)); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(map[string]interface{}{"error": "could not delete item"})
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
