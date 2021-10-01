# Kava Alerts

Bot for sending alerts to Slack for various events on the Kava blockchain.

## Slack Setup

1. [Create a Slack app](https://api.slack.com/apps/new)
2. Add the `chat:write` bot token scope
3. Add the bot to the desired Slack channel

## DynamoDB Setup

1. Create a new DynamoDB table with the primary partition key set to
   `ServiceName` and sort key to `RpcEndpoint`.

The latest alert time is persisted in DynamoDB which is used by
`ALERT_FREQUENCY` below. This will only alert at most once in the given alert
frequency period. For example, `ALERT_FREQUENCY=8h` would at most send 1 Slack
message every 8 hours.

Alert times are saved per service and per RPC URL, meaning that different
networks can have their own separate alerts in the same DynamoDB table. Other
alert services may also use the same table.

## Setup

Copy the example .env file.

```bash
cp .example_env .env
```

Install and start the alerts service with the desired AWS profile.

For auction alerts, run:

```bash
make install

AWS_PROFILE=development $GOPATH/bin/alerts auctions run
```

For USDX price deviation alerts, run:

```bash
make install

AWS_PROFILE=development $GOPATH/bin/alerts usdx run
```
