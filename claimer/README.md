# Claimer

Claimer is a bot that services user claim requests. After a request is submitted to claimer's REST api it is sent to individual workers which process and send the transaction. The primary use case is improving a Kava user's experience on web applications by handling the claim transaction for them. User funds are safe, as the random number communicated between the web application and this bot cannot be used to change the swap's intended recipient.

## Set up

Update the configuration file located at config/config.json with your account mnemonics. These mnemonics must be associated with addresses that actually exist on each respective chain, otherwise the workers are unable to send transactions to the blockchain. Additionally, the accounts must hold a small balance to cover transaction gas fees.

Several mnemonics are recommended for Kava, while only one mnemonic is required for Binance Chain. This is because additional load balancing is required to ensure that each Kava claim transaction is correctly submitted to the blockchain.

Claimer requires a connection to Kava's RPC server.
```bash
# for local testing, start the blockchain
kvd start
```

Install and start the Claimer bot
```bash
make install

go run *.go
```

You should see a message similar to:
```bash
# INFO[0000] Loading configuration path ~/go/src/github.com/kava-labs/go-tools/claimer/config/config.json
# I[2020-08-03|12:40:26.000] Starting WSEvents                            impl=WSEvents
# I[2020-08-03|12:40:26.000] Starting WSClient                            impl="WSClient{kava3.data.kava.io:26657 (/websocket
# INFO[0000] Starting server...
```

Claimer is now ready to receive claim requests.

## Usage

Claims are submitted to the HTTP server via POST. Requests should be sent to the host url at port 8080 on route /claim, such as `http://localhost:8080/claim`.

Claim requests have three required parameters:
- target-chain: must be 'kava' or 'binance'/binance chain'
- swap-id
- random-number

Putting it all together, we can build a valid HTTP POST claim request:

`http://localhost:8080/claim?target-chain=kava&swap-id=C1EF52C8762BEC5BE6AA53970019D799E7BD4EBBB1D2BD2D7EF471978088729C&random-number=FC5CD40E6DC8EC5E9A0F70D914D76072D5AE9091619B108D3BFE1E356BD22EA8`

The server should log a message acknowledging the request and attempt to process it.
```bash
INFO[0090] Received claim request for C1EF52C8762BEC5BE6AA53970019D799E7BD4EBBB1D2BD2D7EF471978088729C on kava
INFO[0090] Claim tx sent to Kava: 781B8C6FCDD6BC7F03D86B6C6C217C3407D10482863E15557EDC3D37EF58195A
```

After successfully processing a claim request on Kava, the bot waits a full block (7 seconds) before releasing the claimer worker to prevent sequence errors.

If the initial relay process is unsuccessful, the bot will retry the claim up to 5 times. The bot will log each attempt.
```bash
INFO[0300] Received claim request for 618526297d4deb153f8218ad245e76d16bb2d1130e89ea3c6521130ac1b8019a on binance 
INFO[0310] retrying: swap 618526297d4deb153f8218ad245e76d16bb2d1130e89ea3c6521130ac1b8019a not found in state 
INFO[0320] retrying: swap 618526297d4deb153f8218ad245e76d16bb2d1130e89ea3c6521130ac1b8019a not found in state 
INFO[0330] retrying: swap 618526297d4deb153f8218ad245e76d16bb2d1130e89ea3c6521130ac1b8019a not found in state 
INFO[0341] retrying: swap 618526297d4deb153f8218ad245e76d16bb2d1130e89ea3c6521130ac1b8019a not found in state 
ERRO[0341] timed out after 5 attempts, last error: swap 618526297d4deb153f8218ad245e76d16bb2d1130e89ea3c6521130ac1b8019a not found in state
```