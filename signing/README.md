# Signing Package

This package provides functionality to sign and broadcast transactions to a Cosmos node.

## Key Components:
- **MsgRequest**:
  - Represents a request to sign a transaction, including transaction details such as messages, gas limit, fees, and memo.

- **MsgResponse**: 
  - Represents the response after a transaction has been signed and broadcast, including transaction details and any errors.

- **EncodingConfig Interface**:
  - Abstract the encoding and decoding of transactions. This allows the Signer to be reused across different 
  Cosmos SDK chains, not just Kava.

- **Signer Struct**:
  - Represents the configuration and clients needed to sign and broadcast transactions, such as chain ID, 
  encoding configuration, authentication client, transaction client, private key, in-flight transaction limit, 
  logger, and account status.

## Core Functions:

- **NewSigner**:
  - Creates a new Signer instance with provided parameters and performs validation (e.g., inflightTxLimit 
  must be greater than zero).

- **GetAccountError, setAccountError, clearAccountError**:
  - Methods to get, set, and clear account-related errors.

- **pollAccountState**:
  - Periodically polls the account state and retries on errors, sending updates via a channel.

- **getAccountState**:
  - Queries the account state using the private key and unpacks it.
  
- **Run**:
  - Main function to start the signer, processing incoming requests and broadcasting transactions:
    - Polls the account state in a separate goroutine. 
    - Continuously processes signing requests and attempts to broadcast them until successfully placed into 
    the node's mempool. 
    - Handles various errors and retries broadcasting as needed. 
    - Manages sequence errors and adjusts the transaction sequence to ensure transactions are broadcast correctly.

- **Sign**:
  - Signs a transaction using the provided private key and signer data, returns the signed transaction and its raw bytes.
  
- **GetAccAddress**:
  - Returns the account address for a given private key.

## Broadcast Loop Logic:
  - The broadcast loop in the Run method is designed to ensure that transactions are placed into the node's mempool, 
  handling different types of errors and retrying if necessary:
    - txOK: 
      - Transaction successfully broadcast and in the mempool. 
    - txFailed: 
      - Transaction failed and is not recoverable. 
    - txRetry: 
      - Transaction failed but can be retried. 
    - txResetSequence: 
      - Transaction sequence is invalid and needs to be reset.


## Flow Summary:
- Create a Signer instance with necessary configurations. 
- Periodically poll the account state and handle errors. 
- Process incoming transaction signing requests. 
- Attempt to broadcast each transaction, handling errors and retrying if needed. 
- Manage transaction sequences to ensure proper placement into the node's mempool. 
- Respond with the result of each transaction attempt.

## Testing:
- Run a local chain with kvtool
  - `$ kvtool testnet boostrap`
- cd go run main.go