package db

import (
	"context"
	"fmt"
	"github.com/go-funcards/mongodb"
	"github.com/go-funcards/user-service/internal/user"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

var _ user.Storage = (*storage)(nil)

const (
	timeout    = 5 * time.Second
	collection = "users"
)

type storage struct {
	c   *mongo.Collection
	log logrus.FieldLogger
}

func NewStorage(ctx context.Context, db *mongo.Database, log logrus.FieldLogger) *storage {
	s := &storage{
		c:   db.Collection(collection),
		log: log,
	}
	s.indexes(ctx)
	return s
}

func (s *storage) indexes(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	models := []mongo.IndexModel{
		{Keys: bson.D{{"email", 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{"created_at", 1}}},
	}

	names, err := s.c.Indexes().CreateMany(ctx, models)
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"collection": collection,
			"error":      err,
		}).Fatal("index not created")
	}

	s.log.WithFields(logrus.Fields{
		"collection": collection,
		"name":       names,
	}).Info("index created")
}

func (s *storage) Save(ctx context.Context, model user.User) error {
	data, err := mongodb.ToBson(model)
	if err != nil {
		return err
	}

	delete(data, "_id")
	delete(data, "created_at")

	s.log.WithField("user_id", model.UserID).Info("user save")

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	result, err := s.c.UpdateOne(
		ctx,
		bson.M{"_id": model.UserID},
		bson.M{
			"$set": data,
			"$setOnInsert": bson.M{
				"created_at": model.CreatedAt,
			},
		},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		return fmt.Errorf(fmt.Sprintf("user save: %s", mongodb.ErrMsgQuery), err)
	}

	s.log.WithFields(logrus.Fields{
		"user_id": model.UserID,
		"result":  result,
	}).Info("user saved")

	return nil
}

func (s *storage) Delete(ctx context.Context, id string) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	s.log.WithField("user_id", id).Debug("user delete")
	result, err := s.c.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf(mongodb.ErrMsgQuery, err)
	}
	if result.DeletedCount == 0 {
		return fmt.Errorf(mongodb.ErrMsgQuery, mongo.ErrNoDocuments)
	}
	s.log.WithField("user_id", id).Debug("user deleted")

	return nil
}

func (s *storage) Find(ctx context.Context, filter user.Filter, index uint64, size uint32) ([]user.User, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	opts := mongodb.FindOptions(index, size).SetSort(bson.D{{"created_at", -1}})
	cur, err := s.c.Find(ctx, s.build(filter), opts)
	if err != nil {
		return nil, fmt.Errorf(mongodb.ErrMsgQuery, err)
	}
	return mongodb.DecodeAll[user.User](ctx, cur)
}

func (s *storage) Count(ctx context.Context, filter user.Filter) (uint64, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	total, err := s.c.CountDocuments(ctx, s.build(filter))
	if err != nil {
		return 0, fmt.Errorf(mongodb.ErrMsgQuery, err)
	}
	return uint64(total), nil
}

func (s *storage) build(filter user.Filter) any {
	f := make(mongodb.Filter, 0)
	var ids, emails mongodb.Expr
	if len(filter.UserIDs) > 0 {
		ids = mongodb.In("_id", filter.UserIDs)
	}
	if len(filter.Emails) > 0 {
		emails = mongodb.In("email", filter.Emails)
	}
	if ids != nil && emails != nil {
		f = append(f, mongodb.Or(ids, emails))
	} else if ids != nil {
		f = append(f, ids)
	} else if emails != nil {
		f = append(f, emails)
	}
	return f.Build()
}
