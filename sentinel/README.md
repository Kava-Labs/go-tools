# Sentinel - _keeping your CDPs safe_

Sentinel is a bot that will watch over a CDP on the kava blockchain and try to prevent it from being liquidated.

Sentinel will keep a CDP’s collateral ratio within a range by repaying (or withdrawing) debt.

## Get Started

Sentinel has been designed to be as reliable as possible. To further increase reliability, operators can run more than one copy of sentinel on different servers, using different kava nodes.

  1) Install go
  2) Install sentinel with `git clone <this repo>`, `make install`  
       (`go install github.com/kava-labs/go-tools/sentinel` should also work)
  3) Create your `config.toml` file based off `config.toml.example`. Documentation can be found in `main.go`
  4) Run `sentinel` (with `config.toml` in the current directory, or in `~/.sentinel/`)

## Design

Sentinel is designed to have multiple copies running at once. This redundancy should make it possible to reliably get a tx confirmed quickly during critical market movements.

It is important to ensure there is no race conditions between copies. For example two bots could decide to withdraw debt and each send in tx. If both txs are confirmed double the amount of debt will be withdrawn.

Sentinel achieves safety by ensuring, per iteration, all queried state is from the same block height. The data used to decide to repay/withdraw is from the same height as the cdp's account sequence number. So between many copies, for a given height only one bot will have a tx confirmed. All others will fail as they used the same sequence number.

## Future Improvements

- alerting
- improved secret handling
- support for CDPs of different denoms
