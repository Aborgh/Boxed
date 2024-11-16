package handlers

import (
	"Boxed/internal/services"
	"github.com/gofiber/fiber/v2"
	"net/http"
	"strings"
)

type FileHandler struct {
	service services.FileService
}

func NewFileHandler(service services.FileService) *FileHandler {
	return &FileHandler{service: service}
}

func (h *FileHandler) UploadFile(c *fiber.Ctx) error {
	boxName := c.Params("box")
	filePath := c.Params("*")

	box, err := h.service.FindBoxByPath(boxName)
	if err != nil || box == nil {
		return c.Status(http.StatusBadRequest).JSON(map[string]interface{}{"error": "Box not found"})
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(map[string]interface{}{"error": "Invalid file"})
	}

	flat := c.Query("flat") == "true"

	item, err := h.service.CreateFileStructure(box, filePath, fileHeader, flat)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(map[string]interface{}{"error": err.Error()})
	}

	return c.Status(http.StatusCreated).JSON(item)
}

func (h *FileHandler) ListFileOrFolder(c *fiber.Ctx) error {
	boxName := c.Params("box")
	itemPath := c.Params("*")
	itemPath = strings.TrimLeft(itemPath, "/")

	item, err := h.service.ListFileOrFolder(boxName, itemPath)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(map[string]interface{}{"error": err.Error()})
	}
	return c.Status(http.StatusOK).JSON(item)
}
