package main

import (
	"context"

	"github.com/infanasotku/farang-edge/internal/app"
)

func main() {
	ctx := context.Background()
	app, err := app.New()
	if err != nil {
		panic(err)
	}

	err = app.Run(ctx)
	if err != nil {
		panic(err)
	}
}
