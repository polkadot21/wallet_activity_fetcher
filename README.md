# Top `N` Active Wallet Fetcher

This application is designed to fetch the top N active wallets on the Ethereum mainnet blockchain,
where N is a user-defined parameter.

## What is an active wallet?

An active wallet is one that engages in receiving or transferring ERC20 tokens. The activity score of a wallet 
increases with the number of transactions it participates in.

## RPC Node Docs

For detailed information on the Ethereum JSON-RPC API and specifics about the `eth_blockNumber` method used by our
application, please refer to the official documentation provided by `GetBlock`:

[Ethereum JSON-RPC API Documentation](https://getblock.io/docs/eth/json-rpc/eth_eth_blocknumber/)


## Env vars:

For the demo, the application fetches the top 5 active wallets from
the past 100 blocks.

```bash
RPC_ENDPOINT=<RPC_ENDPOINT>
N_TOP_WALLETS=5
N_BLOCKS=100
LOG_LEVEL=debug
SAVE_PATH=<SAVE_PATH>
```

## How to run?

#### Build & Run Locally

Provide the environment variables in a `.env` file.
```bash
# Fetch the dependencies for your application
go mod download

# Build your application
go build -o walletActivityParser .

# Run the application
./walletActivityParser

```
The results will be stored in a JSON file at `./<save_path>`.

#### Docker

Provide the environment variables in a `.env` file.

```bash
# Build and run your application using Docker Compose
docker compose up --build -d
```
The results will be stored in a JSON file at `./exported_data/<save_path>`.