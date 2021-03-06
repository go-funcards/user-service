package user

import (
	"context"
	"fmt"
	"github.com/go-funcards/slice"
	"github.com/go-funcards/user-service/proto/v1"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

var _ v1.UserServer = (*server)(nil)

type server struct {
	v1.UnimplementedUserServer
	storage Storage
	log     zerolog.Logger
}

func NewUserServer(storage Storage, log zerolog.Logger) *server {
	return &server{
		storage: storage,
		log:     log,
	}
}

func (s *server) CreateUser(ctx context.Context, in *v1.CreateUserRequest) (*emptypb.Empty, error) {
	model := CreateUser(in)

	if err := model.GeneratePasswordHash(); err != nil {
		return nil, err
	}

	err := s.storage.Save(ctx, model)

	return s.empty(err)
}

func (s *server) UpdateUser(ctx context.Context, in *v1.UpdateUserRequest) (*emptypb.Empty, error) {
	model := UpdateUser(in)

	if in.GetOldPassword() != in.GetNewPassword() && len(in.GetNewPassword()) > 0 {
		users, err := s.storage.Find(ctx, Filter{UserIDs: []string{in.GetUserId()}}, 0, 1)
		if err != nil {
			return nil, fmt.Errorf("failed to find user by id, err: %w", err)
		}
		if len(users) != 1 {
			return nil, status.Errorf(codes.NotFound, "user %s not found", in.GetUserId())
		}
		if err = users[0].CheckPassword(in.GetOldPassword()); err != nil {
			return nil, err
		}
		if err = model.GeneratePasswordHash(); err != nil {
			return nil, err
		}
	}

	err := s.storage.Save(ctx, model)

	return s.empty(err)
}

func (s *server) DeleteUser(ctx context.Context, in *v1.DeleteUserRequest) (*emptypb.Empty, error) {
	err := s.storage.Delete(ctx, in.GetUserId())

	return s.empty(err)
}

func (s *server) GetUsers(ctx context.Context, in *v1.UsersRequest) (*v1.UsersResponse, error) {
	filter := Filter{UserIDs: in.GetUserIds(), Emails: in.GetEmails()}

	data, err := s.storage.Find(ctx, filter, in.GetPageIndex(), in.GetPageSize())
	if err != nil {
		return nil, err
	}

	total := uint64(len(data))
	if len(in.GetUserIds()) == 0 && len(in.GetEmails()) == 0 {
		if total, err = s.storage.Count(ctx, filter); err != nil {
			return nil, err
		}
	}

	return &v1.UsersResponse{
		Users: slice.Map(data, func(item User) *v1.UserResponse {
			return item.toProto()
		}),
		Total: total,
	}, nil
}

func (s *server) GetUserByEmailAndPassword(ctx context.Context, in *v1.UserByEmailAndPasswordRequest) (*v1.UserResponse, error) {
	models, err := s.storage.Find(ctx, Filter{Emails: []string{in.GetEmail()}}, 0, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to find user by email, err: %w", err)
	}
	if len(models) != 1 {
		return nil, status.Errorf(codes.NotFound, "user %s not found", in.GetEmail())
	}

	if err = models[0].CheckPassword(in.GetPassword()); err != nil {
		s.log.Error().Err(err).Msg("failed to find user by email")

		return nil, status.Errorf(codes.NotFound, "user %s not found", in.GetEmail())
	}

	return models[0].toProto(), nil
}

func (s *server) empty(err error) (*emptypb.Empty, error) {
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}
