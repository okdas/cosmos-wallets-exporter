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

type RevShare struct {
	Address            string `json:"address"`
	RevSharePercentage string `json:"rev_share_percentage"`
}

type ServiceConfig struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type ServiceEndpoint struct {
	Configs []ServiceConfig `json:"configs"`
	RpcType string          `json:"rpc_type"`
	URL     string          `json:"url"`
}

type Service struct {
	Endpoints []ServiceEndpoint `json:"endpoints"`
	RevShare  []RevShare        `json:"rev_share"`
	ServiceID string            `json:"service_id"`
}

type ServiceConfigHistoryEntry struct {
	ActivationHeight   string  `json:"activation_height"`
	DeactivationHeight string  `json:"deactivation_height"`
	OperatorAddress    string  `json:"operator_address"`
	Service            Service `json:"service"`
}

type SupplierData struct {
	OperatorAddress         string                      `json:"operator_address"`
	OwnerAddress            string                      `json:"owner_address"`
	ServiceConfigHistory    []ServiceConfigHistoryEntry `json:"service_config_history"`
	Services                []Service                   `json:"services"`
	Stake                   SupplierStake               `json:"stake"`
	UnstakeSessionEndHeight string                      `json:"unstake_session_end_height"`
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

type RevShareMetadata struct {
	Chain                string
	SupplierOperatorAddr string
	SupplierOwnerAddr    string
	SupplierName         string
	ServiceID            string
	RevSharePercentage   string
	DetailedMetrics      bool
}

type RevShareAggregateKey struct {
	Chain                string
	SupplierOperatorAddr string
	SupplierOwnerAddr    string
	SupplierName         string
	RevShareAddress      string
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
