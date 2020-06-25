package db

import (
	"errors"
	"time"

	"github.com/kafkaesque-io/pubsub-function/src/model"

	log "github.com/sirupsen/logrus"
)

/**
 * An in memory database implmentation of the restful API data store
 * no data is persisted.
 * This is for testing only.
 */

// InMemoryHandler is the in memory cache driver
type InMemoryHandler struct {
	functions map[string]model.FunctionConfig
	logger    *log.Entry
}

//Init is a Db interface method.
func (s *InMemoryHandler) Init() error {
	s.logger = log.WithFields(log.Fields{"app": "inmemory-db"})
	s.functions = make(map[string]model.FunctionConfig)
	return nil
}

//Sync is a Db interface method.
func (s *InMemoryHandler) Sync() error {
	return nil
}

//Health is a Db interface method
func (s *InMemoryHandler) Health() bool {
	return true
}

// Close closes database
func (s *InMemoryHandler) Close() error {
	return nil
}

//NewInMemoryHandler initialize a Mongo Db
func NewInMemoryHandler() (*InMemoryHandler, error) {
	handler := InMemoryHandler{}
	err := handler.Init()
	return &handler, err
}

// Create creates a new document
func (s *InMemoryHandler) Create(functionCfg *model.FunctionConfig) (string, error) {
	key, err := getKey(functionCfg)
	if err != nil {
		return key, err
	}

	if _, ok := s.functions[key]; ok {
		return key, errors.New(DocAlreadyExisted)
	}

	functionCfg.ID = key
	functionCfg.CreatedAt = time.Now()
	functionCfg.UpdatedAt = functionCfg.CreatedAt

	s.functions[functionCfg.ID] = *functionCfg
	log.Infof("created a function %s database size %d", functionCfg.ID, len(s.functions))
	return key, nil
}

// GetByTopic gets a document by the topic name and pulsar URL
func (s *InMemoryHandler) GetByTopic(tenant, functionName string) (*model.FunctionConfig, error) {
	key, err := model.GetKeyFromNames(tenant, functionName)
	if err != nil {
		return &model.FunctionConfig{}, err
	}
	return s.GetByKey(key)
}

// GetByKey gets a document by the key
func (s *InMemoryHandler) GetByKey(hashedTopicKey string) (*model.FunctionConfig, error) {
	if v, ok := s.functions[hashedTopicKey]; ok {
		return &v, nil
	}
	return &model.FunctionConfig{}, errors.New(DocNotFound)
}

// Load loads the entire database as a list
func (s *InMemoryHandler) Load() ([]*model.FunctionConfig, error) {
	results := []*model.FunctionConfig{}
	for _, v := range s.functions {
		results = append(results, &v)
	}
	log.Infof("load database table size %d", len(results))
	return results, nil
}

// Update updates or creates a topic config document
func (s *InMemoryHandler) Update(functionCfg *model.FunctionConfig) (string, error) {
	key, err := getKey(functionCfg)
	if err != nil {
		return key, err
	}

	if _, ok := s.functions[key]; !ok {
		return s.Create(functionCfg)
	}

	v := s.functions[key]
	v.Tenant = functionCfg.Tenant
	v.FunctionStatus = functionCfg.FunctionStatus
	v.UpdatedAt = time.Now()
	// v.Webhooks = functionCfg.Webhooks

	s.logger.Infof("upsert %s", key)
	s.functions[functionCfg.ID] = *functionCfg
	return key, nil

}

// Delete deletes a document
func (s *InMemoryHandler) Delete(tenant, functionName string) (string, error) {
	key, err := model.GetKeyFromNames(tenant, functionName)
	if err != nil {
		return "", err
	}
	return s.DeleteByKey(key)
}

// DeleteByKey deletes a document based on key
func (s *InMemoryHandler) DeleteByKey(hashedTopicKey string) (string, error) {
	if _, ok := s.functions[hashedTopicKey]; !ok {
		return "", errors.New(DocNotFound)
	}

	delete(s.functions, hashedTopicKey)
	return hashedTopicKey, nil
}
