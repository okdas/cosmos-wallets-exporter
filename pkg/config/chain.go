package config

import (
	"errors"
	"fmt"
)

type Chain struct {
	Name         string        `toml:"name"`
	LCDEndpoint  string        `toml:"lcd-endpoint"`
	Denoms       []DenomInfo   `toml:"denoms"`
	Wallets      []Wallet      `toml:"wallets"`
	Applications []Application `toml:"applications"`
}

func (c *Chain) Validate() error {
	if c.Name == "" {
		return errors.New("empty chain name")
	}

	if c.LCDEndpoint == "" {
		return errors.New("no LCD endpoint provided")
	}

	if len(c.Wallets) == 0 && len(c.Applications) == 0 {
		return errors.New("no wallets or applications provided")
	}

	for index, wallet := range c.Wallets {
		if err := wallet.Validate(); err != nil {
			return fmt.Errorf("error in wallet %d: %s", index, err)
		}
	}

	for index, application := range c.Applications {
		if err := application.Validate(); err != nil {
			return fmt.Errorf("error in application %d: %s", index, err)
		}
	}

	return nil
}

func (c *Chain) FindDenomByName(denom string) (*DenomInfo, bool) {
	for _, denomIterated := range c.Denoms {
		if denomIterated.Denom == denom {
			return &denomIterated, true
		}
	}

	return nil, false
}
