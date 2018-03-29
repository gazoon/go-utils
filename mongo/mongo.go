package mongo

import (
	log "github.com/Sirupsen/logrus"
	"github.com/gazoon/go-utils"
	"github.com/globalsign/mgo"
	"strconv"
	"time"
)

const (
	duplicateKeyCode = 11000
)

func Connect(settings *utils.MongoDBSettings) (*mgo.Database, error) {
	info := &mgo.DialInfo{
		Addrs:     []string{settings.Host + ":" + strconv.Itoa(settings.Port)},
		Database:  settings.Database,
		Username:  settings.User,
		Password:  settings.Password,
		Timeout:   time.Duration(settings.Timeout) * time.Second,
		PoolLimit: settings.PoolSize,
	}
	log.Infof("Connecting to mongodb. Dial info: %+v", info)
	session, err := mgo.DialWithInfo(info)
	if err != nil {
		return nil, err
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
