package main

import (
	"context"
	"gabagool/cli"
	"gabagool/config"
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

	go func() {
		if err := serve(ctx, cfg); err != nil {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()
}
