package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/infanasotku/farang-edge/internal/app"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app, err := app.New()
	if err != nil {
		panic(err)
	}

	err = app.Run(ctx)
	if err != nil {
		panic(err)
	}
}
