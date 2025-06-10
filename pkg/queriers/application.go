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

type ApplicationQuerier struct {
	Config *config.Config
	Logger zerolog.Logger
	RPCs   []*tendermint.RPC
	Tracer trace.Tracer
}

func NewApplicationQuerier(
	config *config.Config,
	logger zerolog.Logger,
	tracer trace.Tracer,
) *ApplicationQuerier {
	rpcs := make([]*tendermint.RPC, len(config.Chains))

	for index, chain := range config.Chains {
		rpcs[index] = tendermint.NewRPC(chain, logger, tracer)
	}

	return &ApplicationQuerier{
		Config: config,
		Logger: logger.With().Str("component", "application_querier").Logger(),
		RPCs:   rpcs,
		Tracer: tracer,
	}
}

func (q *ApplicationQuerier) GetMetrics(ctx context.Context) ([]prometheus.Collector, []types.QueryInfo) {
	childCtx, span := q.Tracer.Start(ctx, "Querying application stake metrics")
	defer span.End()

	applicationStakeGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cosmos_wallets_exporter_application_stake",
			Help: "A Pocket Network application stake (in tokens)",
		},
		[]string{"chain", "address", "name", "group", "denom"},
	)

	var queryInfos []types.QueryInfo

	var wg sync.WaitGroup
	var mutex sync.Mutex

	for index, chain := range q.Config.Chains {
		rpc := q.RPCs[index]

		for _, application := range chain.Applications {
			wg.Add(1)
			go func(application config.Application, chain config.Chain, rpc *tendermint.RPC) {
				chainCtx, chainSpan := q.Tracer.Start(childCtx, "Querying chain and application")
				chainSpan.SetAttributes(attribute.String("chain", chain.Name))
				chainSpan.SetAttributes(attribute.String("application", application.Address))
				defer chainSpan.End()

				defer wg.Done()

				applicationResponse, queryInfo, err := rpc.GetApplicationStake(application.Address, chainCtx)

				mutex.Lock()
				defer mutex.Unlock()

				queryInfos = append(queryInfos, queryInfo)

				if err != nil {
					q.Logger.Error().
						Err(err).
						Str("chain", chain.Name).
						Str("application", application.Address).
						Msg("Error querying application stake")
					return
				}

				stake := applicationResponse.Application.Stake
				denom := stake.Denom

				// Convert amount string to float64
				amount, err := strconv.ParseFloat(stake.Amount, 64)
				if err != nil {
					q.Logger.Error().
						Err(err).
						Str("chain", chain.Name).
						Str("application", application.Address).
						Str("amount", stake.Amount).
						Msg("Error parsing stake amount")
					return
				}

				denomInfo, found := chain.FindDenomByName(stake.Denom)
				if found {
					denom = denomInfo.GetName()
					amount /= math.Pow10(denomInfo.DenomExponent)
				}

				applicationStakeGauge.With(prometheus.Labels{
					"chain":   chain.Name,
					"address": application.Address,
					"name":    application.Name,
					"group":   application.Group,
					"denom":   denom,
				}).Set(amount)
			}(application, chain, rpc)
		}
	}

	wg.Wait()

	return []prometheus.Collector{applicationStakeGauge}, queryInfos
}
