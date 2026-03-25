package grpc

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	d "github.com/tasker-iniutin/auth-service/internal/domain"
	uc "github.com/tasker-iniutin/auth-service/internal/usecase"

	pb "github.com/tasker-iniutin/api-contracts/gen/go/proto/auth/v1alpha"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Server struct {
	pb.UnimplementedAuthServiceServer

	login    *uc.LoginUser
	loggout  *uc.LogoutUser
	refresh  *uc.RefreshUser
	register *uc.RegisterUser
}

func NewServer(reg *uc.RegisterUser, login *uc.LoginUser, ref *uc.RefreshUser, loggout *uc.LogoutUser) *Server {
	return &Server{
		login:    login,
		loggout:  loggout,
		refresh:  ref,
		register: reg,
	}
}

func toPBUser(u d.User) *pb.User {
	return &pb.User{
		Id:    strconv.FormatUint(uint64(u.ID), 10),
		Email: u.Email,
		Login: u.Login,
	}
}
func toPBTokenPair(p d.TokenPair) *pb.TokenPair {
	return &pb.TokenPair{
		RefreshToken: p.RefreshToken,
		AccessToken:  p.AccessToken,
	}
}

func (s *Server) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.AuthResponse, error) {
	if req.GetEmail() == "" || req.GetLogin() == "" || req.GetPassword() == "" {
		return nil, status.Error(codes.InvalidArgument, "email, login and password are required")
	}

	user, tokens, err := s.register.Exec(ctx, d.UserCreateRequest{
		Email: req.GetEmail(),
		Login: req.GetLogin(),
	}, req.GetPassword())
	if err != nil {
		return nil, mapErr(err)
	}

	return &pb.AuthResponse{
		User:   toPBUser(user),
		Tokens: toPBTokenPair(tokens),
	}, nil
}

func (s *Server) Login(ctx context.Context, req *pb.LoginRequest) (*pb.AuthResponse, error) {
	if req.GetPassword() == "" {
		return nil, status.Error(codes.InvalidArgument, "password is required")
	}
	if req.GetEmail() == "" && req.GetLogin() == "" {
		return nil, status.Error(codes.InvalidArgument, "email or login is required")
	}

	user, tokens, err := s.login.Exec(ctx, &d.UserLoginRequest{
		Email:    req.GetEmail(),
		Login:    req.GetLogin(),
		Password: req.GetPassword(),
	})
	if err != nil {
		return nil, mapErr(err)
	}

	return &pb.AuthResponse{
		User:   toPBUser(user),
		Tokens: toPBTokenPair(tokens),
	}, nil
}

func (s *Server) Refresh(ctx context.Context, req *pb.RefreshRequest) (*pb.TokenPair, error) {
	if req.GetRefreshToken() == "" {
		return nil, status.Error(codes.InvalidArgument, "refresh_token is required")
	}

	tokens, err := s.refresh.Exec(ctx, req.GetRefreshToken())
	if err != nil {
		return nil, mapErr(err)
	}

	return toPBTokenPair(tokens), nil
}

func (s *Server) Logout(ctx context.Context, req *pb.LogoutRequest) (*emptypb.Empty, error) {
	if req.GetRefreshToken() == "" {
		return nil, status.Error(codes.InvalidArgument, "refresh_token is required")
	}

	if err := s.loggout.Exec(ctx, req.GetRefreshToken()); err != nil {
		return nil, mapErr(err)
	}

	return &emptypb.Empty{}, nil
}

func mapErr(err error) error {
	print := func(c codes.Code, e error) error {
		return status.Error(c, fmt.Sprintf("authorization failed: %v", err))
	}
	switch {
	case errors.Is(err, d.ErrNotFound):
		return print(codes.NotFound, err)
	case errors.Is(err, d.ErrConflict):
		return print(codes.AlreadyExists, err)
	case errors.Is(err, d.ErrUnauthorized):
		return print(codes.Unauthenticated, err)
	case errors.Is(err, d.ErrValidation):
		return print(codes.InvalidArgument, err)
	default:
		return status.Error(codes.Internal, fmt.Sprintf("internal error: %v", err))
	}
}
