package user

import "context"

type Storage interface {
	Save(ctx context.Context, model User) error
	Delete(ctx context.Context, id string) error
	Find(ctx context.Context, filter Filter, index uint64, size uint32) ([]User, error)
	Count(ctx context.Context, filter Filter) (uint64, error)
}
