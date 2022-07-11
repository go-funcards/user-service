package user

import (
	"context"
	"fmt"
	"github.com/go-funcards/slice"
	"github.com/go-funcards/user-service/proto/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

var _ v1.UserServer = (*userService)(nil)

type userService struct {
	v1.UnimplementedUserServer
	storage Storage
	log     *zap.Logger
}

func NewUserService(storage Storage, logger *zap.Logger) *userService {
	return &userService{
		storage: storage,
		log:     logger,
	}
}

func (s *userService) CreateUser(ctx context.Context, in *v1.CreateUserRequest) (*emptypb.Empty, error) {
	model := CreateUser(in)

	if err := model.GeneratePasswordHash(); err != nil {
		return nil, err
	}

	if err := s.storage.Save(ctx, model); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *userService) UpdateUser(ctx context.Context, in *v1.UpdateUserRequest) (*emptypb.Empty, error) {
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

	if err := s.storage.Save(ctx, model); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *userService) DeleteUser(ctx context.Context, in *v1.DeleteUserRequest) (*emptypb.Empty, error) {
	if err := s.storage.Delete(ctx, in.GetUserId()); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *userService) GetUsers(ctx context.Context, in *v1.UsersRequest) (*v1.UsersResponse, error) {
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
			return item.toResponse()
		}),
		Total: total,
	}, nil
}

func (s *userService) GetUserByEmailAndPassword(ctx context.Context, in *v1.UserByEmailAndPasswordRequest) (*v1.UserResponse, error) {
	models, err := s.storage.Find(ctx, Filter{Emails: []string{in.GetEmail()}}, 0, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to find user by email, err: %w", err)
	}
	if len(models) != 1 {
		return nil, status.Errorf(codes.NotFound, "user %s not found", in.GetEmail())
	}

	if err = models[0].CheckPassword(in.GetPassword()); err != nil {
		s.log.Error("failed to find user by email", zap.Error(err))
		return nil, status.Errorf(codes.NotFound, "user %s not found", in.GetEmail())
	}

	return models[0].toResponse(), nil
}
