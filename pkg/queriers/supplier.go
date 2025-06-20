package queriers

import (
	"context"
	"main/pkg/config"
	"main/pkg/tendermint"
	"main/pkg/types"
	"math"
	"strconv"
	"sync"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
)

type SupplierQuerier struct {
	Config *config.Config
	Logger zerolog.Logger
	RPCs   []*tendermint.RPC
	Tracer trace.Tracer
}

func NewSupplierQuerier(
	config *config.Config,
	logger zerolog.Logger,
	tracer trace.Tracer,
) *SupplierQuerier {
	rpcs := make([]*tendermint.RPC, len(config.Chains))

	for index, chain := range config.Chains {
		rpcs[index] = tendermint.NewRPC(chain, logger, tracer)
	}

	return &SupplierQuerier{
		Config: config,
		Logger: logger.With().Str("component", "supplier_querier").Logger(),
		RPCs:   rpcs,
		Tracer: tracer,
	}
}

func (q *SupplierQuerier) collectSupplierStakes(ctx context.Context, supplierStakeGauge *prometheus.GaugeVec, revShareMap map[string][]types.RevShareMetadata) ([]types.QueryInfo, map[string]types.SupplierData) {
	var queryInfos []types.QueryInfo
	var wg sync.WaitGroup
	var mutex sync.Mutex
	supplierDataMap := make(map[string]types.SupplierData)

	for index, chain := range q.Config.Chains {
		rpc := q.RPCs[index]

		for _, supplier := range chain.Suppliers {
			wg.Add(1)
			go func(supplier config.Supplier, chain config.Chain, rpc *tendermint.RPC) {
				chainCtx, chainSpan := q.Tracer.Start(ctx, "Querying chain and supplier")
				chainSpan.SetAttributes(attribute.String("chain", chain.Name))
				chainSpan.SetAttributes(attribute.String("supplier", supplier.Address))
				defer chainSpan.End()

				defer wg.Done()

				supplierResponse, queryInfo, err := rpc.GetSupplierStake(supplier.Address, chainCtx)

				mutex.Lock()
				defer mutex.Unlock()

				queryInfos = append(queryInfos, queryInfo)

				if err != nil {
					q.Logger.Error().
						Err(err).
						Str("chain", chain.Name).
						Str("supplier", supplier.Address).
						Msg("Error querying supplier stake")
					return
				}

				supplierData := supplierResponse.Supplier
				supplierKey := chain.Name + ":" + supplier.Address
				supplierDataMap[supplierKey] = supplierData

				// Process stake metric
				q.processSupplierStakeMetric(supplierData, supplier, chain, supplierStakeGauge)

				// Collect rev_share addresses
				q.collectRevShareAddresses(supplierData, supplier, chain, revShareMap)
			}(supplier, chain, rpc)
		}
	}

	wg.Wait()
	return queryInfos, supplierDataMap
}

func (q *SupplierQuerier) processSupplierStakeMetric(supplierData types.SupplierData, supplier config.Supplier, chain config.Chain, supplierStakeGauge *prometheus.GaugeVec) {
	stake := supplierData.Stake
	denom := stake.Denom

	// Convert amount string to float64
	amount, err := strconv.ParseFloat(stake.Amount, 64)
	if err != nil {
		q.Logger.Error().
			Err(err).
			Str("chain", chain.Name).
			Str("supplier", supplier.Address).
			Str("amount", stake.Amount).
			Msg("Error parsing stake amount")
		return
	}

	denomInfo, found := chain.FindDenomByName(stake.Denom)
	if found {
		denom = denomInfo.GetName()
		amount /= math.Pow10(denomInfo.DenomExponent)
	}

	supplierStakeGauge.With(prometheus.Labels{
		"chain":   chain.Name,
		"address": supplier.Address,
		"name":    supplier.Name,
		"group":   supplier.Group,
		"denom":   denom,
	}).Set(amount)
}

