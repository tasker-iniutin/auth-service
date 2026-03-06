package app

import (
	"crypto/rand"
	"crypto/rsa"
	"log"
	"net"
	"os"
	"time"

	g "google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/redis/go-redis/v9"
	authpb "github.com/tasker-iniutin/api-contracts/gen/go/proto/auth/v1alpha"
	sec "github.com/tasker-iniutin/common/authsecurity"

	mem "github.com/tasker-iniutin/auth-service/internal/store/mem"
	redrepo "github.com/tasker-iniutin/auth-service/internal/store/redis"
	grpc "github.com/tasker-iniutin/auth-service/internal/transport/grpc"
	"github.com/tasker-iniutin/auth-service/internal/usecase"
)

type App struct {
	grpcAddr string
}

func CreateApp(grpcAddr string) *App {
	return &App{grpcAddr: grpcAddr}
}

func (a *App) Run() error {
	// ----- infra: repos -----
	userRepo := mem.NewUserRepo()

	redisAddr := getenv("REDIS_ADDR", "127.0.0.1:6379")
	redisPass := os.Getenv("REDIS_PASSWORD")
	redisDB := 0

	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPass,
		DB:       redisDB,
	})

	sessionRepo := redrepo.NewRedisRepo(rdb)

	// ----- security: keys/tokens -----
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	issuerName := "todo-auth"
	audience := "todo-api"
	accessTTL := 15 * time.Minute
	keyID := "k1"

	issuer := sec.NewRS256Issuer(privateKey, issuerName, audience, accessTTL, keyID)
	verifier := sec.NewRS256Verifier(&privateKey.PublicKey, issuerName, audience) // если есть

	// ----- usecases -----
	regUser := usecase.NewRegisterUser(sessionRepo, userRepo, issuer)
	logUser := usecase.NewLoginUser(sessionRepo, userRepo, issuer)
	refreshUC := usecase.NewRefreshUser(sessionRepo, issuer)
	logoutUC := usecase.NewLogoutUser(sessionRepo, verifier)

	// ----- handler -----
	srv := grpc.NewServer(regUser, logUser, refreshUC, logoutUC)

	// ----- gRPC server -----
	grpcServer := g.NewServer()
	authpb.RegisterAuthServiceServer(grpcServer, srv)
	reflection.Register(grpcServer) // dev-only

	lis, err := net.Listen("tcp", a.grpcAddr)
	if err != nil {
		return err
	}

	log.Printf("auth-service gRPC listening on %s", a.grpcAddr)
	if err := grpcServer.Serve(lis); err != nil {
		log.Printf("gRPC server stopped: %v", err)
		return err
	}
	return nil
}

func getenv(k, def string) string {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	return v
}
