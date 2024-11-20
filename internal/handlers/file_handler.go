package handlers

import (
	"Boxed/internal/services"
	"fmt"
	"github.com/gofiber/fiber/v2"
	_ "github.com/gofiber/fiber/v2/utils"
	"net/http"
	"path/filepath"
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
	properties := c.FormValue("properties")

	box, err := h.service.FindBoxByPath(boxName)
	if err != nil || box == nil {
		return c.Status(http.StatusBadRequest).JSON(map[string]interface{}{"error": "Box not found"})
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(map[string]interface{}{"error": "Invalid file"})
	}

	flat := c.Query("flat") == "true"

	item, err := h.service.CreateFileStructure(box, filePath, fileHeader, flat, properties)
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

func (h *FileHandler) DownloadFile(c *fiber.Ctx) error {
	boxName := c.Params("box")
	filePath := c.Params("*")
	filePath = strings.TrimLeft(filePath, "/")

	//filePath = filepath.Clean("/" + filePath)

	if strings.Contains(filePath, "..") {
		return c.Status(http.StatusBadRequest).JSON(map[string]interface{}{"error": "Invalid path"})
	}

	box, err := h.service.FindBoxByPath(boxName)
	if err != nil || box == nil {
		return c.Status(http.StatusBadRequest).JSON(map[string]interface{}{"error": "Box not found"})
	}

	item, err := h.service.GetFileItem(box, filePath)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(map[string]interface{}{"error": err.Error()})
	}

	if item.Type == "folder" {
		return c.Status(http.StatusNotFound).JSON(map[string]interface{}{"error": "Not a file"})
	}

	fullFilePath := filepath.Join(h.service.GetStoragePath(), box.Name, item.Path)
	mimeType := fiber.MIMEOctetStream

	c.Set("Content-Type", mimeType)
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", item.Name))

	// Send the file
	return c.SendFile(fullFilePath)
}
