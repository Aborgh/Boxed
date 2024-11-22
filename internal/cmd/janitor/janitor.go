package janitor

import (
	"Boxed/internal/config"
	"Boxed/internal/services"
	"errors"
	"fmt"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
	"sync"
)

type Janitor struct {
	itemService   services.ItemService
	boxService    services.BoxService
	fileService   services.FileService
	configuration *config.Configuration
	logService    services.LogService
	cleaning      bool
	mutex         sync.Mutex
	stopChan      chan struct{}
	cron          *cron.Cron
}

func NewJanitor(
	itemService services.ItemService,
	boxService services.BoxService,
	fileService services.FileService,
	logService services.LogService,
	configuration *config.Configuration,

) *Janitor {
	return &Janitor{
		itemService:   itemService,
		fileService:   fileService,
		boxService:    boxService,
		logService:    logService,
		cleaning:      false,
		mutex:         sync.Mutex{},
		configuration: configuration,
		cron:          cron.New(),
	}
}

func (j *Janitor) ForceStartCleanCycle() error {
	j.mutex.Lock()
	defer j.mutex.Unlock()
	if j.cleaning {
		return errors.New("cleaning is in progress")
	}
	j.cleaning = true
	j.startClean()
	j.cleaning = false
	return nil
}

func (j *Janitor) StartCleanCycle() {
	j.mutex.Lock()
	defer j.mutex.Unlock()

	if j.cleaning {
		return // Already running
	}

	// Mark as cleaning
	j.cleaning = true
	// Add a cron job for cleaning
	cronSchedule := j.configuration.Server.CleanConfig.Schedule
	_, err := j.cron.AddFunc(cronSchedule, func() {
		j.startClean()
	})

	if err != nil {
		j.logService.Log.WithFields(logrus.Fields{
			"job":   "clean",
			"error": err.Error(),
		}).Error("failed to start cleaning job")
		j.cleaning = false
		return
	}

	j.cron.Start()
}

func (j *Janitor) StopClean() {
	j.mutex.Lock()
	defer j.mutex.Unlock()

	if !j.cleaning {
		return
	}

	// Stop the cron scheduler
	j.cron.Stop()

	// Reset state
	j.cleaning = false
	j.logService.Log.WithFields(logrus.Fields{
		"job":    "clean",
		"status": "stopped",
	}).Info("Janitor clean stopped")
}

func (j *Janitor) IsCleaning() bool {
	j.mutex.Lock()
	defer j.mutex.Unlock()
	return j.cleaning
}

func (j *Janitor) startClean() {
	items, err := j.itemService.FindDeleted()
	if err != nil {
		j.logService.Log.WithFields(logrus.Fields{
			"job":    "clean",
			"status": "error",
			"error":  err.Error(),
		}).Error("Failed to find deleted items")
	}
	if len(items) > 0 {
		j.logService.Log.WithFields(logrus.Fields{
			"job":    "clean",
			"status": "start",
			"cron":   j.configuration.Server.CleanConfig.Schedule,
		}).Info(fmt.Sprintf("Found %d items to delete", len(items)))
	}
	var deletedCount int
	for _, item := range items {
		j.logService.Log.WithFields(logrus.Fields{
			"job":    "clean",
			"status": "deleting",
			"item":   item.Name,
			"path":   item.Path,
		})
		box, err := j.boxService.GetBoxByID(item.BoxID)
		if err != nil {
			j.logService.Log.WithFields(logrus.Fields{
				"job":    "clean",
				"status": "error",
				"error":  err.Error(),
			}).Error("Failed to find box")
		}
		err = j.fileService.DeleteItemOnDisk(item, box)
		if err != nil {
			j.logService.Log.WithFields(logrus.Fields{
				"job":    "clean",
				"status": "error",
				"error":  err.Error(),
			}).Error("Failed to delete item")
		}
		deletedCount++
	}
	if deletedCount > 0 {
		j.logService.Log.WithFields(logrus.Fields{
			"job":    "clean",
			"status": "success",
			"count":  deletedCount,
		}).Info("cleaning job finished")
	}
}
