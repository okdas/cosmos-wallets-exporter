package config

import (
	"errors"
	"fmt"
)

type Chain struct {
	Name                    string        `toml:"name"`
	LCDEndpoint             string        `toml:"lcd-endpoint"`
	Denoms                  []DenomInfo   `toml:"denoms"`
	Wallets                 []Wallet      `toml:"wallets"`
	Applications            []Application `toml:"applications"`
	Suppliers               []Supplier    `toml:"suppliers"`
	RevShareDetailedMetrics *bool         `toml:"rev-share-detailed-metrics"`
}

func (c *Chain) Validate() error {
	if c.Name == "" {
		return errors.New("empty chain name")
	}

	if c.LCDEndpoint == "" {
		return errors.New("no LCD endpoint provided")
	}

	if len(c.Wallets) == 0 && len(c.Applications) == 0 && len(c.Suppliers) == 0 {
		return errors.New("no wallets, applications, or suppliers provided")
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

	for index, supplier := range c.Suppliers {
		if err := supplier.Validate(); err != nil {
			return fmt.Errorf("error in supplier %d: %s", index, err)
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

func (c *Chain) IsRevShareDetailedMetricsEnabled() bool {
	if c.RevShareDetailedMetrics == nil {
		return true // Default to detailed metrics for backward compatibility
	}
	return *c.RevShareDetailedMetrics
}
