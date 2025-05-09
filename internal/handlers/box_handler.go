package handlers

import (
	"Boxed/internal/services"
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

type BoxHandler struct {
	service services.BoxService
}

func NewBoxHandler(service services.BoxService) *BoxHandler {
	return &BoxHandler{service: service}
}

func (h *BoxHandler) CreateBox(c *fiber.Ctx) error {
	var req struct {
		Name       string                 `json:"name"`
		Properties map[string]interface{} `json:"properties"`
		Path       string                 `json:"path"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(map[string]interface{}{"error": "invalid input"})
	}

	// Validate required fields
	if req.Name == "" {
		return c.Status(http.StatusBadRequest).JSON(map[string]interface{}{"error": "name is required"})
	}

	// Initialize empty properties if nil
	if req.Properties == nil {
		req.Properties = make(map[string]interface{})
	}

	if req.Path == "" {
		return c.Status(http.StatusBadRequest).JSON(map[string]interface{}{"error": "path is required"})
	}

	box, err := h.service.CreateBox(req.Name, req.Properties, req.Path)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(map[string]interface{}{"error": err.Error()})
	}

	return c.Status(http.StatusCreated).JSON(box)
}

func (h *BoxHandler) GetBoxByID(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(map[string]interface{}{"error": "invalid box ID"})
	}

	box, err := h.service.GetBoxByID(uint(id))
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(map[string]interface{}{"error": "box not found"})
	}

	return c.JSON(box)
}

func (h *BoxHandler) UpdateBox(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(map[string]interface{}{"error": "invalid box ID"})
	}

	var req struct {
		Name       string                 `json:"name"`
		Properties map[string]interface{} `json:"properties"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(map[string]interface{}{"error": "invalid input"})
	}

	// Validate required fields for update
	if req.Name == "" {
		return c.Status(http.StatusBadRequest).JSON(map[string]interface{}{"error": "name is required"})
	}

	box, err := h.service.UpdateBox(uint(id), req.Name, req.Properties)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(map[string]interface{}{"error": "could not update box"})
	}

	return c.JSON(box)
}

func (h *BoxHandler) DeleteBox(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(map[string]interface{}{"error": "invalid box ID"})
	}

	if err := h.service.DeleteBox(uint(id)); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(map[string]interface{}{"error": "could not delete box"})
	}

	return c.SendStatus(http.StatusNoContent)
}

func (h *BoxHandler) ListBoxes(c *fiber.Ctx) error {
	boxes, err := h.service.GetBoxes()
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(map[string]interface{}{"error": "could not list boxes"})
	}
	return c.JSON(boxes)
}
