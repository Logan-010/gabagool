package main

import (
	"context"
	"errors"
	"fmt"
	"gabagool/cli"
	"gabagool/config"
	"gabagool/connect"
	"gabagool/serve"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/BurntSushi/toml"
)

func main() {
	args := cli.NewCli()

	if !args.Daemon && args.Route.Action == "" {
		cli.PrintUsage()
		return
	}

	cfg, err := config.LoadConfig(args.Config)
	if err != nil {
		log.Fatal(err)
	}

	if args.Daemon {
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
	} else {
		switch args.Route.Action {
		case "new":
			route := config.SendRoute{Name: args.Route.New.Name, Auth: config.NewRouteAuth(), Addr: args.Route.New.Addr, Tcp: args.Route.New.Tcp, Udp: args.Route.New.Udp}

			for _, r := range cfg.Send {
				if r.Name == route.Name || r.Addr == route.Addr {
					log.Fatalf("Route \"%s\" already exists for \"%s\"", r.Name, r.Addr)
				}
			}

			secret, err := route.Auth.TailcatAddr()
			if err != nil {
				log.Fatal(err)
			}

			cfg.Send = append(cfg.Send, route)
			if err := cfg.Save(args.Config); err != nil {
				log.Fatal(err)
			}

			ext := ""
			if route.Tcp {
				ext = fmt.Sprintf("%s -tcp", ext)
			}
			if route.Udp {
				ext = fmt.Sprintf("%s -udp", ext)
			}

			fmt.Printf("Route \"%s\" created! To add, run:\n\tgabagool route add -name %s -addr %s -server %s%s\n", route.Name, route.Name, route.Addr, secret, ext)
		case "add":
			route := config.RecvRoute{Name: args.Route.Add.Name, Server: args.Route.Add.Server, Addr: args.Route.Add.Addr, Tcp: args.Route.Add.Tcp, Udp: args.Route.Add.Udp}

			for _, r := range cfg.Recv {
				if r.Name == route.Name || r.Addr == route.Addr {
					log.Fatalf("Route \"%s\" already exists for \"%s\"", r.Name, r.Addr)
				}
			}

			cfg.Recv = append(cfg.Recv, route)
			if err := cfg.Save(args.Config); err != nil {
				log.Fatal(err)
			}

			fmt.Printf("Added route \"%s\"", route.Name)
		case "export":
			route := config.SendRoute{Name: args.Route.Export.Name, Auth: config.NewRouteAuth(), Addr: args.Route.Export.Addr, Tcp: args.Route.Export.Tcp, Udp: args.Route.Export.Udp}

			for _, r := range cfg.Send {
				if r.Name == route.Name || r.Addr == route.Addr {
					log.Fatalf("Route \"%s\" already exists for \"%s\"", r.Name, r.Addr)
				}
			}

			secret, err := route.Auth.TailcatAddr()
			if err != nil {
				log.Fatal(err)
			}

			encoded, err := toml.Marshal(config.RecvRoute{Name: route.Name, Server: secret, Addr: route.Addr, Tcp: route.Tcp, Udp: route.Udp})
			if err != nil {
				log.Fatal(err)
			}

			parent := filepath.Dir(args.Route.Export.Path)
			_, err = os.Stat(parent)
			doesNotExist := errors.Is(err, os.ErrNotExist)
			if !doesNotExist && err != nil {
				log.Fatal(err)
			}

			if doesNotExist {
				if err := os.MkdirAll(parent, 0755); err != nil {
					log.Fatal(err)
				}
			}

			file, err := os.Create(args.Route.Export.Path)
			if err != nil {
				log.Fatal(err)
			}
			defer file.Close()

			written := 0
			for written != len(encoded) {
				n, err := file.Write(encoded[written:])
				if err != nil {
					log.Fatal(err)
				}

				written += n
			}

			cfg.Send = append(cfg.Send, route)
			if err := cfg.Save(args.Config); err != nil {
				log.Fatal(err)
			}

			fmt.Printf("Wrote route \"%s\" to \"%s\"", route.Name, args.Route.Export.Path)
		case "import":
			file, err := os.Open(args.Route.Import.Path)
			if err != nil {
				log.Fatal(err)
			}
			defer file.Close()

			content, err := io.ReadAll(file)
			if err != nil {
				log.Fatal(err)
			}

			var route config.RecvRoute
			if err := toml.Unmarshal(content, &route); err != nil {
				log.Fatal(err)
			}

			overrides := args.Route.Import
			if overrides.Overrides["name"] {
				route.Name = overrides.Name
			}
			if overrides.Overrides["addr"] {
				route.Addr = overrides.Addr
			}
			if overrides.Overrides["server"] {
				route.Server = overrides.Server
			}
			if overrides.Overrides["tcp"] {
				route.Tcp = overrides.Tcp
			}
			if overrides.Overrides["udp"] {
				route.Udp = overrides.Udp
			}

			for _, r := range cfg.Recv {
				if r.Name == route.Name || r.Addr == route.Addr {
					log.Fatalf("Route \"%s\" already exists for \"%s\"", r.Name, r.Addr)
				}
			}

			cfg.Recv = append(cfg.Recv, route)
			if err := cfg.Save(args.Config); err != nil {
				log.Fatal(err)
			}

			fmt.Printf("Added route \"%s\"", route.Name)
		default:
			panic("Invalid route action")
		}
	}
}
