package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/tasker-iniutin/auth-service/internal/app"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	a := app.New(app.LoadConfig())
	if err := a.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
