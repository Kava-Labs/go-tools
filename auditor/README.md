# Kava Auditor

## Inputs

* Start block
* End block
* Grpc endpoint
* List of addresses

## Outputs

* CSV file for all transactions within the block range
  * 1 file per address
* 1 CSV file for each address with 2 balance snapshots @ start and end block
  * Liquid balance
  * CDP position
  * Hard position
  * Liquid deposit in earn

## Usage

```
go run .
```
