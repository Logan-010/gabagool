package cli

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

type Cli struct {
	Config string
	Daemon bool
	Route  RouteCommand
}

type RouteCommand struct {
	Action string
	New    NewRouteArgs
	Add    AddRouteArgs
	Export ExportRouteArgs
	Import ImportRouteArgs
}

type NewRouteArgs struct {
	Addr string
	Udp  bool
	Tcp  bool
	Name string
}

type AddRouteArgs struct {
	Server string
	Addr   string
	Tcp    bool
	Udp    bool
	Name   string
}

type ExportRouteArgs struct {
	Addr string
	Udp  bool
	Tcp  bool
	Name string
	Path string
}

type ImportRouteArgs struct {
	Overrides map[string]bool
	Path      string
	Server    string
	Addr      string
	Tcp       bool
	Udp       bool
	Name      string
}

func NewCli() Cli {
	flag.Usage = PrintUsage
	config := flag.String("config", "", "Sets path to config")
	if *config == "" {
		*config = os.Getenv("GABAGOOL_CONFIG")
	}
	if *config == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatal(err)
		}

		*config = filepath.Join(home, ".gabagool", "config.toml")
	}

	flag.Parse()

	// Handle the help command after parsing global flags.
	if len(flag.Args()) > 0 {
		if flag.Args()[0] == "help" || flag.Args()[0] == "-h" || flag.Args()[0] == "--help" {
			PrintUsage()
			os.Exit(0)
		}
	}

	// Handle daemon command
	if len(flag.Args()) > 0 && flag.Args()[0] == "daemon" {
		return Cli{*config, true, RouteCommand{}}
	}

	// Handle route subcommand
	if len(flag.Args()) > 0 && flag.Args()[0] == "route" {
		// For the route subcommand, we need a different approach
		if len(flag.Args()) > 1 {
			// Get the action (new, add, export, import)
			action := flag.Args()[1]

			switch action {
			case "new":
				// Create a new flag set for new command
				newCmd := flag.NewFlagSet("route new", flag.ContinueOnError)
				newAddr := newCmd.String("addr", "", "Address to listen on")
				newCmd.StringVar(newAddr, "a", "", "Address to listen on")
				newUdp := newCmd.Bool("udp", false, "Enable UDP")
				newCmd.BoolVar(newUdp, "u", false, "Enable UDP")
				newTcp := newCmd.Bool("tcp", false, "Enable TCP")
				newCmd.BoolVar(newTcp, "t", false, "Enable TCP")
				newName := newCmd.String("name", "", "Name for the route")

				// Parse args for new command
				err := newCmd.Parse(flag.Args()[2:])
				if err != nil {
					log.Fatal("Error parsing route new arguments:", err)
				}

				// Validate required arguments
				if *newAddr == "" && !*newUdp && !*newTcp {
					log.Fatal("new command requires at least one of: addr, udp, or tcp")
				}

				return Cli{
					Config: *config,
					Daemon: false,
					Route: RouteCommand{
						Action: "new",
						New: NewRouteArgs{
							Addr: *newAddr,
							Udp:  *newUdp,
							Tcp:  *newTcp,
							Name: *newName,
						},
					},
				}

			case "add":
				// Create a new flag set for add command
				addCmd := flag.NewFlagSet("route add", flag.ContinueOnError)
				addServer := addCmd.String("server", "", "Tailcat server address")
				addCmd.StringVar(addServer, "s", "", "Tailcat server address")
				addAddr := addCmd.String("addr", "", "Address to connect to")
				addTcp := addCmd.Bool("tcp", false, "Enable TCP for connection")
				addUdp := addCmd.Bool("udp", false, "Enable UDP for connection")
				addName := addCmd.String("name", "", "Name for the route")

				// Parse args for add command
				err := addCmd.Parse(flag.Args()[2:])
				if err != nil {
					log.Fatal("Error parsing route add arguments:", err)
				}

				// Validate required arguments
				if *addAddr == "" {
					log.Fatal("add command requires addr")
				}

				return Cli{
					Config: *config,
					Daemon: false,
					Route: RouteCommand{
						Action: "add",
						Add: AddRouteArgs{
							Server: *addServer,
							Addr:   *addAddr,
							Tcp:    *addTcp,
							Udp:    *addUdp,
							Name:   *addName,
						},
					},
				}

			case "export":
				// Create a new flag set for export command
				exportCmd := flag.NewFlagSet("route export", flag.ContinueOnError)
				exportAddr := exportCmd.String("addr", "", "Address to export (for new routes)")
				exportUdp := exportCmd.Bool("udp", false, "Enable UDP for export")
				exportTcp := exportCmd.Bool("tcp", false, "Enable TCP for export")
				exportName := exportCmd.String("name", "", "Name for the exported route")
				exportPath := exportCmd.String("path", "", "Export file path")

				// Parse args for export command
				err := exportCmd.Parse(flag.Args()[2:])
				if err != nil {
					log.Fatal("Error parsing route export arguments:", err)
				}

				// Validate required arguments
				if *exportPath == "" {
					log.Fatal("export command requires path")
				}

				return Cli{
					Config: *config,
					Daemon: false,
					Route: RouteCommand{
						Action: "export",
						Export: ExportRouteArgs{
							Addr: *exportAddr,
							Udp:  *exportUdp,
							Tcp:  *exportTcp,
							Name: *exportName,
							Path: *exportPath,
						},
					},
				}

			case "import":
				// Create a new flag set for import command
				importCmd := flag.NewFlagSet("route import", flag.ContinueOnError)
				importPath := importCmd.String("path", "", "Import file path")
				importServer := importCmd.String("server", "", "Tailcat server address")
				importCmd.StringVar(importServer, "s", "", "Tailcat server address")
				importAddr := importCmd.String("addr", "", "Address to connect to (for imported routes)")
				importTcp := importCmd.Bool("tcp", false, "Enable TCP for imported connection")
				importUdp := importCmd.Bool("udp", false, "Enable UDP for imported connection")
				importName := importCmd.String("name", "", "Name for the imported route")

				// Parse args for import command
				err := importCmd.Parse(flag.Args()[2:])
				if err != nil {
					log.Fatal("Error parsing route import arguments:", err)
				}

				// Validate required arguments
				if *importPath == "" {
					log.Fatal("import command requires path")
				}

				overrides := make(map[string]bool)
				importCmd.Visit(func(f *flag.Flag) {
					name := f.Name
					if name == "s" {
						name = "server"
					}
					overrides[name] = true
				})

				return Cli{
					Config: *config,
					Daemon: false,
					Route: RouteCommand{
						Action: "import",
						Import: ImportRouteArgs{
							Overrides: overrides,
							Path:      *importPath,
							Server:    *importServer,
							Addr:      *importAddr,
							Tcp:       *importTcp,
							Udp:       *importUdp,
							Name:      *importName,
						},
					},
				}

			default:
				log.Fatal("Invalid route action:", action)
			}
		} else {
			log.Fatal("Route command requires an action (new, add, export, import)")
		}
	}

	return Cli{*config, false, RouteCommand{}}
}

