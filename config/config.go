package config

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"tailscale.com/types/key"
)

type RouteAuth struct {
	Node string `toml:"node"`
	Key  string `toml:"key"`
}

func NewRouteAuth() RouteAuth {
	encoded, err := key.NewNode().MarshalText()
	if err != nil {
		log.Fatal(err)
	}

	var n [32]byte
	if _, err := io.ReadFull(rand.Reader, n[:]); err != nil {
		log.Fatal(err)
	}

	return RouteAuth{string(encoded), base64.StdEncoding.EncodeToString(n[:])}
}

func CreateRouteAuth(psk [32]byte) RouteAuth {
	encoded, err := key.NewNode().MarshalText()
	if err != nil {
		log.Fatal(err)
	}

	return RouteAuth{string(encoded), base64.StdEncoding.EncodeToString(psk[:])}
}

func (c *RouteAuth) Identity() (key.NodePrivate, error) {
	var out key.NodePrivate

	return out, out.UnmarshalText([]byte(c.Node))
}

func (c *RouteAuth) Psk() ([32]byte, error) {
	out, err := base64.StdEncoding.DecodeString(c.Key)
	if err != nil {
		return [32]byte{}, err
	}

	if len(out) != 32 {
		return [32]byte{}, errors.New("invalid PSK")
	}

	return [32]byte(out), nil
}

type SendRoute struct {
	Name string    `toml:"name"`
	Auth RouteAuth `toml:"auth"`
	Addr string    `toml:"addr"`
	Tcp  bool      `toml:"tcp"`
	Udp  bool      `toml:"udp"`
}

type RecvRoute struct {
	Name   string `toml:"name"`
	Server string `toml:"server"`
	Addr   string `toml:"addr"`
	Tcp    bool   `toml:"tcp"`
	Udp    bool   `toml:"udp"`
}

type Config struct {
	Send []SendRoute `toml:"send"`
	Recv []RecvRoute `toml:"recv"`
}

func LoadConfig(from string) (Config, error) {
	_, err := os.Stat(from)
	doesNotExist := errors.Is(err, os.ErrNotExist)
	if err != nil && !doesNotExist {
		return Config{}, err
	}

	if doesNotExist {
		out := Config{}

		if err := out.Save(from); err != nil {
			return out, err
		}

		return out, nil
	} else {
		file, err := os.Open(from)
		if err != nil {
			return Config{}, err
		}
		defer file.Close()

		content, err := io.ReadAll(file)
		if err != nil {
			return Config{}, err
		}

		var out Config
		if err := toml.Unmarshal(content, &out); err != nil {
			return Config{}, err
		}

		return out, nil
	}
}

func (c *Config) Save(to string) error {
	encoded, err := toml.Marshal(*c)
	if err != nil {
		return err
	}

	parent := filepath.Dir(to)
	_, err = os.Stat(parent)
	doesNotExist := errors.Is(err, os.ErrNotExist)
	if err != nil && !doesNotExist {
		return err
	}

	if doesNotExist {
		if err := os.MkdirAll(parent, 0755); err != nil {
			return err
		}
	}

	file, err := os.Create(to)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := file.Write(encoded); err != nil {
		return err
	}

	return nil
}
