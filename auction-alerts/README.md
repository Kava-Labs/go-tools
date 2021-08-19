# Kava Auction Alerts

Bot for sending auction related alerts to Slack.

## Slack Setup

1. [Create a Slack app](https://api.slack.com/apps/new)
2. Add the `chat:write` bot token scope
3. Add the bot to the desired Slack channel

## Setup

Create a `.env` file:

```
# RPC endpoint
KAVA_RPC_URL="https://rpc.data.kava.io:443"
# Slack bot user OAuth token
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
