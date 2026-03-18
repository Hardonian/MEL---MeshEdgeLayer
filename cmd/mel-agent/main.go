package main

import (
	"context"
	"flag"
	"os/signal"
	"syscall"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/service"
)

func main() {
	cfgPath := flag.String("config", "configs/mel.example.json", "config path")
	flag.Parse()
	cfg, _, err := config.Load(*cfgPath)
	if err != nil {
		panic(err)
	}
	app, err := service.New(cfg)
	if err != nil {
		panic(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := app.Start(ctx); err != nil {
		panic(err)
	}
}
