package cmd

import (
	"Boxed/internal/config"
	"Boxed/internal/repository"
	"fmt"
	"path/filepath"
	"sync"
	"time"
)

type Janitor struct {
	itemRepository repository.ItemRepository
	configuration  *config.Configuration
	cleaning       bool
	mutex          sync.Mutex
	stopChan       chan struct{}
}

func NewJanitor(itemRepository repository.ItemRepository, configuration *config.Configuration) *Janitor {
	return &Janitor{
		itemRepository: itemRepository,
		cleaning:       false,
		mutex:          sync.Mutex{},
		configuration:  configuration,
	}
}

func (j *Janitor) StartClean() {
	j.mutex.Lock()
	defer j.mutex.Unlock()

	j.cleaning = true
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				fmt.Println("Cleaning items")
				j.findItemsToClean()
			case <-j.stopChan:
				fmt.Println("Cleaning items stopped")
				return
			}
		}
	}()
}

func (j *Janitor) StopClean() {
	j.mutex.Lock()
	defer j.mutex.Unlock()
	if !j.cleaning {
		return
	}
	close(j.stopChan)
	j.stopChan = make(chan struct{})
	j.cleaning = false
}

func (j *Janitor) IsCleaning() bool {
	j.mutex.Lock()
	defer j.mutex.Unlock()
	return j.cleaning
}

func (j *Janitor) findItemsToClean() {
	items, err := j.itemRepository.FindDeleted()
	if err != nil {
		println(err.Error())
	}
	for _, item := range items {
		path := filepath.Join(j.configuration.Storage.Path, item.Path)
		println(path)
	}
}
