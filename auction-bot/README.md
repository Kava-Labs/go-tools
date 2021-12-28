# Kava Auction Bot

Automated bot for bidding on collateral and debt auctions on the Kava platform. Note - only compatible with v0.14+ of `kava`.

## Setup

Create a `.env` file:

```
# GRPC endpoint
KAVA_GRPC_URL="grpc.testnet.kava.io:443"
# Mnemonic
KEEPER_MNEMONIC="secret words here"
# Profit margin required for bot to bid (1.5% in the example)
BID_MARGIN="0.015"
```

## Usage

```
go run .
```

Bot will bid attempt to bid on all auctions where the profit margin is greater than what is specified in `BID_MARGIN`. Note, bot does not currently track account balances, so it will attempt to create bids even for auctions for which it doesn't have sufficient funds.  