func PrintUsage() {
	fmt.Println(`gabagool - Network tunneling utility using Tailscale

Usage:
  gabagool [flags] [command]

Commands:
  route     Manage routes (new, add, export, import)
  daemon    Run as daemon

Flags:
  -config string
        Sets path to config

Available commands:
  gabagool help              Show this help message

Route subcommands:
  gabagool route new         Create a new route (listen on address)
  gabagool route add         Add an existing route to connect to
  gabagool route export      Export route configuration
  gabagool route import      Import route configuration

Example usage:
  gabagool daemon                  Run as a daemon
  gabagool route new -addr :8080 -tcp
  gabagool route add -addr example.com:8080 -tcp
  gabagool route export -name my-route -path /tmp/my-route.toml

Route 'new' command flags:
  -a, -addr string    Address to listen on (required for new)
  -u, -udp           Enable UDP (required for new)
  -t, -tcp           Enable TCP (required for new)
  -name string       Name for the route

Route 'add' command flags:
  -s, -server string  Tailcat server address
  -addr string       Address to connect to (required)
  -tcp               Enable TCP for connection
  -udp               Enable UDP for connection
  -name string       Name for the route

Route 'export' command flags:
  -addr string       Address to export (for new routes)
  -udp               Enable UDP for export
  -tcp               Enable TCP for export
  -name string       Name for the exported route
  -path string       Export file path (required)

Route 'import' command flags:
  -path string       Import file path (required)
  -s, -server string  Tailcat server address
  -addr string       Address to connect to (for imported routes)
  -tcp               Enable TCP for imported connection
  -udp               Enable UDP for imported connection
  -name string       Name for the imported route`)
}
