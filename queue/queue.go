package queue

import (
	"context"

	log "github.com/Sirupsen/logrus"
	"github.com/gazoon/go-utils"
	"github.com/gazoon/go-utils/logging"
	"github.com/gazoon/go-utils/mongo"
	"github.com/gazoon/go-utils/request"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/pkg/errors"
	"github.com/satori/go.uuid"
)

const (
	maxProcessingTime = 20000
)

type MongoWriter struct {
	client *mgo.Database
}

func NewMongoWriter(settings *utils.MongoDBSettings) (*MongoWriter, error) {

	client, err := mongo.ConnectDatabase(settings)
	if err != nil {
		return nil, err
	}
	return &MongoWriter{client: client}, nil
}

func (self *MongoWriter) Put(ctx context.Context, queueName string, chatId int, message interface{}) error {
	collection := self.client.C(queueName)
	messageEnvelope := map[string]interface{}{
		"created_at": utils.TimestampMilliseconds(),
		"payload":    message,
		"request_id": request.FromContext(ctx),
	}
	_, err := collection.Upsert(
		bson.M{"chat_id": chatId},
		bson.M{
			"$set":  bson.M{"chat_id": chatId},
			"$push": bson.M{"msgs": messageEnvelope},
		})
	return errors.Wrap(err, "add message to the queue")
}

type Document struct {
	ChatID int `bson:"chat_id"`
	Msgs   []*struct {
		CreatedAt int         `bson:"created_at"`
		Payload   interface{} `bson:"payload"`
		RequestId string      `bson:"request_id"`
	} `bson:"msgs"`
	Processing struct {
		StartedAt int    `bson:"started_at"`
		Id        string `bson:"id"`
	} `bson:"processing"`
}

type ReadyMessage struct {
	Payload      interface{}
	RequestId    string // for tracing purposes
	ProcessingId string // used to identify process currently processing chat message
}

func (self ReadyMessage) String() string {
	return utils.ObjToString(&self)
}

type MongoReader struct {
	*logging.LoggerMixin
	client *mgo.Collection
}

func NewMongoReader(settings *utils.MongoDBSettings) (*MongoReader, error) {
	collection, err := mongo.ConnectCollection(settings)
	if err != nil {
		return nil, err
	}
	logger := logging.NewLoggerMixin("mongo_queue_reader", nil)
	return &MongoReader{client: collection, LoggerMixin: logger}, nil
}

func (self *MongoReader) GetAndRemoveNext(ctx context.Context) (*ReadyMessage, error) {
	var doc Document
	currentTime := utils.TimestampMilliseconds()
	processingID := uuid.NewV4().String()
	_, err := self.client.Find(
		bson.M{
			"$or": []bson.M{
				{"processing.started_at": bson.M{"$exists": false}},
				{"processing.started_at": bson.M{"$lt": currentTime - maxProcessingTime}},
			}}).Sort("msgs.0.created_at").Apply(
		mgo.Change{Update: bson.M{
			"$set": bson.M{"processing": bson.M{"started_at": currentTime, "id": processingID}},
			"$pop": bson.M{"msgs": -1},
		}},
		&doc)
	if err == mgo.ErrNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, errors.Wrap(err, "get and remove next message document")
	}
	logger := self.GetLogger(ctx).WithFields(
		log.Fields{"chat_id": doc.ChatID, "processing_id": processingID})
	if len(doc.Msgs) == 0 {
		logger.Warn("Got document without messages, finish processing")
		self.FinishProcessing(ctx, processingID)
		return nil, nil
	}
	message := doc.Msgs[0]
	if doc.Processing.StartedAt != 0 && doc.Processing.StartedAt < currentTime-maxProcessingTime {
		logger.Warn("Previous processing for chat took to long")
	}
	return &ReadyMessage{
		Payload:      utils.ConvertBsonToMap(message.Payload),
		RequestId:    message.RequestId,
		ProcessingId: processingID}, nil
}

func (self *MongoReader) FinishProcessing(ctx context.Context, processingID string) error {
	err := self.client.Remove(bson.M{"msgs": []interface{}{}, "processing.id": processingID})
	if err != nil && err != mgo.ErrNotFound {
		return errors.Wrap(err, "remove document after processing")
	}
	if err == nil {
		return nil
	}

	err = self.client.Update(
		bson.M{"processing.id": processingID},
		bson.M{"$unset": bson.M{"processing": ""}},
	)
	if err == mgo.ErrNotFound {
		logger := self.GetLogger(ctx).WithField("processing_id", processingID)
		logger.Warn("Message document with such processing id no longer exists")
		return nil
	}
	return errors.Wrap(err, "reset document processing id")
}
