package handlers

import (
	"Boxed/internal/helpers"
	"Boxed/internal/services"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type FileHandler struct {
	fileService  services.FileService
	moverService services.MoverService
}

func NewFileHandler(service services.FileService, moverService services.MoverService) *FileHandler {
	return &FileHandler{fileService: service, moverService: moverService}
}

func (h *FileHandler) DeleteFile(c *fiber.Ctx) error {
	boxParam := c.Params("box")
	itemParam := c.Params("*")
	forceParam := c.Query("force", "false")
	force, err := strconv.ParseBool(forceParam)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Unable to parse force parameter")
	}
	if itemParam == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Please set item parameter")
	}
	if boxParam == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Please set box parameter")
	}
	box, err := h.fileService.FindBoxByPath(boxParam)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Box not found")
	}
	item, err := h.fileService.GetFileItem(box, itemParam)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Item not found")
	}
	return h.fileService.AddForDeletion(item, force)
}

func (h *FileHandler) UploadFile(c *fiber.Ctx) error {
	boxName := c.Params("box")
	filePath := c.Params("*")
	properties := c.FormValue("properties")

	box, err := h.fileService.FindBoxByPath(boxName)
	if err != nil || box == nil {
		return c.Status(http.StatusBadRequest).JSON(map[string]interface{}{"error": "Box not found"})
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(map[string]interface{}{"error": "Invalid file"})
	}

	flat := c.Query("flat") == "true"

	item, err := h.fileService.CreateFileStructure(box, filePath, fileHeader, flat, properties)
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
		box, err := h.fileService.FindBoxByPath(boxName)
		if err != nil {
			return c.Status(http.StatusBadRequest).JSON(map[string]interface{}{"error": "Box not found"})
		}
		item, err := h.fileService.GetFileItem(box, itemPath)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(map[string]interface{}{"error": err.Error()})
		}
		return c.Status(http.StatusOK).JSON(item.Properties)
	}

	item, err := h.fileService.ListFileOrFolder(boxName, itemPath)
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

	box, err := h.fileService.FindBoxByPath(boxName)
	if err != nil || box == nil {
		return c.Status(http.StatusBadRequest).JSON(map[string]interface{}{"error": "Box not found"})
	}

	item, err := h.fileService.GetFileItem(box, filePath)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(map[string]interface{}{"error": err.Error()})
	}

	if item.Type == "folder" {
		return c.Status(http.StatusNotFound).JSON(map[string]interface{}{"error": "Not a file"})
	}

	// For hash-based storage, construct the path based on the hash
	firstHashPrefix := item.SHA256[:2]
	secondHashPrefix := item.SHA256[2:4]
	hashFilePath := filepath.Join(box.Path, firstHashPrefix, secondHashPrefix, item.SHA256)

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
	box, err := h.fileService.FindBoxByPath(boxParam)
	if err != nil || box == nil {
		return c.Status(http.StatusBadRequest).JSON(map[string]interface{}{"error": "Box not found"})
	}
	item, err := h.fileService.GetFileItem(box, itemParam)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(map[string]interface{}{"error": err.Error()})
	}
	updatedItem, err := h.fileService.UpdateItem(item)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(map[string]interface{}{"error": err.Error()})
	}
	return c.Status(http.StatusOK).JSON(updatedItem)
}

func (h *FileHandler) CopyOrMoveItem(c *fiber.Ctx) error {
	var req struct {
		To         string `json:"to"`
		Properties string `json:"properties"`
		Force      bool   `json:"force"`
	}
	boxParam := c.Params("box")
	itemParam := c.Params("*")
	action := c.Query("action")
	err := c.BodyParser(&req)
	if err != nil {
		return err
	}
	if action == "copy" {
		err := h.moverService.CopyItem(filepath.Join(boxParam, itemParam), req.To)
		if err != nil {
			return c.Status(http.StatusBadRequest).JSON(map[string]interface{}{"error": err.Error()})
		}
	}
	if action == "move" {

	}

	return c.Status(http.StatusOK).JSON(map[string]interface{}{"message": "ok"})
}
