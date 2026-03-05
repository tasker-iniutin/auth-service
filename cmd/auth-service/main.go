package main

import (
	"log"

	"todo/auth-service/internal/app"
)

func main() {
	a := app.CreateApp(":50052")
	if err := a.Run(); err != nil {
		log.Fatal(err)
	}
}
