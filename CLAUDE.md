# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

### Build and Development
- `make build` - Build the binary (outputs to current directory)
- `make install` - Install binary to $GOPATH/bin
- `make lint` - Run golangci-lint with automatic fixes
- `make test` - Run tests with coverage
- `make coverage` - Generate HTML coverage report

### Running the Application
- `./cosmos-wallets-exporter --config config.toml` - Run with config file
- Docker: `docker run -p 9550:9550 -v $(pwd)/config.toml:/app/config.toml ghcr.io/okdas/cosmos-wallets-exporter:main --config /app/config.toml`

### Testing Individual Components
- `go test -v ./pkg/config` - Test config package
- `go test -v ./pkg/queriers` - Test queriers
- `go test -run TestSpecificFunction ./pkg/...` - Run specific test

## Architecture

This is a Prometheus exporter for monitoring Cosmos SDK wallet balances. Key architectural patterns:

### Core Flow
1. **Configuration** (`/pkg/config`) - Parses TOML config defining chains, wallets, and denoms
2. **Application** (`/pkg/app.go`) - Orchestrates queries across multiple chains concurrently
3. **Queriers** (`/pkg/queriers`) - Fetch data from LCD endpoints:
   - `balance.go` - Queries wallet balances via `/cosmos/bank/v1beta1/balances/{address}`
   - `price.go` - Fetches token prices from CoinGecko API
   - `application_stake.go` - Queries Pocket Network application stakes
4. **HTTP Server** (`/pkg/http`) - Exposes metrics on `/metrics` endpoint (default port 9550)

### Key Design Patterns
- **Interface-based design**: All queriers implement common interfaces for testability
- **Concurrent execution**: Queries run in parallel with proper timeout handling
- **Dependency injection**: Components receive dependencies through constructors
- **Error tracking**: Failed queries are tracked as metrics for alerting

### Configuration Processing
The application processes config through multiple stages:
1. TOML parsing into structs
2. Validation of required fields
3. Conversion for Helm chart integration (handles denom-exponent formatting)

### Metrics Architecture
Each metric includes labels for filtering:
- `chain` - Chain name from config
- `address` - Wallet/application address
- `denom` - Token denomination
- `group` - Logical grouping (e.g., "validator", "restake")
- `name` - Human-readable identifier

### Special Features
- **Pocket Network Support**: Applications are dual-monitored as both staked apps and wallets
- **Multi-denom Support**: Each chain can have multiple token denominations
- **Price Integration**: Optional USD value calculation via CoinGecko

## Pocket Network Entity Monitoring

This exporter includes custom logic for monitoring Pocket Network entities like applications and suppliers.

### Application Monitoring

#### How It Works
1. **Dual Monitoring**: Each Pocket application is automatically monitored in two ways:
   - **Application Stake**: Queries `/pokt-network/poktroll/application/application/{address}` for staked amount
   - **Wallet Balance**: Queries standard Cosmos balance endpoint for liquid tokens

2. **Configuration**: Applications are defined in the chain config:
   ```toml
   [[chains]]
   name = "poktroll"
   lcd-endpoint = "https://your-pokt-lcd-endpoint"
   applications = [
       { address = "pokt1abc123...", group = "gateway", name = "my-pocket-app" }
   ]
   ```

3. **Metrics Generated**:
   - `cosmos_wallets_exporter_application_stake` - Staked/bonded POKT tokens
   - `cosmos_wallets_exporter_balance` - Liquid/available POKT tokens

### Supplier Monitoring

#### How It Works
1. **Dual Monitoring**: Each Pocket supplier is automatically monitored in two ways:
   - **Supplier Stake**: Queries `/pokt-network/poktroll/supplier/supplier/{operator_address}` for staked amount
   - **Wallet Balance**: Queries standard Cosmos balance endpoint for liquid tokens

2. **Configuration**: Suppliers are defined in the chain config:
   ```toml
   [[chains]]
   name = "poktroll"
   lcd-endpoint = "https://your-pokt-lcd-endpoint"
   suppliers = [
       { address = "pokt1xyz789...", group = "supplier", name = "my-pocket-supplier" }
   ]
   ```

3. **Metrics Generated**:
   - `cosmos_wallets_exporter_supplier_stake` - Staked/bonded POKT tokens
   - `cosmos_wallets_exporter_balance` - Liquid/available POKT tokens

### Implementation Details
- **Application Querier** (`/pkg/queriers/application.go`): Fetches application stake data
- **Supplier Querier** (`/pkg/queriers/supplier.go`): Fetches supplier stake data
- **Custom RPC Methods** (`/pkg/tendermint/tendermint.go`): 
  - `GetApplicationStake()` handles application-specific endpoint
  - `GetSupplierStake()` handles supplier-specific endpoint
- **Auto-inclusion**: Applications and suppliers don't need to be duplicated in wallets array - they're automatically included in balance monitoring

### Data Structures
**Application endpoint** returns:
- `stake`: Contains `amount` and `denom` for staked tokens
- `delegatee_gateway_addresses`: Associated gateways
- `service_configs`: Services the application provides
- `unstake_session_end_height`: Unstaking status

**Supplier endpoint** returns:
- `stake`: Contains `amount` and `denom` for staked tokens
- `operator_address`: Operator address of the supplier
- `owner_address`: Owner address of the supplier
- `services`: Services the supplier provides
- `unstake_session_end_height`: Unstaking status

### Extensibility
The architecture is designed to easily add more Pocket Network entities:
- New queriers can be added following the existing pattern
- The tendermint client can be extended with new RPC methods
- Metrics follow consistent labeling (chain, address, name, group)

## Deployment Considerations

### Helm Chart (`/charts/cosmos-wallets-exporter`)
The Helm chart handles:
- ConfigMap generation from values
- ServiceMonitor/PodMonitor for Prometheus Operator
- Security contexts (non-root user 65532)
- Resource limits and health checks

### Docker Image
Multi-stage build produces ~15MB Alpine-based image:
- Non-root execution
- Health checks included
- Published to `ghcr.io/okdas/cosmos-wallets-exporter`