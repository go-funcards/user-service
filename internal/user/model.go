package user

import (
	"errors"
	"fmt"
	"github.com/go-funcards/slice"
	"github.com/go-funcards/user-service/proto/v1"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/protobuf/types/known/timestamppb"
	"time"
)

type User struct {
	UserID    string    `json:"user_id" bson:"_id,omitempty"`
	Name      string    `json:"name" bson:"name,omitempty"`
	Email     string    `json:"email" bson:"email,omitempty"`
	Password  string    `json:"-" bson:"password,omitempty"`
	Roles     []string  `json:"roles" bson:"roles,omitempty"`
	CreatedAt time.Time `json:"created_at" bson:"created_at,omitempty"`
}

type Filter struct {
	UserIDs []string `json:"user_ids,omitempty"`
	Emails  []string `json:"emails,omitempty"`
}

func (u *User) CheckPassword(password string) error {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	if err != nil {
		return errors.New("password does not match")
	}
	return nil
}

func (u *User) GeneratePasswordHash() error {
	pwd, err := generatePasswordHash(u.Password)
	if err != nil {
		return err
	}
	u.Password = pwd
	return nil
}

func (u *User) toProto() *v1.UserResponse {
	return &v1.UserResponse{
		UserId:    u.UserID,
		Name:      u.Name,
		Email:     u.Email,
		Roles:     u.Roles,
		CreatedAt: timestamppb.New(u.CreatedAt),
	}
}

func CreateUser(in *v1.CreateUserRequest) User {
	return User{
		UserID:    in.GetUserId(),
		Name:      in.GetName(),
		Email:     in.GetEmail(),
		Password:  in.GetPassword(),
		Roles:     in.GetRoles(),
		CreatedAt: time.Now().UTC(),
	}
}

func UpdateUser(in *v1.UpdateUserRequest) User {
	return User{
		UserID:   in.GetUserId(),
		Name:     in.GetName(),
		Email:    in.GetEmail(),
		Password: in.GetNewPassword(),
		Roles:    slice.Copy(in.GetRoles()),
	}
}

func generatePasswordHash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password, err: %w", err)
	}
	return string(hash), nil
}
