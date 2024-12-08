package services

import (
	"Boxed/internal/config"
	"Boxed/internal/models"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type MoverService interface{}

type MoverServiceImpl struct {
	itemService   ItemService
	boxService    BoxService
	configuration config.Configuration
	logService    LogService
}

func NewMoverServiceImpl(
	itemService ItemService,
	boxService BoxService,
	configuration config.Configuration,
	logService LogService,
) *MoverServiceImpl {
	return &MoverServiceImpl{
		itemService:   itemService,
		boxService:    boxService,
		configuration: configuration,
		logService:    logService,
	}
}

type ProgressWriter struct {
	Writer       io.Writer
	BytesWritten int64
	ProgressChan chan int64
}

func (pw *ProgressWriter) Write(p []byte) (int, error) {
	n, err := pw.Writer.Write(p)
	if err != nil {
		return n, err
	}
	pw.BytesWritten += int64(n)
	if pw.ProgressChan != nil {
		pw.ProgressChan <- pw.BytesWritten
	}
	return n, nil
}

func (m *MoverServiceImpl) CopyItem(sourcePath string, destinationPath string) error {
	item, box, err := m.getItemAndBox(sourcePath)
	if err != nil {
		return err
	}

	sourceDir := filepath.Join(m.configuration.Storage.Path, box.Name, item.Path)
	srcItem, err := os.Open(sourceDir)
	if err != nil {
		return err
	}
	defer srcItem.Close()

	dstItem, err := os.Create(destinationPath)
	if err != nil {
		return err
	}
	defer dstItem.Close()

	progressChan := make(chan int64)
	defer close(progressChan)

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				progress := <-progressChan
				m.logService.Log.Info(item.ID, progress)
			}
		}
	}()

	pw := &ProgressWriter{
		Writer:       dstItem,
		ProgressChan: progressChan,
	}

	_, err = io.Copy(pw, srcItem)
	if err != nil {
		return err
	}

	m.logService.Log.Info(item.ID, pw.BytesWritten)
	return nil
}

func (m *MoverServiceImpl) MoveItem(sourcePath string, destinationPath string) error {
	item, _, err := m.getItemAndBox(sourcePath)
	if err != nil {
		return err
	}
	if item.Type == "folder" {
		// TODO: Move folder
	}
	if item.Type == "file" {
		// TODO: Move file
	}
	// TODO: Add moving job to database
	return nil
}

func (m *MoverServiceImpl) getItemAndBox(sourcePath string) (*models.Item, *models.Box, error) {
	cleanSource := filepath.Clean(sourcePath)
	boxAndItemPath := strings.SplitN(cleanSource, string(filepath.Separator), 2)
	if boxName := boxAndItemPath[0]; boxName != "" {
		return nil, nil, errors.New("invalid path: top-level directory (boxName) is missing")
	}
	if itemPath := boxAndItemPath[1]; itemPath != "" {
		return nil, nil, errors.New("invalid path: path to item is missing")
	}
	boxName := boxAndItemPath[0]
	itemPath := boxAndItemPath[1]
	box, err := m.boxService.GetBoxByPath(boxName)
	if err != nil {
		return nil, nil, err
	}
	item, err := m.itemService.FindByPathAndBoxId(itemPath, box.ID)
	if err != nil {
		return nil, nil, err
	}
	return item, box, nil
}
