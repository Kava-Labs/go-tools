# Kava Alerts

Bot for sending alerts to Slack for various events on the Kava blockchain.

## Slack Setup

1. Create a Slack Workflow that starts with webhook
   * Channel settings > Integrations > Add Automation > Start with webhook
   * Use this webhook's URL as `SLACK_WEBHOOK_URL`
2. Configure the workflow with data variable `text`
3. Output the `text` variable to the desired channel

There is a utility command `alerts slack-test` that can be used to test your workflow's webhook trigger.

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

## Alerts

### Auction

Auction alerts tracks ongoing auctions on the Kava Chain and alerts for:

- Total value of auctions above configured value
- Percentage price deviation of auction clearing price above configured value

Runs with:
```bash
go run . auctions run
```

### USDX

USDX alerts tracks the USDX price on the Kava Chain and alerts for:

- Price deviation of USDX below configured value

Runs with:
```bash
go run . usdx run
```