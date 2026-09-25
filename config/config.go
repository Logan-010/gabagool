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

type configInner struct {
	Node string `toml:"node"`
	Key  string `toml:"key"`
	Addr string `toml:"addr"`
}

type Config struct {
	node string
	key  string
	Addr string
}

func NewConfig() Config {
	encoded, err := key.NewNode().MarshalText()
	if err != nil {
		log.Fatal(err)
	}

	var n [32]byte
	if _, err := io.ReadFull(rand.Reader, n[:]); err != nil {
		log.Fatal(err)
	}

	return Config{string(encoded), base64.StdEncoding.EncodeToString(n[:]), "[::]:0"}
}

func LoadConfig(from string) (Config, error) {
	_, err := os.Stat(from)
	doesNotExist := errors.Is(err, os.ErrNotExist)
	if err != nil && !doesNotExist {
		return NewConfig(), err
	}

	if doesNotExist {
		out := NewConfig()

		if err := out.Save(from); err != nil {
			return out, err
		}

		return out, nil
	} else {
		file, err := os.Open(from)
		if err != nil {
			return NewConfig(), err
		}
		defer file.Close()

		content, err := io.ReadAll(file)
		if err != nil {
			return NewConfig(), err
		}

		var out configInner
		err = toml.Unmarshal(content, &out)
		if err != nil {
			return NewConfig(), err
		}

		return Config{out.Node, out.Key, out.Addr}, nil
	}
}

func (c *Config) Save(to string) error {
	inner := configInner{c.node, c.key, c.Addr}
	encoded, err := toml.Marshal(inner)
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

func (c *Config) Identity() (key.NodePrivate, error) {
	var out key.NodePrivate

	return out, out.UnmarshalText([]byte(c.node))
}

func (c *Config) Psk() ([32]byte, error) {
	out, err := base64.StdEncoding.DecodeString(c.key)
	if err != nil {
		return [32]byte{}, err
	}

	if len(out) != 32 {
		return [32]byte{}, errors.New("invalid PSK")
	}

	return [32]byte(out), nil
}
