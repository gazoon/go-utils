package mongo

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/gazoon/go-utils"
	"github.com/globalsign/mgo"
	"github.com/pkg/errors"
	"strconv"
	"time"
)

const (
	duplicateKeyCode = 11000
)

func ConnectCollection(settings *utils.MongoDBSettings) (*mgo.Collection, error) {
	db, err := ConnectDatabase(settings)
	if err != nil {
		return nil, err
	}
	if settings.Collection == "" {
		return nil, errors.Errorf("can't connect to mongo collection: %+v", settings)
	}
	return db.C(settings.Collection), nil
}

func ConnectDatabase(settings *utils.MongoDBSettings) (*mgo.Database, error) {
	info := &mgo.DialInfo{
		Addrs:     []string{settings.Host + ":" + strconv.Itoa(settings.Port)},
		Database:  settings.Database,
		Username:  settings.User,
		Password:  settings.Password,
		Timeout:   time.Duration(settings.Timeout) * time.Second,
		PoolLimit: settings.PoolSize,
	}
	log.WithField("dial_info", fmt.Sprintf("%+v", info)).Info("Connecting to mongodb")
	session, err := mgo.DialWithInfo(info)
	if err != nil {
		return nil, errors.Wrap(err, "mongo dial")
	}
	session.SetSafe(&mgo.Safe{WMode: "majority", J: true, WTimeout: settings.Timeout * 1000})
	session.SetMode(mgo.Eventual, true)
	db := session.DB("")
	return db, nil
}

func IsDuplicationErr(err error) bool {
	mgoErr, ok := err.(*mgo.LastError)
	return ok && mgoErr.Code == duplicateKeyCode
}
