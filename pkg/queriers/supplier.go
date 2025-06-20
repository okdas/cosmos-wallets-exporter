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

func (q *SupplierQuerier) GetMetrics(ctx context.Context) ([]prometheus.Collector, []types.QueryInfo) {
	childCtx, span := q.Tracer.Start(ctx, "Querying supplier stake metrics")
	defer span.End()

	supplierStakeGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cosmos_wallets_exporter_supplier_stake",
			Help: "A Pocket Network supplier stake (in tokens)",
		},
		[]string{"chain", "address", "name", "group", "denom"},
	)

	var queryInfos []types.QueryInfo

	var wg sync.WaitGroup
	var mutex sync.Mutex

	for index, chain := range q.Config.Chains {
		rpc := q.RPCs[index]

		for _, supplier := range chain.Suppliers {
			wg.Add(1)
			go func(supplier config.Supplier, chain config.Chain, rpc *tendermint.RPC) {
				chainCtx, chainSpan := q.Tracer.Start(childCtx, "Querying chain and supplier")
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

				stake := supplierResponse.Supplier.Stake
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
			}(supplier, chain, rpc)
		}
	}

	wg.Wait()

	return []prometheus.Collector{supplierStakeGauge}, queryInfos
}
