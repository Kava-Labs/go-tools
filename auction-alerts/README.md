# Kava Auction Alerts

Bot for sending auction related alerts to Slack.

## Slack Setup

1. [Create a Slack app](https://api.slack.com/apps/new)
2. Add the `chat:write` bot token scope
3. Add the bot to the desired Slack channel

## DynamoDB Setup

1. Create a new DynamoDB table with the primary parition key set to `Service` and
   sort key to `RpcEndpoint`.

The latest alert time is persisted in DynamoDB which is used by
`ALERT_FREQUENCY` below. This will only alert at most once in the given alert
frequency period. For example, `ALERT_FREQUENCY=8h` would at most send 1 Slack
message every 8 hours.

Alert times are saved per service and per RPC URL, meaning that different
networks can have their own separate alerts in the same DynamoDB table. Other
alert services may also use the same table.

## Setup

Create a `.env` file:

```
# RPC endpoint
KAVA_RPC_URL="https://rpc.data.kava.io:443"
DYNAMODB_TABLE_NAME="service_alerts"
# Slack bot user OAuth token
SLACK_TOKEN="slack_token"
SLACK_CHANNEL_ID="channel_id"
# Interval at which the process runs to check ongoing auctions
INTERVAL="10m"
# How frequent an alert will be sent when ongoing auctions exceed threshold
ALERT_FREQUENCY="8h"
# US dollar value of auctions that triggers alert
USD_THRESHOLD="100000"
```

## Usage

```
go run .
```
