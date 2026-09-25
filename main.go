package main

import (
	"context"
	"gabagool/cli"
	"gabagool/config"
	"gabagool/connect"
	"gabagool/serve"
	"log"
	"os/signal"
	"syscall"
)

func main() {
	cli := cli.NewCli()
	cfg, err := config.LoadConfig(cli.Config)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGABRT, syscall.SIGINT)
	defer cancel()

	log.Println("Starting listeners...")
	for _, route := range cfg.Send {
		go func(route config.SendRoute) {
			if err := serve.Serve(ctx, route); err != nil {
				log.Fatal(err)
			}
		}(route)
	}

	log.Println("Connecting to routes...")
	for _, route := range cfg.Recv {
		go func(route config.RecvRoute) {
			if err := connect.Connect(ctx, route); err != nil {
				log.Fatal(err)
			}
		}(route)
	}

	<-ctx.Done()

	log.Println("Shutting down...")
}
