package app

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	g "google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	authpb "github.com/tasker-iniutin/api-contracts/gen/go/proto/auth/v1alpha"
	sec "github.com/tasker-iniutin/common/authsecurity"

	"github.com/tasker-iniutin/auth-service/internal/store/postgre"
	redrepo "github.com/tasker-iniutin/auth-service/internal/store/redis"
	grpc "github.com/tasker-iniutin/auth-service/internal/transport/grpc"
	"github.com/tasker-iniutin/auth-service/internal/usecase"
)

type App struct {
	grpcAddr         string
	privateKeyPath   string
	jwtIssuer        string
	jwtAudience      string
	jwtAccessTTL     time.Duration
	jwtKeyID         string
	enableReflection bool
	databaseAddr     string
}

func CreateApp(
	grpcAddr string,
	privateKeyPath string,
	jwtIssuer string,
	jwtAudience string,
	jwtAccessTTL time.Duration,
	jwtKeyID string,
	enableReflection bool,
	databaseAddr string,
) *App {
	return &App{
		grpcAddr:         grpcAddr,
		privateKeyPath:   privateKeyPath,
		jwtIssuer:        jwtIssuer,
		jwtAudience:      jwtAudience,
		jwtAccessTTL:     jwtAccessTTL,
		jwtKeyID:         jwtKeyID,
		enableReflection: enableReflection,
		databaseAddr:     databaseAddr,
	}
}

func (a *App) Run() error {
	// userRepo := mem.NewUserRepo() in mem
	db, err := pgxpool.New(context.Background(), a.databaseAddr)
	if err != nil {
		return fmt.Errorf("create db pool: %w", err)
	}
	if err := db.Ping(context.Background()); err != nil {
		db.Close()
		return fmt.Errorf("ping  db: %w", err)
	}
	defer db.Close()

	userRepo := postgre.NewPostgreRepo(db)

	redisAddr := getenv("REDIS_ADDR", "127.0.0.1:6379")
	redisPass := os.Getenv("REDIS_PASSWORD")

	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPass,
		DB:       0,
	})

	sessionRepo := redrepo.NewRedisRepo(rdb)

	privateKey, err := loadRSAPrivateKey(a.privateKeyPath)
	if err != nil {
		return fmt.Errorf("load private key: %w", err)
	}

	issuer := sec.NewRS256Issuer(privateKey, a.jwtIssuer, a.jwtAudience, a.jwtAccessTTL, a.jwtKeyID)
	verifier := sec.NewRS256Verifier(&privateKey.PublicKey, a.jwtIssuer, a.jwtAudience)

	regUser := usecase.NewRegisterUser(sessionRepo, userRepo, issuer)
	logUser := usecase.NewLoginUser(sessionRepo, userRepo, issuer)
	refreshUC := usecase.NewRefreshUser(sessionRepo, issuer)
	logoutUC := usecase.NewLogoutUser(sessionRepo, verifier)

	srv := grpc.NewServer(regUser, logUser, refreshUC, logoutUC)

	grpcServer := g.NewServer()
	authpb.RegisterAuthServiceServer(grpcServer, srv)
	if a.enableReflection {
		reflection.Register(grpcServer)
	}

	lis, err := net.Listen("tcp", a.grpcAddr)
	if err != nil {
		return err
	}

	log.Printf("auth-service gRPC listening on %s", a.grpcAddr)
	log.Printf(
		"auth-service jwt config: private_key=%s issuer=%s audience=%s key_id=%s access_ttl=%s",
		a.privateKeyPath,
		a.jwtIssuer,
		a.jwtAudience,
		a.jwtKeyID,
		a.jwtAccessTTL,
	)
	return grpcServer.Serve(lis)
}

func loadRSAPrivateKey(path string) (*rsa.PrivateKey, error) {
	keyBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(keyBytes)
	if block == nil {
		return nil, errors.New("invalid PEM: no block found")
	}

	// PKCS#1: -----BEGIN RSA PRIVATE KEY-----
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}

	// PKCS#8: -----BEGIN PRIVATE KEY-----
	keyAny, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	key, ok := keyAny.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("PEM does not contain RSA private key")
	}

	return key, nil
}

func getenv(k, def string) string {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	return v
}
