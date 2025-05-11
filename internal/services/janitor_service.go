package services

import (
	"Boxed/internal/config"
	"errors"
	"fmt"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
	"sync"
)

type Janitor struct {
	itemService   ItemService
	boxService    BoxService
	fileService   FileService
	configuration *config.Configuration
	logService    LogService
	cleaning      bool
	mutex         sync.Mutex
	stopChan      chan struct{}
	cron          *cron.Cron
}

func NewJanitorService(
	itemService ItemService,
	boxService BoxService,
	fileService FileService,
	logService LogService,
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
	if j.cleaning {
		j.mutex.Unlock()
		return errors.New("cleaning is in progress")
	}
	j.cleaning = true
	j.mutex.Unlock()

	// Run the cleaning process
	go func() {
		defer func() {
			j.mutex.Lock()
			j.cleaning = false
			j.mutex.Unlock()
		}()
		j.startClean(true)
	}()

	return nil
}

func (j *Janitor) StartCleanCycle() {
	j.logService.Log.Debug("starting cleaning job")
	j.mutex.Lock()
	if j.cleaning {
		j.mutex.Unlock()
		return
	}
	j.mutex.Unlock()

	cronSchedule := j.configuration.Server.CleanConfig.Schedule
	_, err := j.cron.AddFunc(cronSchedule, func() {
		j.mutex.Lock()
		if j.cleaning {
			j.mutex.Unlock()
			return
		}
		j.cleaning = true
		j.mutex.Unlock()

		defer func() {
			j.mutex.Lock()
			j.cleaning = false
			j.mutex.Unlock()
		}()
		j.startClean(false)
	})

	if err != nil {
		j.logService.Log.WithFields(logrus.Fields{
			"job":   "clean",
			"error": err.Error(),
		}).Error("Failed to start cleaning job")
	}
	j.cron.Start()
}

func (j *Janitor) StopClean() {
	j.mutex.Lock()
	defer j.mutex.Unlock()

	if !j.cleaning {
		return
	}

	j.cron.Stop()

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

func (j *Janitor) startClean(forced bool) {
	j.logService.Log.Debug("getting deleted items")
	items, err := j.itemService.FindDeleted()
	j.logService.Log.Debug(fmt.Sprintf("found %d items", len(items)))
	if err != nil {
		j.logService.Log.WithFields(logrus.Fields{
			"job":    "clean",
			"status": "error",
			"error":  err.Error(),
		}).Error("Failed to find deleted items")
	}
	if len(items) > 0 {
		var logFields logrus.Fields
		if !forced {
			logFields = logrus.Fields{
				"job":    "clean",
				"status": "start",
				"cron":   j.configuration.Server.CleanConfig.Schedule,
			}
		} else {
			logFields = logrus.Fields{
				"job":    "clean",
				"status": "forced",
			}
		}
		j.logService.Log.WithFields(logFields).Info(fmt.Sprintf("Found %d items to delete", len(items)))
	}
	var deletedCount int
	for i := range items {
		j.logService.Log.WithFields(logrus.Fields{
			"job":    "clean",
			"status": "deleting",
			"item":   items[i].Name,
			"path":   items[i].Path,
		})
		box, err := j.boxService.GetBoxByID(items[i].BoxID)
		if err != nil {
			j.logService.Log.WithFields(logrus.Fields{
				"job":    "clean",
				"status": "error",
				"error":  err.Error(),
				"item":   items[i].Name,
				"path":   items[i].Path,
				"boxId":  items[i].ID,
			}).Error("Failed to get box")
		}
		err = j.fileService.DeleteItemOnDisk(items[i], box)
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
	j.cleaning = false
}

func (j *Janitor) getDeletedBoxes() {
	boxes, err := j.boxService.GetDeletedBoxes()
	j.logService.Log.WithFields(logrus.Fields{
		"job":   "clean",
		"boxes": boxes,
	})
	if err != nil {
		j.logService.Log.WithFields(logrus.Fields{
			"job":    "clean",
			"status": "error",
			"error":  err.Error(),
		})
		return
	}

}