func (q *SupplierQuerier) collectRevShareAddresses(supplierData types.SupplierData, supplier config.Supplier, chain config.Chain, revShareMap map[string][]types.RevShareMetadata) {
	detailedMetrics := chain.IsRevShareDetailedMetricsEnabled()
	for _, service := range supplierData.Services {
		for _, revShare := range service.RevShare {
			metadata := types.RevShareMetadata{
				Chain:                chain.Name,
				SupplierOperatorAddr: supplierData.OperatorAddress,
				SupplierOwnerAddr:    supplierData.OwnerAddress,
				SupplierName:         supplier.Name,
				ServiceID:            service.ServiceID,
				RevSharePercentage:   revShare.RevSharePercentage,
				DetailedMetrics:      detailedMetrics,
			}
			revShareMap[revShare.Address] = append(revShareMap[revShare.Address], metadata)
		}
	}
}

func (q *SupplierQuerier) queryRevShareBalances(ctx context.Context, revShareMap map[string][]types.RevShareMetadata) ([]types.QueryInfo, map[string]types.Balances) {
	var queryInfos []types.QueryInfo
	var revShareWg sync.WaitGroup
	var mutex sync.Mutex
	revShareBalances := make(map[string]types.Balances)

	for revShareAddr := range revShareMap {
		// Find the chain for this address (use the first one, they should all be the same chain)
		if len(revShareMap[revShareAddr]) == 0 {
			continue
		}
		chainName := revShareMap[revShareAddr][0].Chain

		// Find the RPC for this chain
		var rpc *tendermint.RPC
		for index, chain := range q.Config.Chains {
			if chain.Name == chainName {
				rpc = q.RPCs[index]
				break
			}
		}

		if rpc == nil {
			continue
		}

		revShareWg.Add(1)
		go func(address string, chainName string, rpc *tendermint.RPC) {
			defer revShareWg.Done()

			balancesResponse, queryInfo, err := rpc.GetWalletBalances(address, ctx)

			mutex.Lock()
			defer mutex.Unlock()

			queryInfos = append(queryInfos, queryInfo)

			if err != nil {
				q.Logger.Error().
					Err(err).
					Str("chain", chainName).
					Str("rev_share_address", address).
					Msg("Error querying rev share balance")
				return
			}

			revShareBalances[address] = balancesResponse.Balances
		}(revShareAddr, chainName, rpc)
	}

	revShareWg.Wait()
	return queryInfos, revShareBalances
}

