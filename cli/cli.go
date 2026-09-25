package cli

import (
	"flag"
	"log"
	"os"
	"path/filepath"
)

type Cli struct {
	Config string
}

func NewCli() Cli {
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

	return Cli{*config}
}
