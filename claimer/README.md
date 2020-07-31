# Claimer

Claimer is a bot that services user claim requests. After a request is submitted to claimer's REST api it is sent to individual workers which process and send the transaction. The primary use case is improving a Kava user's experience on web applications by handling the claim transaction for them. User funds are safe, as the random number communicated between the web application and this bot cannot be used to change the swap's intended recipient.

## Set up

Update the configuration file located at config/config.json with your account mnemonics. These mnemonics must be associated with addresses that actually exist on each respective chain, otherwise the workers are unable to send transactions to the blockchain. Additionally, the accounts must hold a small balance to cover transaction gas fees.

Several mnemonics are recommended for Kava, while only one mnemonic is required for Binance Chain. This is because additional load balancing is required to ensure that each Kava claim transaction is correctly submitted to the blockchain.

Claimer requires a connection to Kava's RPC server.
```bash
kvd start
```

Start the Claimer bot in another process.
```bash
go run main.go
```

You should see a message similar to:
```bash
# Loading configuration path ~/go/src/github.com/kava-labs/go-tools/claimer/config/config.json
# Starting server...
```

Claimer is now ready to receive claim requests.

## Usage

Claims are submitted to the HTTP server via POST. Requests should be sent to the host url at port 8080 on route /claim, such as `http://localhost:8080/claim`.

Claim requests have three required parameters:
- target-chain: must be 'kava' or 'binance'/binance chain'
- swap-id
- random-number

Putting it all together, this a valid HTTP POST claim request:

`http://localhost:8080/claim?target-chain=kava&swap-id=C1EF52C8762BEC5BE6AA53970019D799E7BD4EBBB1D2BD2D7EF471978088729C&random-number=FC5CD40E6DC8EC5E9A0F70D914D76072D5AE9091619B108D3BFE1E356BD22EA8`