func (q *SupplierQuerier) createRevShareMetrics(revShareMap map[string][]types.RevShareMetadata, revShareBalances map[string]types.Balances, detailedGauge *prometheus.GaugeVec, aggregateGauge *prometheus.GaugeVec) []prometheus.Collector {
	var usedCollectors []prometheus.Collector
	aggregateBalances := make(map[types.RevShareAggregateKey]map[string]float64)

	for revShareAddr, metadataList := range revShareMap {
		balances, found := revShareBalances[revShareAddr]
		if !found || len(metadataList) == 0 {
			continue
		}

		// Find the chain config to get denom info (use first metadata entry)
		firstMetadata := metadataList[0]
		var chainConfig config.Chain
		for _, chain := range q.Config.Chains {
			if chain.Name == firstMetadata.Chain {
				chainConfig = chain
				break
			}
		}

		for _, metadata := range metadataList {
			for _, balance := range balances {
				denom := balance.Denom
				amount := balance.Amount.MustFloat64()

				denomInfo, found := chainConfig.FindDenomByName(balance.Denom)
				if found {
					denom = denomInfo.GetName()
					amount /= math.Pow10(denomInfo.DenomExponent)
				}

				if metadata.DetailedMetrics {
					detailedGauge.With(prometheus.Labels{
						"chain":                     metadata.Chain,
						"supplier_operator_address": metadata.SupplierOperatorAddr,
						"supplier_owner_address":    metadata.SupplierOwnerAddr,
						"supplier_name":             metadata.SupplierName,
						"rev_share_address":         revShareAddr,
						"service_id":                metadata.ServiceID,
						"rev_share_percentage":      metadata.RevSharePercentage,
						"denom":                     denom,
					}).Set(amount)
				} else {
					aggKey := types.RevShareAggregateKey{
						Chain:                metadata.Chain,
						SupplierOperatorAddr: metadata.SupplierOperatorAddr,
						SupplierOwnerAddr:    metadata.SupplierOwnerAddr,
						SupplierName:         metadata.SupplierName,
						RevShareAddress:      revShareAddr,
					}

					if aggregateBalances[aggKey] == nil {
						aggregateBalances[aggKey] = make(map[string]float64)
					}
					aggregateBalances[aggKey][denom] += amount
				}
			}
		}
	}

	// Check if we have detailed metrics
	hasDetailedMetrics := false
	for _, metadataList := range revShareMap {
		for _, metadata := range metadataList {
			if metadata.DetailedMetrics {
				hasDetailedMetrics = true
				break
			}
		}
		if hasDetailedMetrics {
			break
		}
	}

	if hasDetailedMetrics {
		usedCollectors = append(usedCollectors, detailedGauge)
	}

	if len(aggregateBalances) > 0 {
		for aggKey, denomBalances := range aggregateBalances {
			for denom, amount := range denomBalances {
				aggregateGauge.With(prometheus.Labels{
					"chain":                     aggKey.Chain,
					"supplier_operator_address": aggKey.SupplierOperatorAddr,
					"supplier_owner_address":    aggKey.SupplierOwnerAddr,
					"supplier_name":             aggKey.SupplierName,
					"rev_share_address":         aggKey.RevShareAddress,
					"denom":                     denom,
				}).Set(amount)
			}
		}
		usedCollectors = append(usedCollectors, aggregateGauge)
	}

	return usedCollectors
}

func (q *SupplierQuerier) GetMetrics(ctx context.Context) ([]prometheus.Collector, []types.QueryInfo) {
	childCtx, span := q.Tracer.Start(ctx, "Querying supplier stake and rev share metrics")
	defer span.End()

	supplierStakeGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cosmos_wallets_exporter_supplier_stake",
			Help: "A Pocket Network supplier stake (in tokens)",
		},
		[]string{"chain", "address", "name", "group", "denom"},
	)

	revShareBalanceDetailedGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cosmos_wallets_exporter_supplier_rev_share_balance_detailed",
			Help: "Detailed balance of revenue share addresses for Pocket Network suppliers (per service and percentage)",
		},
		[]string{"chain", "supplier_operator_address", "supplier_owner_address", "supplier_name", "rev_share_address", "service_id", "rev_share_percentage", "denom"},
	)

	revShareBalanceAggregateGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cosmos_wallets_exporter_supplier_rev_share_balance",
			Help: "Aggregated balance of revenue share addresses for Pocket Network suppliers (summed across services)",
		},
		[]string{"chain", "supplier_operator_address", "supplier_owner_address", "supplier_name", "rev_share_address", "denom"},
	)

	// Phase 1: Collect supplier data and rev_share addresses
	revShareMap := make(map[string][]types.RevShareMetadata)
	queryInfos1, _ := q.collectSupplierStakes(childCtx, supplierStakeGauge, revShareMap)

	// Phase 2: Query unique rev_share addresses
	queryInfos2, revShareBalances := q.queryRevShareBalances(childCtx, revShareMap)

	// Phase 3: Create rev_share balance metrics
	usedCollectors := q.createRevShareMetrics(revShareMap, revShareBalances, revShareBalanceDetailedGauge, revShareBalanceAggregateGauge)

	// Combine query infos
	var allQueryInfos []types.QueryInfo
	allQueryInfos = append(allQueryInfos, queryInfos1...)
	allQueryInfos = append(allQueryInfos, queryInfos2...)

	// Always include supplier stake gauge
	finalCollectors := []prometheus.Collector{supplierStakeGauge}
	finalCollectors = append(finalCollectors, usedCollectors...)

	return finalCollectors, allQueryInfos
}
