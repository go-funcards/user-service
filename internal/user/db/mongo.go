package db

import (
	"context"
	"github.com/go-funcards/mongodb"
	"github.com/go-funcards/user-service/internal/user"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
	"time"
)

var _ user.Storage = (*storage)(nil)

const (
	timeout    = 5 * time.Second
	collection = "users"
)

type storage struct {
	c mongodb.Collection[user.User]
}

func NewStorage(ctx context.Context, db *mongo.Database, logger *zap.Logger) (*storage, error) {
	s := &storage{c: mongodb.Collection[user.User]{
		Inner: db.Collection(collection),
		Log:   logger,
	}}

	if err := s.indexes(ctx); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *storage) indexes(ctx context.Context) error {
	models := []mongo.IndexModel{
		{Keys: bson.D{{"email", 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{"created_at", 1}}},
	}
	names, err := s.c.Inner.Indexes().CreateMany(ctx, models)
	if err == nil {
		s.c.Log.Info("indexes created", zap.String("collection", collection), zap.Strings("names", names))
	}

	return err
}

func (s *storage) Save(ctx context.Context, model user.User) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	data, err := s.c.ToM(model)
	if err != nil {
		return err
	}

	delete(data, "_id")
	delete(data, "created_at")

	return s.c.UpdateOne(
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
}

func (s *storage) Delete(ctx context.Context, id string) error {
	return s.c.DeleteOne(ctx, bson.M{"_id": id})
}

func (s *storage) Find(ctx context.Context, filter user.Filter, index uint64, size uint32) ([]user.User, error) {
	return s.c.Find(ctx, s.filter(filter), s.c.FindOptions(index, size).SetSort(bson.D{{"created_at", -1}}))
}

func (s *storage) Count(ctx context.Context, filter user.Filter) (uint64, error) {
	return s.c.CountDocuments(ctx, s.filter(filter))
}

func (s *storage) filter(filter user.Filter) bson.M {
	f := make(bson.M)
	if len(filter.UserIDs) > 0 {
		f["_id"] = bson.M{"$in": filter.UserIDs}
	}
	if len(filter.Emails) > 0 {
		f["email"] = bson.M{"$in": filter.Emails}
	}
	return f
}
