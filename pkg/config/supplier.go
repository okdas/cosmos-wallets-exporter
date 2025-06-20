package config

import (
	"errors"
)

type Supplier struct {
	Address string `toml:"address"`
	Name    string `toml:"name"`
	Group   string `toml:"group"`
}

func (s Supplier) Validate() error {
	if s.Address == "" {
		return errors.New("address for supplier is not specified")
	}

	return nil
}
