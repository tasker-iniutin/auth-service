package app

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
	authpb "github.com/tasker-iniutin/api-contracts/gen/go/proto/auth/v1alpha"
	sec "github.com/tasker-iniutin/common/authsecurity"
	"github.com/tasker-iniutin/common/postgres"
	"github.com/tasker-iniutin/common/runtime"
	"google.golang.org/grpc"

	"github.com/tasker-iniutin/auth-service/internal/store/postgre"
	redrepo "github.com/tasker-iniutin/auth-service/internal/store/redis"
	handlergrpc "github.com/tasker-iniutin/auth-service/internal/transport/grpc"
	"github.com/tasker-iniutin/auth-service/internal/usecase"
)

type App struct {
	cfg Config
}

func New(cfg Config) *App {
	return &App{cfg: cfg}
}

func (a *App) Run() error {
	db, err := postgres.Open(context.Background(), a.cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	userRepo := postgre.NewPostgreRepo(db)

	rdb := redis.NewClient(&redis.Options{
		Addr:     a.cfg.RedisAddr,
		Password: a.cfg.RedisPassword,
		DB:       0,
	})

	sessionRepo := redrepo.NewRedisRepo(rdb)

	privateKey, err := sec.LoadRSAPrivateKeyFromPEMFile(a.cfg.PrivateKeyPath)
	if err != nil {
		return err
	}

	issuer := sec.NewRS256Issuer(privateKey, a.cfg.JWTIssuer, a.cfg.JWTAudience, a.cfg.JWTAccessTTL, a.cfg.JWTKeyID)

	regUser := usecase.NewRegisterUser(sessionRepo, userRepo, issuer)
	logUser := usecase.NewLoginUser(sessionRepo, userRepo, issuer)
	refreshUC := usecase.NewRefreshUser(sessionRepo, issuer)
	logoutUC := usecase.NewLogoutUser(sessionRepo)

	handler := handlergrpc.NewServer(regUser, logUser, refreshUC, logoutUC)

	log.Printf("auth-service gRPC listening on %s", a.cfg.GRPCAddr)
	log.Printf(
		"auth-service jwt config: private_key=%s issuer=%s audience=%s key_id=%s access_ttl=%s",
		a.cfg.PrivateKeyPath,
		a.cfg.JWTIssuer,
		a.cfg.JWTAudience,
		a.cfg.JWTKeyID,
		a.cfg.JWTAccessTTL,
	)

	return runtime.ServeGRPC(
		a.cfg.GRPCAddr,
		func(server *grpc.Server) {
			authpb.RegisterAuthServiceServer(server, handler)
		},
		a.cfg.EnableReflection,
	)
}
