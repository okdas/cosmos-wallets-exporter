package config

import (
	"errors"
)

type Application struct {
	Address string `toml:"address"`
	Name    string `toml:"name"`
	Group   string `toml:"group"`
}

func (a Application) Validate() error {
	if a.Address == "" {
		return errors.New("address for application is not specified")
	}

	return nil
}
