# Kava Auction Bot

Automated bot for bidding on collateral and debt auctions on the Kava platform. Note - only compatible with v0.14+ of `kava`.

## Setup

Create a `.env` file:

```
# RPC endpoint
KAVA_RPC_URL="https://rpc.testnet-12000.kava.io:443"
# Mnemonic
KEEPER_MNEMONIC="secret words here"
# Profit margin required for bot to bid (1.5% in the example)
BID_MARGIN="0.015"
```

## Usage

```
go run .
```
