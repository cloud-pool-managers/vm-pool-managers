package auth

import (
	"context"
	"control_center/frontcontrolpb"
	"control_center/models"
	"fmt"

	"gorm.io/gorm"
)

type Service struct {
	frontcontrolpb.UnimplementedAuthServiceServer
	DB *gorm.DB
}

func New() *Service {
	return &Service{}
}

func (s *Service) CreateUser(ctx context.Context, req *frontcontrolpb.CreateUserRequest) (*frontcontrolpb.CreateUserResponse, error) {
	if req.Username == "" || req.Email == "" || req.Password == "" {
		return &frontcontrolpb.CreateUserResponse{
			Success: false,
			UserId:  "",
		}, fmt.Errorf("Missing required fields")
	}
	u := models.User{
		Name:     req.Username,
		Email:    req.Email,
		Password: req.Password,
	}
	if err := s.DB.Create(&u).Error; err != nil {
		return &frontcontrolpb.CreateUserResponse{
			Success: false,
			UserId:  "",
		}, fmt.Errorf("Failed to create user: %v", err)
	}
	return &frontcontrolpb.CreateUserResponse{
		Success: true,
		UserId:  fmt.Sprintf("%d", u.ID),
	}, nil
}

func (s *Service) AuthenticateUser(ctx context.Context, req *frontcontrolpb.AuthenticateUserRequest) (*frontcontrolpb.AuthenticateUserResponse, error) {
	var user models.User
	if err := s.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return &frontcontrolpb.AuthenticateUserResponse{
				Success: false,
				Token:   "",
			}, fmt.Errorf("User not found")
		}
		return &frontcontrolpb.AuthenticateUserResponse{
			Success: false,
			Token:   "",
		}, fmt.Errorf("Database error: %v", err)
	}

	if user.Password != req.Password {
		return &frontcontrolpb.AuthenticateUserResponse{
			Success: false,
			Token:   "",
		}, fmt.Errorf("Invalid password")
	}

	// Here you would normally generate a JWT or session token
	token := "dummy-token-for-user-" + fmt.Sprintf("%d", user.ID)

	return &frontcontrolpb.AuthenticateUserResponse{
		Success: true,
		Token:   token,
	}, nil
}
