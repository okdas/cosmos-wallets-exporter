# cosmos-wallets-exporter

> **Note**: This is a fork of [QuokkaStake/cosmos-wallets-exporter](https://github.com/QuokkaStake/cosmos-wallets-exporter) with additional Docker and Kubernetes deployment support.

![Docker Image](https://img.shields.io/badge/docker-ghcr.io%2Fokdas%2Fcosmos--wallets--exporter-blue)
[![Build image and push to ghcr.io](https://github.com/okdas/cosmos-wallets-exporter/actions/workflows/docker.yaml/badge.svg)](https://github.com/okdas/cosmos-wallets-exporter/actions/workflows/docker.yaml)

cosmos-wallets-exporter is a Prometheus scraper that fetches the wallet balances from an LCD server exposed by a fullnode.

## What can I use it for?

If you have a wallet that does transactions on an app's behalf without your interaction and will stop working correctly if it cannot broadcast transactions anymore due to zero balance and not enough tokens to pay for transaction fee (some examples: Axelar's broadcaster; Sentinel's dVPN node; ReStake's bot wallets; faucets), you can use this tool to scrape the balances to Prometheus and build alerts if a wallet balance falls under a specific threshold.

## Quick Start with Docker

The easiest way to get started is using our pre-built Docker image:

```sh
# Pull the latest image
docker pull ghcr.io/okdas/cosmos-wallets-exporter:main

# Create a config file (see config.example.toml for reference)
cp config.example.toml my-config.toml
# Edit my-config.toml with your chains and wallets...

# Run the container
docker run -d \
  --name cosmos-wallets-exporter \
  -p 9550:9550 \
  -v $(pwd)/my-config.toml:/app/config.toml \
  ghcr.io/okdas/cosmos-wallets-exporter:main --config /app/config.toml
```

## Kubernetes Deployment with Helm

This fork includes a Helm chart for easy Kubernetes deployment:

```sh
# Clone this repository
git clone https://github.com/okdas/cosmos-wallets-exporter.git
cd cosmos-wallets-exporter

# Install with Helm
helm install cosmos-wallets-exporter ./charts/cosmos-wallets-exporter \
  --set config.chains[0].name="osmosis" \
  --set config.chains[0].lcd-endpoint="https://lcd-osmosis.blockapsis.com" \
  --set config.chains[0].denoms[0].denom="uosmo" \
  --set config.chains[0].denoms[0].display-denom="osmo" \
  --set config.chains[0].denoms[0].coingecko-currency="osmosis" \
  --set config.chains[0].wallets[0].address="osmo1..." \
  --set config.chains[0].wallets[0].group="validator" \
  --set config.chains[0].wallets[0].name="my-osmosis-wallet"

# Or create a custom values.yaml file
helm install cosmos-wallets-exporter ./charts/cosmos-wallets-exporter -f my-values.yaml
```

### Enable Prometheus Monitoring

To enable automatic Prometheus scraping with Prometheus Operator:

```sh
helm upgrade cosmos-wallets-exporter ./charts/cosmos-wallets-exporter \
  --set serviceMonitor.enabled=true \
  --set serviceMonitor.additionalLabels.release=prometheus-operator
```

## Manual Installation

## How can I set it up?

First, you need to download the latest release from [the releases page](https://github.com/QuokkaStake/cosmos-wallets-exporter/releases/). After that, you should unzip it, and you are ready to go:

```sh
wget <the link from the releases page>
tar xvfz <file you just downloaded>
./cosmos-wallets-exporter
```

Alternatively, you can build it from source (golang >= 1.21 is required):
```sh
git clone https://github.com/okdas/cosmos-wallets-exporter.git
cd cosmos-wallets-exporter
# Either build it (this will put the resulting binary into the current folder)...
make build
# ... or install it, which will put the resulting binary into $GOPATH/bin
make install
```

To run it detached, you need to run it as a systemd service. First, we have to copy the file to the system apps folder:

```sh
sudo cp ./cosmos-wallets-exporter /usr/bin
```

Then we need to create a systemd service for our app:

```sh
sudo nano /etc/systemd/system/cosmos-wallets-exporter.service
```

You can use this template (change the user to whatever user you want this to be executed from. It's advised to create a separate user for that instead of running it from root):

```
[Unit]
Description=Cosmos Wallets Exporter
After=network-online.target

[Service]
User=<username>
TimeoutStartSec=0
CPUWeight=95
IOWeight=95
ExecStart=cosmos-wallets-exporter --config <path to config>
Restart=always
RestartSec=2
LimitNOFILE=800000
KillSignal=SIGTERM

[Install]
WantedBy=multi-user.target
```

Then we'll add this service to autostart and run it:

```sh
sudo systemctl daemon-reload # reflect the systemd file change
sudo systemctl enable cosmos-wallets-exporter # enable the scraper to run on system startup
sudo systemctl start cosmos-wallets-exporter # start it
sudo systemctl status cosmos-wallets-exporter # validate it's running
```

If you need to, you can also see the logs of the process:

```sh
sudo journalctl -u cosmos-wallets-exporter -f --output cat
```

## How can I scrape data from it?

Here's the example of the Prometheus config you can use for scraping data:

```yaml
scrape-configs:
  - job_name:       'cosmos-wallets-exporter'
    scrape_interval: 30s
    static_configs:
      - targets:
        - localhost:9550 # replace localhost with scraper IP if it's on the other host
```

Then restart Prometheus and you're good to go!

All the metrics provided by cosmos-wallets-exporter have the `cosmos_wallets_exporter_` as a prefix, here's the list of the exposed metrics:
- `cosmos_wallets_exporter_balance` - wallet balance in tokens.
- `cosmos_wallets_exporter_price` - a price of 1 token on chain.
- `cosmos_wallets_exporter_success` - a count of successful queries for chain.
- `cosmos_wallets_exporter_error` - a count of failed queries for chain. You may use it in alerting to get notified if some of your requests are failing because the node is down.
- `cosmos_wallets_exporter_timings` - time it took to get a response from an LCD endpoint, in seconds.

## How can I configure it?

All configuration is done via the .toml config file, which is passed to the application via the `--config` app parameter. Check `config.example.toml` for a config reference.

## How can I contribute?

Bug reports and feature requests are always welcome! If you want to contribute, feel free to open issues or PRs.
