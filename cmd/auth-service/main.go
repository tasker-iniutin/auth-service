package main

import (
	"log"

	"github.com/tasker-iniutin/auth-service/internal/app"
)

func main() {
	a := app.New(app.LoadConfig())
	if err := a.Run(); err != nil {
		log.Fatal(err)
	}
}
