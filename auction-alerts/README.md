# Kava Auction Alerts

Bot for sending auction related alerts to Slack.

## Setup

Create a `.env` file:

```
# RPC endpoint
KAVA_RPC_URL="https://rpc.data.kava.io:443"


# Slack bot token
SLACK_TOKEN="slack_token"
SLACK_CHANNEL_ID="channel_id"
# Interval at which the process runs to check ongoing auctions
INTERVAL="10m"
# US dollar value of auctions that triggers alert
USD_THRESHOLD="100000"
```

## Usage

```
go run .
```
