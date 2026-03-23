package main

import (
	"log"
	"os"
	"time"

	"github.com/tasker-iniutin/auth-service/internal/app"
)

func main() {
	a := app.CreateApp(
		getenv("AUTH_GRPC_ADDR", ":50052"),
		getenv("JWT_PRIVATE_KEY_PEM", "./keys/private.pem"),
		getenv("JWT_ISSUER", "todo-auth"),
		getenv("JWT_AUDIENCE", "todo-api"),
		getDurationenv("JWT_ACCESS_TTL", 15*time.Minute),
		getenv("JWT_KEY_ID", "k1"),
		getenv("ENABLE_GRPC_REFLECTION", "true") == "true",
		getenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/auth?sslmode=disable"),
	)
	if err := a.Run(); err != nil {
		log.Fatal(err)
	}
}

func getenv(k, def string) string {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	return v
}

func getDurationenv(k string, def time.Duration) time.Duration {
	v := os.Getenv(k)
	if v == "" {
		return def
	}

	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}

	return d
}
