# Claimer

Claimer is a bot that services user claim requests. After a request is submitted to claimer's REST api it is sent to individual workers which process and send the transaction. The primary use case is improving a Kava user's experience on web applications by handling the claim transaction for them. User funds are safe, as the random number communicated between the web application and this bot cannot be used to change the swap's intended recipient.

## Set up

Update the configuration file located at config/config.json with your account mnemonics. These mnemonics must be associated with addresses that actually exist on each respective chain, otherwise the workers are unable to send transactions to the blockchain. Additionally, the accounts must hold a small balance to cover transaction gas fees.

Several mnemonics are recommended for Kava, while only one mnemonic is required for Binance Chain. This is because additional load balancing is required to ensure that each Kava claim transaction is correctly submitted to the blockchain.

Claimer requires a connection to Kava's GRPC server.

```bash
# for local testing, start the blockchain
kava start
```

Install and start the Claimer bot

```bash
make install

$GOPATH/bin/claimer
```

You should see a message similar to:

```bash
> INFO[0000] Loading configuration path ~/go/src/github.com/kava-labs/go-tools/claimer/config/config.json
> I[2020-08-03|12:40:26.000] Starting WSEvents                            impl=WSEvents
> I[2020-08-03|12:40:26.000] Starting WSClient                            impl="WSClient{kava3.data.kava.io:26657 (/websocket
> INFO[0000] Starting server...
```

Claimer is now ready to receive claim requests.

## Usage

Claims are submitted to the HTTP server via POST. Requests should be sent to the host url at port 8080 on route /claim, such as `http://localhost:8080/claim`.

Claim requests have three required parameters:

- target-chain: must be 'kava' or 'binance'/binance chain'
- swap-id
- random-number

Putting it all together, we can build a valid HTTP POST claim request:

`http://localhost:8080/claim?target-chain=kava&swap-id=4c8bd80e18d386777c2e507a0579307a54b910a6828d75375ac4bd2eae86c31c&random-number=9b955e36ac165c6b8ab69b9af5cd042a37c5fddb85129a80cc2138b6e22ef940`

The server will log a message acknowledging the request and attempt to process it.

```bash
> level=info msg="claim request received" request_id=b69c1636-258b-440d-b9a3-b3e47c385c1f url="/claim?target-chain=kava&swap-id=4c8bd80e18d386777c2e507a0579307a54b910a6828d75375ac4bd2eae86c31c&random-number=9b955e36ac165c6b8ab69b9af5cd042a37c5fddb85129a80cc2138b6e22ef940"
> level=info msg="claim request submitted to queue for processing" request_id=b69c1636-258b-440d-b9a3-b3e47c385c1f swap_id=4c8bd80e18d386777c2e507a0579307a54b910a6828d75375ac4bd2eae86c31c target_chain=kava
> level=info msg="claim request begin processing" request_id=b69c1636-258b-440d-b9a3-b3e47c385c1f swap_id=4c8bd80e18d386777c2e507a0579307a54b910a6828d75375ac4bd2eae86c31c target_chain=kava
```

After successfully processing a claim request on Kava, the bot waits until the tx is included in a block before releasing the claimer worker to prevent sequence errors.

If the initial relay process is unsuccessful, the bot will retry the claim several times. The bot will log each attempt.

```bash
> level=debug msg="claim retrying" error="swap 4c8bd80e18d386777c2e507a0579307a54b910a6828d75375ac4bd2eae86c31c not found in state" recipient= request_id=b69c1636-258b-440d-b9a3-b3e47c385c1f swap_id=4c8bd80e18d386777c2e507a0579307a54b910a6828d75375ac4bd2eae86c31c target_chain=kava
> level=debug msg="claim retrying" error="rpc error: code = Unavailable desc = Bad Gateway: HTTP status code 502; transport: received the unexpected content-type \"text/html\"" recipient= request_id=b69c1636-258b-440d-b9a3-b3e47c385c1f swap_id=4c8bd80e18d386777c2e507a0579307a54b910a6828d75375ac4bd2eae86c31c target_chain=kava
```

then log success

```bash
> level=info msg="claim confirmed" recipient=kava173w2zz287s36ewnnkf4mjansnthnnsz7rtrxqc request_id=b69c1636-258b-440d-b9a3-b3e47c385c1f swap_id=4c8bd80e18d386777c2e507a0579307a54b910a6828d75375ac4bd2eae86c31c target_chain=kava tx_hash=F937A1142E6B47F9A5546680E369A515433271757B83A89069619FB3C004E0AD
```

or failure

```bash
> level=error msg="claim failed" error="rpc error: code = Unavailable desc = Bad Gateway: HTTP status code 502; transport: received the unexpected content-type \"text/html\"" recipient=kava173w2zz287s36ewnnkf4mjansnthnnsz7rtrxqc request_id=b69c1636-258b-440d-b9a3-b3e47c385c1f swap_id=4c8bd80e18d386777c2e507a0579307a54b910a6828d75375ac4bd2eae86c31c target_chain=kava
```

# Development

To run e2e tests, ensure docker is running, then

```bash
make test-integration
```