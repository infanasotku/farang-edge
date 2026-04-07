package main

import (
	"context"

	"github.com/infanasotku/farang-edge/internal/app"
)

func main() {
	ctx := context.Background()
	_, err := app.New(ctx)
	if err != nil {
		panic(err)
	}

	// Use the app instance 'a' to start your application logic
}
