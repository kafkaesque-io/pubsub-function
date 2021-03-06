package db

import (
	"errors"

	"github.com/kafkaesque-io/pubsub-function/src/model"

	log "github.com/sirupsen/logrus"
)

// dbConn is a singlton of Db instance
var dbConn Db

// Crud interface specifies typical CRUD opertaions for database
type Crud interface {
	GetByTopic(topicFullName, pulsarURL string) (*model.FunctionConfig, error)
	GetByKey(hashedTopicKey string) (*model.FunctionConfig, error)
	Update(topicCfg *model.FunctionConfig) (string, error)
	Create(topicCfg *model.FunctionConfig) (string, error)
	Delete(topicFullName, pulsarURL string) (string, error)
	DeleteByKey(hashedTopicKey string) (string, error)

	// Load is invoked by the webhook.go to start new wekbooks and stop deleted ones
	Load() ([]*model.FunctionConfig, error)
}

// Ops interface specifies required database access operations
type Ops interface {
	Init() error
	Sync() error
	Close() error
	Health() bool
}

// Db interface embeds two other database interfaces
type Db interface {
	Crud
	Ops
}

// NewDb is a database factory pattern to create a new database
func NewDb(reqDbType string) (Db, error) {
	if dbConn != nil {
		log.Infof("return existing db %s", reqDbType)
		return dbConn, nil
	}

	var err error
	switch reqDbType {
	case "pulsarAsDb":
		dbConn, err = NewPulsarHandler()
	case "inmemory":
		dbConn, err = NewInMemoryHandler()
	default:
		err = errors.New("unsupported db type")
	}
	return dbConn, err
}

// NewDbWithPanic ensures a database is returned panic otherwise
func NewDbWithPanic(reqDbType string) Db {
	newDb, err := NewDb(reqDbType)
	if err != nil {
		log.Fatalf("init db with error %s", err.Error())
	}
	return newDb
}

// DocNotFound means no document found in the database
var DocNotFound = "no document found"

// DocAlreadyExisted means document already existed in the database when a new creation is requested
var DocAlreadyExisted = "document already existed"

func getKey(cfg *model.FunctionConfig) (string, error) {
	return cfg.Tenant + cfg.Name, nil
}
