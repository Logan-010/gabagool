package main

import (
	"context"
	"gabagool/config"
	"log"

	"github.com/tailscale/tailcat"
)

func serve(ctx context.Context, cfg config.Config) error {
	psk, err := cfg.Psk()
	if err != nil {
		log.Fatal(err)
	}

	node, err := cfg.Identity()
	if err != nil {
		log.Fatal(err)
	}

	server := tailcat.Server{Key: node, PresharedKey: psk}
	defer server.Close()

	socket, err := server.Listen(ctx, "tcp", cfg.Addr)
	if err != nil {
		log.Fatal(err)
	}
	defer socket.Close()

	log.Printf("Listening on \"%s\"...", socket.Addr())

	return nil
}
