package handlers

import (
	"Boxed/internal/helpers"
	"Boxed/internal/services"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type FileHandler struct {
	service services.FileService
}

func NewFileHandler(service services.FileService) *FileHandler {
	return &FileHandler{service: service}
}

func (h *FileHandler) DeleteFile(c *fiber.Ctx) error {
	itemParam := c.Params("item")
	boxParam := c.Params("box")

	box, err := h.service.FindBoxByPath(boxParam)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Box not found")
	}
	item, err := h.service.GetFileItem(box, itemParam)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Item not found")
	}
	return h.service.DeleteItemOnDisk(*item, box)
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
	// Validate path format
	if err := helpers.ValidatePath(itemPath); err != nil {
		return c.Status(http.StatusBadRequest).JSON(map[string]interface{}{
			"error": err.Error(),
		})
	}

	if _, properties := c.Queries()["properties"]; properties {
		box, err := h.service.FindBoxByPath(boxName)
		if err != nil {
			return c.Status(http.StatusBadRequest).JSON(map[string]interface{}{"error": "Box not found"})
		}
		item, err := h.service.GetFileItem(box, itemPath)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(map[string]interface{}{"error": err.Error()})
		}
		return c.Status(http.StatusOK).JSON(item.Properties)
	}

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

	// For hash-based storage, construct the path based on the hash
	firstHashPrefix := item.SHA256[:2]
	secondHashPrefix := item.SHA256[2:4]
	storageBasePath := h.service.GetStoragePath()
	hashFilePath := filepath.Join(storageBasePath, firstHashPrefix, secondHashPrefix, item.SHA256)

	// Check if the file exists
	if _, err := os.Stat(hashFilePath); os.IsNotExist(err) {
		return c.Status(http.StatusNotFound).JSON(map[string]interface{}{"error": "File content not found"})
	}

	mimeType := fiber.MIMEOctetStream

	// Set the correct content type for the download
	c.Set("Content-Type", mimeType)
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", item.Name))

	return c.SendFile(hashFilePath)
}

func (h *FileHandler) UpdateItem(c *fiber.Ctx) error {
	itemParam := c.Params("*")
	boxParam := c.Params("box")
	box, err := h.service.FindBoxByPath(boxParam)
	if err != nil || box == nil {
		return c.Status(http.StatusBadRequest).JSON(map[string]interface{}{"error": "Box not found"})
	}
	item, err := h.service.GetFileItem(box, itemParam)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(map[string]interface{}{"error": err.Error()})
	}
	updatedItem, err := h.service.UpdateItem(item)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(map[string]interface{}{"error": err.Error()})
	}
	return c.Status(http.StatusOK).JSON(updatedItem)

}
