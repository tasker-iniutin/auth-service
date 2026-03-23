package app

import (
	"time"

	"github.com/tasker-iniutin/common/configenv"
)

type Config struct {
	GRPCAddr         string
	PrivateKeyPath   string
	JWTIssuer        string
	JWTAudience      string
	JWTAccessTTL     time.Duration
	JWTKeyID         string
	EnableReflection bool
	DatabaseURL      string
	RedisAddr        string
	RedisPassword    string
}

func LoadConfig() Config {
	return Config{
		GRPCAddr:         configenv.String("AUTH_GRPC_ADDR", ":50052"),
		PrivateKeyPath:   configenv.String("JWT_PRIVATE_KEY_PEM", ""),
		JWTIssuer:        configenv.String("JWT_ISSUER", "todo-auth"),
		JWTAudience:      configenv.String("JWT_AUDIENCE", "todo-api"),
		JWTAccessTTL:     configenv.Duration("JWT_ACCESS_TTL", 15*time.Minute),
		JWTKeyID:         configenv.String("JWT_KEY_ID", "k1"),
		EnableReflection: configenv.Bool("ENABLE_GRPC_REFLECTION", true),
		DatabaseURL:      configenv.String("DATABASE_URL", "postgres://postgres:postgres@localhost:5433/auth?sslmode=disable"),
		RedisAddr:        configenv.String("REDIS_ADDR", "127.0.0.1:6379"),
		RedisPassword:    configenv.String("REDIS_PASSWORD", ""),
	}
}
