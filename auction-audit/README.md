# auction-audit

auction-audit is a tool to generate CSV formatted data for auctions between a
provided height range.

## Config

Create an `.env` file with the following keys. Values are provided as examples.

```env
GRPC_URL=https://grpc.data.kava.io:443
RPC_URL=https://rpc.data.kava.io:443
START_HEIGHT=2824700
END_HEIGHT=2824800
```

## Usage

```
go run .
```

## Output data example

| Auction ID | End Height | Asset Purchased | Amount Purchased        | Asset Paid | Amount Paid             | Initial Lot             | Liquidated Account                          | Winning Bidder Account                      | USD Value Before Liquidation | USD Value After Liquidation | Amount Returned         | Percent Loss (quantity) | Percent Loss (USD value) |
| ---------- | ---------- | --------------- | ----------------------- | ---------- | ----------------------- | ----------------------- | ------------------------------------------- | ------------------------------------------- | ---------------------------- | --------------------------- | ----------------------- | ----------------------- | ------------------------ |
| 15238      | 2288999    | BUSD            | 0.000007470000000000    | USDX       | 0.000007000000000000    | 0.000024760000000000    | kava14g0rlp6et4ytx6pcze7jq307vxva5pwktzzrzu | kava1me2recm86t27c24q0kple0cdd5jlglqnqz9l3t | 0.000024760000000000         | 0.000017290000000000        | 0.000017290000000000    | 0.301696284329563813    | 0.301696284329563813     |
| 15239      | 2288999    | KAVA            | 4.841625000000000000    | USDX       | 4.314059000000000000    | 15.683923000000000000   | kava14g0rlp6et4ytx6pcze7jq307vxva5pwktzzrzu | kava1me2recm86t27c24q0kple0cdd5jlglqnqz9l3t | 15.009514310999999404        | 9.042476531999999610        | 10.842298000000000000   | 0.308699870561721069    | 0.397550357417424633     |
| 15241      | 2288565    | KAVA            | 3659.531595000000000000 | USDX       | 3250.520387000000000000 | 5000.000000000000000000 | kava160cd0jk7exran8ap9twfuwup8tatxeqzrxm630 | kava1nsh262f50musa6vr4a03ft0xq0ar86aln4udsy | 4754.999999999999780000      | 1136.717207439999967829     | 1340.468405000000000000 | 0.731906319000000000    | 0.760942753430073602     |
