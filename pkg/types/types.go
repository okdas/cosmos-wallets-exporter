package types

import (
	"context"
	"main/pkg/config"
	"time"

	"cosmossdk.io/math"
	"github.com/prometheus/client_golang/prometheus"
)

type Balance struct {
	Denom  string         `json:"denom"`
	Amount math.LegacyDec `json:"amount"`
}

type Balances []Balance

type BalanceResponse struct {
	Balances Balances `json:"balances"`
}

type WalletBalanceEntry struct {
	Chain    string
	Success  bool
	Duration time.Duration
	Wallet   config.Wallet
	Balances Balances
}

type ApplicationStake struct {
	Amount string `json:"amount"`
	Denom  string `json:"denom"`
}

type ApplicationData struct {
	Address                   string                   `json:"address"`
	DelegateeGatewayAddresses []string                 `json:"delegatee_gateway_addresses"`
	PendingTransfer           interface{}              `json:"pending_transfer"`
	PendingUndelegations      map[string]interface{}   `json:"pending_undelegations"`
	ServiceConfigs            []map[string]interface{} `json:"service_configs"`
	Stake                     ApplicationStake         `json:"stake"`
	UnstakeSessionEndHeight   string                   `json:"unstake_session_end_height"`
}

type ApplicationResponse struct {
	Application ApplicationData `json:"application"`
}

type ApplicationStakeEntry struct {
	Chain       string
	Success     bool
	Duration    time.Duration
	Application config.Application
	Stake       ApplicationStake
}

type SupplierStake struct {
	Amount string `json:"amount"`
	Denom  string `json:"denom"`
}

type SupplierData struct {
	OperatorAddress         string                   `json:"operator_address"`
	OwnerAddress            string                   `json:"owner_address"`
	ServiceConfigHistory    []map[string]interface{} `json:"service_config_history"`
	Services                []map[string]interface{} `json:"services"`
	Stake                   SupplierStake            `json:"stake"`
	UnstakeSessionEndHeight string                   `json:"unstake_session_end_height"`
}

type SupplierResponse struct {
	Supplier SupplierData `json:"supplier"`
}

type SupplierStakeEntry struct {
	Chain    string
	Success  bool
	Duration time.Duration
	Supplier config.Supplier
	Stake    SupplierStake
}

type QueryInfo struct {
	Chain    string
	Success  bool
	URL      string
	Duration time.Duration
}

type Querier interface {
	GetMetrics(ctx context.Context) ([]prometheus.Collector, []QueryInfo)
}
